package listen

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

var ErrAlreadyRunning = errors.New("fabrik: listen already running")

type Server struct {
	token      string
	eventsURL  string
	notifyURL  string
	configPath string
	hub        *hub
	issues     IssueAPI
	http       *http.Server
	mu         sync.Mutex
	ln         net.Listener
}

func NewServer(bindAddr, configPath string, issues IssueAPI) (*Server, error) {
	if _, err := os.Stat(configPath); err == nil {
		if existingListenBusy(configPath) {
			return nil, ErrAlreadyRunning
		}
		_ = os.Remove(configPath)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	host, port, err := net.SplitHostPort(bindAddr)
	if err != nil {
		return nil, err
	}
	if !isLoopbackHost(host) {
		return nil, fmt.Errorf("fabrik listen: remote bind refused (use 127.0.0.1)")
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, err
	}
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		ln.Close()
		return nil, fmt.Errorf("fabrik listen: unexpected listener address")
	}
	if !tcpAddr.IP.IsLoopback() {
		ln.Close()
		return nil, fmt.Errorf("fabrik listen: remote bind refused")
	}

	token, err := randomToken()
	if err != nil {
		ln.Close()
		return nil, err
	}

	base := fmt.Sprintf("http://%s", ln.Addr().String())
	s := &Server{
		token:      token,
		eventsURL:  base + "/events",
		notifyURL:  base + "/notify",
		configPath: configPath,
		hub:        newHub(),
		issues:     issues,
		ln:         ln,
	}
	s.http = &http.Server{Handler: s.routes()}

	if err := s.writeConfig(); err != nil {
		ln.Close()
		return nil, err
	}
	return s, nil
}

func (s *Server) Config() Config {
	return Config{URL: s.eventsURL, Token: s.token}
}

func (s *Server) Serve() error {
	return s.http.Serve(s.ln)
}

func (s *Server) Close() error {
	var err error
	s.mu.Lock()
	if s.http != nil {
		err = s.http.Close()
	}
	s.mu.Unlock()
	_ = os.Remove(s.configPath)
	return err
}

func (s *Server) Notify(e Event) {
	s.hub.broadcast(e)
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/events", s.handleEvents)
	mux.HandleFunc("/notify", s.handleNotify)
	mux.HandleFunc("/note", s.handleNote)
	mux.HandleFunc("/issue", s.handleIssue)
	return mux
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !authorizeListen(w, r, s.token) {
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	_, _ = io.WriteString(w, ": connected\n\n")
	flusher.Flush()

	ch := s.hub.subscribe()
	defer s.hub.unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-ch:
			_, _ = fmt.Fprintf(w, "data: %s\n\n", string(e.JSONLine()))
			flusher.Flush()
		}
	}
}

func (s *Server) handleNotify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !authorizeListen(w, r, s.token) {
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var e Event
	if err := json.Unmarshal(body, &e); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if e.At == "" {
		e.At = NewEvent(e.Topic, e.Repo).At
	}
	s.hub.broadcast(e)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) writeConfig() error {
	if err := os.MkdirAll(dirOf(s.configPath), 0o755); err != nil {
		return err
	}
	cfg := s.Config()
	cfg.PID = os.Getpid()
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(s.configPath, b, 0o600)
}

func dirOf(path string) string {
	i := strings.LastIndex(path, string(os.PathSeparator))
	if i < 0 {
		return "."
	}
	return path[:i]
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func clientLoopback(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func existingListenBusy(configPath string) bool {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return false
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil || cfg.URL == "" {
		return false
	}
	if cfg.PID > 0 && processAlive(cfg.PID) {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return false
	}
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func bearerOK(r *http.Request, token string) bool {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	return strings.TrimSpace(h[len(prefix):]) == token
}
