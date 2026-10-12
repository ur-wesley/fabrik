package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/version"
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
	assert.Contains(t, out, "fabrik "+version.Get())
}

func TestVersionFlag(t *testing.T) {
	want := "fabrik " + version.Get()
	out, err := run(t, "--version")
	require.NoError(t, err)
	assert.Contains(t, out, want)
	out, err = run(t, "-v")
	require.NoError(t, err)
	assert.Contains(t, out, want)
}

func TestCheckJSONShape(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	out, err := run(t, "check", "--json")
	require.Error(t, err)
	idx := strings.Index(out, "{")
	require.GreaterOrEqual(t, idx, 0)
	var m map[string]any
	require.NoError(t, json.NewDecoder(strings.NewReader(out[idx:])).Decode(&m))
	for _, k := range []string{"dolt", "dolt_path", "graph", "graph_path", "schema_dolt", "issues", "graph_files"} {
		assert.Contains(t, m, k)
	}
}

func TestInitDryRunViaCLI(t *testing.T) {
	repo := t.TempDir()
	out, err := run(t, "init", repo, "--dry-run", "--yes", "--skip-checks")
	require.NoError(t, err)
	assert.Contains(t, out, "dry-run")
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
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".fabrik", "dolt"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".fabrik", "config.yaml"), []byte("version: 1\n"), 0o644))

	out, err := run(t, "reset", repo, "--dry-run", "--yes")
	require.NoError(t, err)
	assert.Contains(t, out, "would delete")

	// Verify dry-run didn't delete the directory
	assert.DirExists(t, filepath.Join(repo, ".fabrik"))
}

