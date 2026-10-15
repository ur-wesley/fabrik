//go:build cgo

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

func TestInitRepoStoresCreatesDoltAndGraph(t *testing.T) {
	if os.Getenv("CGO_ENABLED") == "0" {
		t.Skip("requires CGO_ENABLED=1")
	}
	repo := t.TempDir()
	res, err := InitRepoStores(context.Background(), repo, exec.OSRunner{})
	require.NoError(t, err)
	assert.True(t, res.Created)
	assert.DirExists(t, res.DoltPath)
	assert.FileExists(t, res.GraphPath)
	assert.FileExists(t, filepath.Join(repo, ".gitignore"))
	assert.Contains(t, string(mustRead(filepath.Join(repo, ".gitignore"))), ".fabrik/dolt/")
}

func mustRead(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

func TestInitRepoStoresIdempotentCreatedFlag(t *testing.T) {
	if os.Getenv("CGO_ENABLED") == "0" {
		t.Skip("requires CGO_ENABLED=1")
	}
	repo := t.TempDir()
	_, err := InitRepoStores(context.Background(), repo, exec.OSRunner{})
	require.NoError(t, err)
	res, err := InitRepoStores(context.Background(), repo, exec.OSRunner{})
	require.NoError(t, err)
	assert.False(t, res.Created)
	_ = store.GraphDBPath(repo)
}
