package repo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGitRepo(t *testing.T) {
	root := t.TempDir()
	assert.False(t, IsGitRepo(root))

	requireGitDir(t, filepath.Join(root, ".git"))
	assert.True(t, IsGitRepo(root))

	sub := filepath.Join(root, "pkg", "nested")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	assert.True(t, IsGitRepo(sub))
}

func requireGitDir(t *testing.T, p string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(p, 0o755))
}
