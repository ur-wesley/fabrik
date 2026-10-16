package initer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

// cli-x3a: init creates the SQLite issue store (no CGO-gated Dolt needed).
// cli-a35: StoreResult carries the neutral IssuesPath; DoltPath stays as a
// deprecated alias with the same value.
func TestInitRepoStoresCreatesIssuesAndGraph(t *testing.T) {
	repo := t.TempDir()
	res, err := InitRepoStores(context.Background(), repo, exec.OSRunner{})
	require.NoError(t, err)
	assert.True(t, res.Created)
	assert.Equal(t, store.IssuesDBPath(repo), res.IssuesPath)
	assert.Equal(t, res.IssuesPath, res.DoltPath)
	assert.FileExists(t, res.IssuesPath)
	assert.FileExists(t, res.GraphPath)
	assert.FileExists(t, filepath.Join(repo, ".gitignore"))
	assert.Contains(t, string(mustRead(filepath.Join(repo, ".gitignore"))), ".fabrik/issues.db")
}

func mustRead(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

func TestInitRepoStoresIdempotentCreatedFlag(t *testing.T) {
	repo := t.TempDir()
	_, err := InitRepoStores(context.Background(), repo, exec.OSRunner{})
	require.NoError(t, err)
	res, err := InitRepoStores(context.Background(), repo, exec.OSRunner{})
	require.NoError(t, err)
	assert.False(t, res.Created)
	_ = store.GraphDBPath(repo)
}

func TestInitRepoStoresStripsLegacyExportIgnore(t *testing.T) {
	repo := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(repo, ".gitignore"),
		[]byte("node_modules\n.fabrik/export/\n"), 0o644))
	_, err := InitRepoStores(context.Background(), repo, exec.OSRunner{})
	require.NoError(t, err)
	body := string(mustRead(filepath.Join(repo, ".gitignore")))
	assert.NotContains(t, body, ".fabrik/export/")
	assert.Contains(t, body, "node_modules")
	assert.Contains(t, body, ".fabrik/issues.db")
}
