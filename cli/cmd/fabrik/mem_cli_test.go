package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemSearchGetSessionCLI(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)

	out, err := run(t, "mem", "session", "start", "--json")
	require.NoError(t, err)
	var start map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &start))
	assert.Equal(t, true, start["created"])

	_, err = run(t, "mem", "save", "--title", "Alpha", "--body", "unique zebra token", "--json")
	require.NoError(t, err)

	out, err = run(t, "mem", "search", "zebra", "--json")
	require.NoError(t, err)
	var hits []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &hits))
	require.Len(t, hits, 1)
	id, _ := hits[0]["id"].(string)
	require.NotEmpty(t, id)

	out, err = run(t, "mem", "get", id, "--json")
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, "Alpha", got["title"])

	_, err = run(t, "mem", "session", "summary", "--goal", "done")
	require.NoError(t, err)

	out, err = run(t, "mem", "session", "end", "--json")
	require.NoError(t, err)
	var end map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &end))
	assert.NotNil(t, end["ended_at"])
}

func TestMemSaveAndContextCLI(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FABRIK_MEMORY_DIR", dir)
	out, err := run(t, "mem", "save", "--title", "t", "--body", "b", "--kind", "decision", "--json")
	require.NoError(t, err)
	var o map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &o))
	assert.Equal(t, "decision", o["kind"])

	out, err = run(t, "mem", "context", "--json")
	require.NoError(t, err)
	var ctx map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &ctx))
	obs, ok := ctx["observations"].([]any)
	require.True(t, ok)
	assert.Len(t, obs, 1)
}
