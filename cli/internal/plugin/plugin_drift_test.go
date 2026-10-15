package plugin

// Drift guard between the embedded mirror (cli/internal/plugin/fabrik)
// and the repo source of truth (plugins/fabrik). go:embed rejects ".."
// patterns, so the mirror is a checked-in copy. Re-copy on change:
//
//	cp -rf plugins/fabrik cli/internal/plugin/fabrik

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoSource locates <repo>/plugins/fabrik relative to this package dir
// (cli/internal/plugin). Test-only: os paths may use "..".
func repoSource(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "..", "..", "plugins", "fabrik"))
	require.NoError(t, err)
	require.DirExists(t, abs, "repo plugin source missing — run from the cli module")
	return abs
}

func TestMirrorMatchesRepoSource(t *testing.T) {
	src := repoSource(t)
	var repoFiles []string
	require.NoError(t, filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		require.NoError(t, err)
		if !d.IsDir() {
			rel, err := filepath.Rel(src, p)
			require.NoError(t, err)
			repoFiles = append(repoFiles, filepath.ToSlash(rel))
		}
		return nil
	}))

	embedded := Files()
	embeddedSet := make(map[string]bool, len(embedded))
	for _, f := range embedded {
		embeddedSet[filepath.ToSlash(f[len("fabrik/"):])] = true
	}
	for _, rel := range repoFiles {
		assert.True(t, embeddedSet[rel], "repo file %s missing from embedded mirror — re-copy: cp -rf plugins/fabrik cli/internal/plugin/fabrik", rel)
	}
	assert.Len(t, embedded, len(repoFiles), "embedded mirror has files absent from repo source — re-copy")

	for _, rel := range repoFiles {
		want, err := os.ReadFile(filepath.Join(src, filepath.FromSlash(rel)))
		require.NoError(t, err)
		got, err := dist.ReadFile("fabrik/" + rel)
		require.NoError(t, err, "embedded read fabrik/%s", rel)
		assert.Equal(t, string(want), string(got), "drift in %s — re-copy", rel)
	}
}
