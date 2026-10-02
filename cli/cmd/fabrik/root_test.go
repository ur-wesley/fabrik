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

func TestResetCmdHelp(t *testing.T) {
	out, err := run(t, "reset", "--help")
	require.NoError(t, err)
	assert.Contains(t, out, "Reset and remove Fabrik configuration")
	assert.Contains(t, out, "--beads")
	assert.Contains(t, out, "--dry-run")
}

func TestResetDryRunViaCLI(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".fabrik"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".fabrik", "config.yaml"), []byte("version: 1\n"), 0o644))

	out, err := run(t, "reset", repo, "--dry-run", "--yes")
	require.NoError(t, err)
	assert.Contains(t, out, "would delete .fabrik")

	// Verify dry-run didn't delete the directory
	assert.DirExists(t, filepath.Join(repo, ".fabrik"))
}

