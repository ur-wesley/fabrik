package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestVersionCmd(t *testing.T) {
	out, err := run(t, "version")
	require.NoError(t, err)
	assert.Contains(t, out, "fabrik 1.0.0")
	assert.Contains(t, out, "beads")
	assert.Contains(t, out, "engram")
}

func TestCheckJSONShape(t *testing.T) {
	out, err := run(t, "check", "--json")
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &m))
	for _, k := range []string{"bd", "engram", "graphify", "bun", "pi", "uv", "ok"} {
		assert.Contains(t, m, k)
	}
}

func TestInitDryRunViaCLI(t *testing.T) {
	repo := t.TempDir()
	out, err := run(t, "init", repo, "--dry-run", "--yes", "--skip-checks")
	_ = out
	// bd missing on PATH (or present) — either error mentions bd or succeeds
	if err != nil {
		assert.Contains(t, err.Error(), "bd")
		return
	}
	entries, rerr := os.ReadDir(repo)
	require.NoError(t, rerr)
	assert.Empty(t, entries)
	_ = filepath.Join
}
