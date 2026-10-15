package listen

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var issueIDPattern = regexp.MustCompile(`^f-[0-9a-f]{8,32}$`)

type NoteJSON struct {
	ID        string `json:"id"`
	IssueID   string `json:"issue_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type IssueAPI interface {
	Note(ctx context.Context, repo, issueID, body string) (NoteJSON, error)
	ShowJSON(ctx context.Context, repo, issueID string) ([]byte, error)
}

type APIError struct {
	Status int
	Msg    string
}

func (e *APIError) Error() string {
	if e.Msg != "" {
		return e.Msg
	}
	return http.StatusText(e.Status)
}

func (s *Server) handleNote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !authorizeListen(w, r, s.token) {
		return
	}
	if s.issues == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var in struct {
		Repo string `json:"repo"`
		ID   string `json:"id"`
		Body string `json:"body"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	repo, err := validateRepo(in.Repo)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	if !issueIDPattern.MatchString(in.ID) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := validateNoteBody(in.Body); err != nil {
		writeAPIError(w, err)
		return
	}

	note, err := s.issues.Note(r.Context(), repo, in.ID, strings.TrimSpace(in.Body))
	if err != nil {
		writeAPIError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(note)
}

func (s *Server) handleIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !authorizeListen(w, r, s.token) {
		return
	}
	if s.issues == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	repo, err := validateRepo(r.URL.Query().Get("repo"))
	if err != nil {
		writeAPIError(w, err)
		return
	}
	id := r.URL.Query().Get("id")
	if !issueIDPattern.MatchString(id) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	raw, err := s.issues.ShowJSON(r.Context(), repo, id)
	if err != nil {
		writeAPIError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(raw)
}

func authorizeListen(w http.ResponseWriter, r *http.Request, token string) bool {
	if !clientLoopback(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	if !bearerOK(r, token) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func validateRepo(repo string) (string, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return "", &APIError{Status: http.StatusBadRequest, Msg: "bad request"}
	}
	if !filepath.IsAbs(repo) && !strings.HasPrefix(repo, "/") {
		return "", &APIError{Status: http.StatusBadRequest, Msg: "bad request"}
	}
	return filepath.Clean(repo), nil
}

func validateNoteBody(body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return &APIError{Status: http.StatusBadRequest, Msg: "bad request"}
	}
	if utf8.RuneCountInString(body) > 8000 {
		return &APIError{Status: http.StatusBadRequest, Msg: "bad request"}
	}
	return nil
}

func writeAPIError(w http.ResponseWriter, err error) {
	var ae *APIError
	if errors.As(err, &ae) {
		msg := ae.Msg
		if msg == "" {
			msg = http.StatusText(ae.Status)
		}
		http.Error(w, msg, ae.Status)
		return
	}
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
