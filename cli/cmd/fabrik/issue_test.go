package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueNoStore(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	_, err := run(t, "issue", "list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no store")
}

func TestIssueListEmptyJSON(t *testing.T) {
	skipWithoutCGO(t)
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755))
	t.Chdir(dir)
	out, err := run(t, "issue", "list", "--json")
	require.NoError(t, err)
	var list []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &list))
	assert.Empty(t, list)
}

func TestIssueCreateShow(t *testing.T) {
	skipWithoutCGO(t)
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755))
	t.Chdir(dir)
	out, err := run(t, "issue", "create", "Document claim", "--acceptance", "Docs name exit codes", "--json")
	require.NoError(t, err)
	var iss map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &iss))
	id, _ := iss["id"].(string)
	require.Regexp(t, `^f-[0-9a-f]{8}$`, id)

	out, err = run(t, "issue", "show", id, "--json")
	require.NoError(t, err)
	var shown map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &shown))
	assert.Equal(t, "Document claim", shown["title"])
}

func TestIssueReadyClaimClose(t *testing.T) {
	skipWithoutCGO(t)
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".fabrik", "dolt"), 0o755))
	t.Chdir(dir)

	out, err := run(t, "issue", "create", "Blocker", "--json")
	require.NoError(t, err)
	var blocker map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &blocker))
	blockerID := blocker["id"].(string)

	out, err = run(t, "issue", "create", "Work", "--json")
	require.NoError(t, err)
	var work map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &work))
	workID := work["id"].(string)

	_, err = run(t, "issue", "dep", workID, blockerID)
	require.NoError(t, err)

	out, err = run(t, "issue", "ready", "--json")
	require.NoError(t, err)
	var ready []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &ready))
	assert.Len(t, ready, 1)
	assert.Equal(t, blockerID, ready[0]["id"])

	_, err = run(t, "issue", "claim", blockerID, "--by", "tester")
	require.NoError(t, err)

	_, err = run(t, "issue", "close", blockerID, "--reason", "ok")
	require.NoError(t, err)

	out, err = run(t, "issue", "show", workID, "--json")
	require.NoError(t, err)
	var shown map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &shown))
	assert.Equal(t, "open", shown["status"])
	time.Sleep(400 * time.Millisecond)
}
