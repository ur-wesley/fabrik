package installer

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
)

func TestSetupCreatesMemoryAndInitsRepo(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("FABRIK_MEMORY_DIR", filepath.Join(home, ".fabrik"))

	var out bytes.Buffer
	called := ""
	d := Deps{
		Exec: &exec.Fake{},
		Out:  &out,
		InitRepo: func(path string) error {
			called = path
			return nil
		},
		RegisterMCP: func(apps []string, repo string) error { return nil },
	}
	require.NoError(t, Setup(context.Background(), Config{Yes: true, RepoPath: repo}, d))
	assert.Contains(t, out.String(), "memory ")
	assert.Contains(t, out.String(), "mcp registered")
	abs, err := filepath.Abs(repo)
	require.NoError(t, err)
	assert.Equal(t, abs, called)
	assert.FileExists(t, filepath.Join(home, ".fabrik", "memory.db"))
}

func TestSetupDryRunTouchesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	var out bytes.Buffer
	d := Deps{Exec: &exec.Fake{}, Out: &out}
	require.NoError(t, Setup(context.Background(), Config{Yes: true, DryRun: true}, d))
	assert.Contains(t, out.String(), "dry-run")
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestSetupWarnsNotGitRepo(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("FABRIK_MEMORY_DIR", filepath.Join(home, ".fabrik"))
	var out bytes.Buffer
	d := Deps{
		Exec: &exec.Fake{},
		Out:  &out,
		InitRepo: func(string) error { return nil },
		RegisterMCP: func([]string, string) error { return nil },
	}
	require.NoError(t, Setup(context.Background(), Config{Yes: true, RepoPath: repo}, d))
	assert.Contains(t, out.String(), "Not a git repository")
}
