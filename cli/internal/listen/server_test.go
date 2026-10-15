package listen

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriberGetsEventAfterNotify(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "listen.json")

	srv, err := NewServer("127.0.0.1:0", cfgPath, nil)
	require.NoError(t, err)
	defer srv.Close()

	go func() { _ = srv.Serve() }()

	cfg := srv.Config()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, cfg.URL, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+cfg.Token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	done := make(chan Event, 1)
	go func() {
		e, err := readOneSSEData(resp.Body)
		if err != nil {
			return
		}
		done <- e
	}()

	time.Sleep(50 * time.Millisecond)
	srv.Notify(NewEvent("memory", ""))

	select {
	case e := <-done:
		assert.Equal(t, "memory", e.Topic)
		assert.Equal(t, "", e.Repo)
		assert.NotEmpty(t, e.At)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SSE event")
	}
}

func TestNotifyHTTP(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "listen.json")

	srv, err := NewServer("127.0.0.1:0", cfgPath, nil)
	require.NoError(t, err)
	defer srv.Close()

	go func() { _ = srv.Serve() }()

	cfg := srv.Config()
	req, err := http.NewRequest(http.MethodGet, cfg.URL, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+cfg.Token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	done := make(chan Event, 1)
	go func() {
		e, err := readOneSSEData(resp.Body)
		if err != nil {
			return
		}
		done <- e
	}()

	time.Sleep(50 * time.Millisecond)

	notifyURL := strings.TrimSuffix(cfg.URL, "/events") + "/notify"
	post, err := http.NewRequest(http.MethodPost, notifyURL, strings.NewReader(`{"topic":"issues","repo":"/repo","at":"2026-10-11T18:00:00Z"}`))
	require.NoError(t, err)
	post.Header.Set("Authorization", "Bearer "+cfg.Token)
	post.Header.Set("Content-Type", "application/json")
	pr, err := http.DefaultClient.Do(post)
	require.NoError(t, err)
	_ = pr.Body.Close()
	require.Equal(t, http.StatusNoContent, pr.StatusCode)

	select {
	case e := <-done:
		assert.Equal(t, "issues", e.Topic)
		assert.Equal(t, "/repo", e.Repo)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SSE event")
	}
}

func TestSecondInstanceRejected(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "listen.json")

	srv, err := NewServer("127.0.0.1:0", cfgPath, nil)
	require.NoError(t, err)
	defer srv.Close()

	_, err = NewServer("127.0.0.1:0", cfgPath, nil)
	require.ErrorIs(t, err, ErrAlreadyRunning)
}

type stubIssues struct {
	note func(ctx context.Context, repo, id, body string) (NoteJSON, error)
	show func(ctx context.Context, repo, id string) ([]byte, error)
}

func (s stubIssues) Note(ctx context.Context, repo, id, body string) (NoteJSON, error) {
	if s.note != nil {
		return s.note(ctx, repo, id, body)
	}
	return NoteJSON{}, nil
}

func (s stubIssues) ShowJSON(ctx context.Context, repo, id string) ([]byte, error) {
	if s.show != nil {
		return s.show(ctx, repo, id)
	}
	return nil, nil
}

func TestNoteHTTPUnauthorized(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "listen.json")
	srv, err := NewServer("127.0.0.1:0", cfgPath, stubIssues{})
	require.NoError(t, err)
	defer srv.Close()
	go func() { _ = srv.Serve() }()

	noteURL := strings.TrimSuffix(srv.Config().URL, "/events") + "/note"
	req, err := http.NewRequest(http.MethodPost, noteURL, strings.NewReader(`{"repo":"/tmp/r","id":"f-12345678","body":"hi"}`))
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestNoteHTTPEmptyBody(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "listen.json")
	srv, err := NewServer("127.0.0.1:0", cfgPath, stubIssues{})
	require.NoError(t, err)
	defer srv.Close()
	go func() { _ = srv.Serve() }()

	cfg := srv.Config()
	noteURL := strings.TrimSuffix(cfg.URL, "/events") + "/note"
	req, err := http.NewRequest(http.MethodPost, noteURL, strings.NewReader(`{"repo":"/tmp/r","id":"f-12345678","body":"   "}`))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestNoteHTTPReturnsJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "listen.json")
	api := stubIssues{
		note: func(_ context.Context, repo, id, body string) (NoteJSON, error) {
			return NoteJSON{
				ID: "n-abcdef01", IssueID: id, Body: body, CreatedAt: "2026-10-11T20:30:00Z",
			}, nil
		},
	}
	srv, err := NewServer("127.0.0.1:0", cfgPath, api)
	require.NoError(t, err)
	defer srv.Close()
	go func() { _ = srv.Serve() }()

	cfg := srv.Config()
	noteURL := strings.TrimSuffix(cfg.URL, "/events") + "/note"
	req, err := http.NewRequest(http.MethodPost, noteURL, strings.NewReader(`{"repo":"/tmp/r","id":"f-12345678","body":"hello"}`))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var out NoteJSON
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	assert.Equal(t, "n-abcdef01", out.ID)
	assert.Equal(t, "f-12345678", out.IssueID)
	assert.Equal(t, "hello", out.Body)
}

func TestIssueHTTPReturnsShowJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "listen.json")
	payload := []byte(`{"id":"f-12345678","title":"t","notes":[{"id":"n-1","body":"x","created_at":"2026-10-11T18:00:00Z"}]}`)
	api := stubIssues{
		show: func(_ context.Context, repo, id string) ([]byte, error) {
			return payload, nil
		},
	}
	srv, err := NewServer("127.0.0.1:0", cfgPath, api)
	require.NoError(t, err)
	defer srv.Close()
	go func() { _ = srv.Serve() }()

	cfg := srv.Config()
	issueURL := strings.TrimSuffix(cfg.URL, "/events") + "/issue?repo=/tmp/r&id=f-12345678"
	req, err := http.NewRequest(http.MethodGet, issueURL, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.JSONEq(t, string(payload), string(body))
}

func readOneSSEData(r io.Reader) (Event, error) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return Event{}, err
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			var e Event
			if err := json.Unmarshal([]byte(line[6:]), &e); err != nil {
				return Event{}, err
			}
			return e, nil
		}
	}
}
