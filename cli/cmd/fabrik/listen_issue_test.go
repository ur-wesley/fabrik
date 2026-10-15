package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/listen"
)

func TestIssueNote(t *testing.T) {
	skipWithoutCGO(t)
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755))
	t.Chdir(dir)

	out, err := run(t, "issue", "create", "Note target", "--json")
	require.NoError(t, err)
	var iss map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &iss))
	id := iss["id"].(string)

	out, err = run(t, "issue", "note", id, "progress update", "--json")
	require.NoError(t, err)
	var note map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &note))
	assert.Equal(t, id, note["issue_id"])
	assert.Equal(t, "progress update", note["body"])

	out, err = run(t, "issue", "show", id, "--json")
	require.NoError(t, err)
	var shown map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &shown))
	notes, ok := shown["notes"].([]any)
	require.True(t, ok)
	require.Len(t, notes, 1)
	first, _ := notes[0].(map[string]any)
	assert.Equal(t, "progress update", first["body"])
}

func TestListenNoteHTTPNotifiesSSE(t *testing.T) {
	skipWithoutCGO(t)
	dir := t.TempDir()
	abs, err := filepath.Abs(dir)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(abs, ".fabrik", "dolt"), 0o755))
	t.Chdir(abs)

	out, err := run(t, "issue", "create", "HTTP note", "--json")
	require.NoError(t, err)
	var iss map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &iss))
	issueID := iss["id"].(string)

	memDir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", memDir)
	cfgPath := filepath.Join(memDir, "listen.json")
	srv, err := listen.NewServer("127.0.0.1:0", cfgPath, listenIssueAPI{})
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

	done := make(chan listen.Event, 1)
	go func() {
		br := bufio.NewReader(resp.Body)
		for {
			e, err := readSSEData(br)
			if err != nil {
				return
			}
			if e.Op == "note" && e.ID == issueID {
				done <- e
				return
			}
		}
	}()

	time.Sleep(50 * time.Millisecond)

	noteURL := strings.TrimSuffix(cfg.URL, "/events") + "/note"
	m, err := json.Marshal(map[string]string{"repo": abs, "id": issueID, "body": "via http"})
	require.NoError(t, err)
	post, err := http.NewRequest(http.MethodPost, noteURL, bytes.NewReader(m))
	require.NoError(t, err)
	post.Header.Set("Authorization", "Bearer "+cfg.Token)
	post.Header.Set("Content-Type", "application/json")
	pr, err := http.DefaultClient.Do(post)
	require.NoError(t, err)
	defer pr.Body.Close()
	require.Equal(t, http.StatusOK, pr.StatusCode)

	select {
	case e := <-done:
		assert.Equal(t, "issues", e.Topic)
		assert.Equal(t, abs, e.Repo)
		assert.Equal(t, issueID, e.ID)
		assert.Equal(t, "note", e.Op)
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for SSE note event")
	}
}

func readSSEData(br *bufio.Reader) (listen.Event, error) {
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return listen.Event{}, err
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			var e listen.Event
			if err := json.Unmarshal([]byte(line[6:]), &e); err != nil {
				return listen.Event{}, err
			}
			return e, nil
		}
	}
}
