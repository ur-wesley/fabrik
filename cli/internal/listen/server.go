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
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ur-wesley/fabrik/cli/internal/fileutil"
)

var ErrAlreadyRunning = errors.New("fabrik: listen already running")

// activeConfigs tracks live servers in this process by config path, guarded
// by activeMu. NewServer holds activeMu for its whole check-and-register
// sequence so two in-process starters cannot both claim the same path, and
// Close unregisters. Cross-process races are handled by the atomic
// temp+rename writeConfig and the token-gated removeOwnConfig.
var (
	activeMu      sync.Mutex
	activeConfigs = make(map[string]*Server)
)

type Server struct {
	token      string
	eventsURL  string
	notifyURL  string
	configPath string
	lockPath   string
	lockFile   *os.File
	hub        *hub
	issues     IssueAPI
	http       *http.Server
	mu         sync.Mutex
	ln         net.Listener
}

// NewServer starts a loopback-only listener and publishes its config atomically.
//
// Dual-:0 race: two starters binding ":0" each get a distinct ephemeral port,
// so the TCP bind cannot serialize them — without extra coordination both
// would listen and the loser's writeConfig would orphan the winner's config.
// claimListenLock closes that hole with a best-effort O_EXCL sidecar lock
// (<configPath>.lock) held open for the server's lifetime; stale locks from
// dead owners are removed and re-claimed.
func NewServer(bindAddr, configPath string, issues IssueAPI) (*Server, error) {
	activeMu.Lock()
	defer activeMu.Unlock()
	if _, ok := activeConfigs[configPath]; ok {
		return nil, ErrAlreadyRunning
	}
	lockFile, err := claimListenLock(configPath)
	if err != nil {
		return nil, err
	}
	releaseOnErr := func() { releaseListenLock(lockFile, lockPathFor(configPath)) }
	// No Remove here: a stale config is replaced atomically by writeConfig
	// (temp file + rename). Removing first would open a TOCTOU window where
	// a concurrent starter/closer observes a missing config.
	if _, err := os.Stat(configPath); err == nil {
		if existingListenBusy(configPath) {
			releaseOnErr()
			return nil, ErrAlreadyRunning
		}
	} else if !os.IsNotExist(err) {
		releaseOnErr()
		return nil, err
	}

	host, port, err := net.SplitHostPort(bindAddr)
	if err != nil {
		releaseOnErr()
		return nil, err
	}
	if !isLoopbackHost(host) {
		releaseOnErr()
		return nil, fmt.Errorf("fabrik listen: remote bind refused (use 127.0.0.1)")
	}

	ln, err := net.Listen("tcp", net.JoinHostPort(host, port))
	if err != nil {
		releaseOnErr()
		return nil, err
	}
	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		ln.Close()
		releaseOnErr()
		return nil, fmt.Errorf("fabrik listen: unexpected listener address")
	}
	if !tcpAddr.IP.IsLoopback() {
		ln.Close()
		releaseOnErr()
		return nil, fmt.Errorf("fabrik listen: remote bind refused")
	}

	token, err := randomToken()
	if err != nil {
		ln.Close()
		releaseOnErr()
		return nil, err
	}

	base := fmt.Sprintf("http://%s", ln.Addr().String())
	s := &Server{
		token:      token,
		eventsURL:  base + "/events",
		notifyURL:  base + "/notify",
		configPath: configPath,
		lockPath:   lockPathFor(configPath),
		lockFile:   lockFile,
		hub:        newHub(),
		issues:     issues,
		ln:         ln,
	}
	s.http = &http.Server{Handler: s.routes()}

	if err := s.writeConfig(); err != nil {
		ln.Close()
		releaseOnErr()
		return nil, err
	}
	activeConfigs[configPath] = s
	return s, nil
}

func (s *Server) Config() Config {
	return Config{URL: s.eventsURL, Token: s.token}
}

func (s *Server) Serve() error {
	return s.http.Serve(s.ln)
}

func (s *Server) Close() error {
	activeMu.Lock()
	if cur, ok := activeConfigs[s.configPath]; ok && cur == s {
		delete(activeConfigs, s.configPath)
	}
	activeMu.Unlock()
	var err error
	s.mu.Lock()
	if s.http != nil {
		err = s.http.Close()
	}
	s.mu.Unlock()
	s.removeOwnConfig()
	releaseListenLock(s.lockFile, s.lockPath)
	return err
}

// removeOwnConfig deletes the config file only if it still belongs to this
// server (token match) and was written by this process. A stale Close (e.g.
// deferred after a successor took over the path) must not delete the
// successor's config.
func (s *Server) removeOwnConfig() {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	if cfg.Token == "" || cfg.Token != s.token {
		return
	}
	if cfg.PID != 0 && cfg.PID != os.Getpid() {
		return
	}
	_ = os.Remove(s.configPath)
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
	if err := os.MkdirAll(dirOf(s.configPath), fileutil.DirPerm); err != nil {
		return err
	}
	cfg := s.Config()
	cfg.PID = os.Getpid()
	b, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	// Atomic replacement: temp file created with O_EXCL (CreateTemp) in the
	// same directory, then renamed over the target. Readers always see a
	// complete config; no Stat/Remove/Write TOCTOU window.
	tmp, err := os.CreateTemp(dirOf(s.configPath), ".listen-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(fileutil.SecretPerm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, s.configPath)
}

func dirOf(path string) string {
	// filepath.Clean first so mixed separators ("a/b\c") resolve to the real
	// parent on every OS; filepath.Dir alone only splits on the native one.
	return filepath.Dir(filepath.Clean(path))
}

// lockPathFor returns the sidecar claim file identifying the live owner of
// configPath (see NewServer).
func lockPathFor(configPath string) string {
	return configPath + ".lock"
}

// claimListenLock atomically claims the config path for this process via an
// O_EXCL-created lock file holding our PID, kept open by the caller for the
// server's lifetime. A pre-existing lock owned by a live process (or serving
// URL) yields ErrAlreadyRunning; a stale lock (dead owner, nothing serving)
// is removed and claimed once. Best-effort caveat: a recycled PID matching a
// stale lock fails closed with ErrAlreadyRunning until that PID exits or the
// .lock file is removed manually.
func claimListenLock(configPath string) (*os.File, error) {
	lockPath := lockPathFor(configPath)
	if err := os.MkdirAll(dirOf(lockPath), fileutil.DirPerm); err != nil {
		return nil, err
	}
	if f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileutil.SecretPerm); err == nil {
		_, _ = fmt.Fprintf(f, "%d", os.Getpid())
		return f, nil
	} else if !os.IsExist(err) {
		return nil, err
	}
	if pid := readLockPID(lockPath); pid > 0 && !processAlive(pid) && !existingListenBusy(configPath) {
		if err := os.Remove(lockPath); err == nil {
			if f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileutil.SecretPerm); err == nil {
				_, _ = fmt.Fprintf(f, "%d", os.Getpid())
				return f, nil
			}
		}
	}
	return nil, ErrAlreadyRunning
}

// releaseListenLock drops our claim. The PID check keeps a stale Close from
// deleting a successor's lock after a stale-lock takeover.
func releaseListenLock(f *os.File, lockPath string) {
	if f == nil {
		return
	}
	_ = f.Close()
	if lockPath == "" {
		return
	}
	if pid := readLockPID(lockPath); pid == os.Getpid() {
		_ = os.Remove(lockPath)
	}
}

func readLockPID(lockPath string) int {
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
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
	// The HTTP probe is authoritative: only the live owner of this config
	// answers on its loopback URL. A bare processAlive(PID) check is NOT
	// sufficient — PIDs are recycled, so Kill-0/FindProcess can succeed for
	// an unrelated successor process (false "busy").
	if probeListenURL(cfg) {
		return true
	}
	// HTTP unreachable but PID alive: the owner may be in its startup window
	// (config written, Serve not yet accepting). Re-probe once after a short
	// delay; a recycled PID never serves this URL, so a second miss means
	// the config is stale.
	if cfg.PID > 0 && processAlive(cfg.PID) {
		time.Sleep(150 * time.Millisecond)
		return probeListenURL(cfg)
	}
	return false
}

func probeListenURL(cfg Config) bool {
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

func bearerOK(r *http.Request, token string) bool {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, prefix) {
		return false
	}
	return strings.TrimSpace(h[len(prefix):]) == token
}
