package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

func TestRepoStartDirUsesFlag(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".fabrik", "dolt"), 0o755))
	repoDirFlag = repo
	defer func() { repoDirFlag = "" }()

	root, ok := store.RepoRoot(repoStartDir())
	require.True(t, ok)
	require.Equal(t, repo, root)
}
