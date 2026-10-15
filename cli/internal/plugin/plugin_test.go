package plugin

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestVersionMatchesEmbeddedManifest(t *testing.T) {
	assert.Equal(t, "0.2.0", Version())
}

func TestInstalledVersionMissing(t *testing.T) {
	home := fakeHome(t)
	assert.Equal(t, "", InstalledVersion(home))
	assert.True(t, NeedsSync(home))
	assert.Contains(t, Advisory(home), "not installed")
}

func TestInstalledVersionUnparsable(t *testing.T) {
	home := fakeHome(t)
	require.NoError(t, os.MkdirAll(Dir(home), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(Dir(home), "plugin.json"), []byte("not json"), 0o644))
	assert.Equal(t, "", InstalledVersion(home))
	assert.True(t, NeedsSync(home))
}

func TestCmpVersion(t *testing.T) {
	assert.Equal(t, 0, cmpVersion("0.2.0", "0.2.0"))
	assert.Equal(t, -1, cmpVersion("0.1.0", "0.2.0"))
	assert.Equal(t, 1, cmpVersion("0.3.0", "0.2.0"))
	assert.Equal(t, 0, cmpVersion("0.2", "0.2.0"), "missing parts count as 0")
	assert.Equal(t, 0, cmpVersion("v0.2.0", "0.2.0"))
	assert.Equal(t, 1, cmpVersion("0.10.0", "0.2.0"), "numeric, not lexicographic")
}

func TestSyncCopiesTree(t *testing.T) {
	home := fakeHome(t)
	var out bytes.Buffer
	require.NoError(t, Sync(home, false, &out))
	assert.Contains(t, out.String(), "plugin fabrik missing -> 0.2.0")
	assert.Equal(t, "0.2.0", InstalledVersion(home))
	assert.False(t, NeedsSync(home))
	assert.Equal(t, "", Advisory(home))
	// Spot-check a nested skill file landed with repo content.
	skill, err := os.ReadFile(filepath.Join(Dir(home), "skills", "fabrik", "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(skill), "Antigravity")
}

func TestSyncUpToDateIsNoop(t *testing.T) {
	home := fakeHome(t)
	var out bytes.Buffer
	require.NoError(t, Sync(home, false, &out))
	skillPath := filepath.Join(Dir(home), "skills", "fabrik", "SKILL.md")
	before, err := os.ReadFile(skillPath)
	require.NoError(t, err)
	infoBefore, err := os.Stat(skillPath)
	require.NoError(t, err)

	out.Reset()
	require.NoError(t, Sync(home, false, &out))
	assert.Contains(t, out.String(), "plugin fabrik 0.2.0 up to date")
	after, err := os.ReadFile(skillPath)
	require.NoError(t, err)
	assert.Equal(t, before, after)
	infoAfter, err := os.Stat(skillPath)
	require.NoError(t, err)
	assert.Equal(t, infoBefore.ModTime(), infoAfter.ModTime(), "up-to-date sync must not rewrite files")
}

func TestSyncDryRunTouchesNothing(t *testing.T) {
	home := fakeHome(t)
	var out bytes.Buffer
	require.NoError(t, Sync(home, true, &out))
	assert.Contains(t, out.String(), "dry-run: sync fabrik plugin")
	entries, err := os.ReadDir(home)
	require.NoError(t, err)
	assert.Empty(t, entries)
	assert.True(t, NeedsSync(home), "dry-run must not change state")
}

func TestSyncOverwritesStaleCopy(t *testing.T) {
	home := fakeHome(t)
	require.NoError(t, os.MkdirAll(Dir(home), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(Dir(home), "plugin.json"),
		[]byte(`{"name":"fabrik","version":"0.1.0"}`),
		0o644,
	))
	assert.True(t, NeedsSync(home))
	assert.Contains(t, Advisory(home), "0.1.0 is stale")

	var out bytes.Buffer
	require.NoError(t, Sync(home, false, &out))
	assert.Contains(t, out.String(), "plugin fabrik 0.1.0 -> 0.2.0")
	assert.Equal(t, "0.2.0", InstalledVersion(home))
}

func TestSyncNewerInstalledLeftAlone(t *testing.T) {
	home := fakeHome(t)
	require.NoError(t, os.MkdirAll(Dir(home), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(Dir(home), "plugin.json"),
		[]byte(`{"name":"fabrik","version":"9.9.9"}`),
		0o644,
	))
	assert.False(t, NeedsSync(home))
	var out bytes.Buffer
	require.NoError(t, Sync(home, false, &out))
	assert.Contains(t, out.String(), "up to date")
	assert.Equal(t, "9.9.9", InstalledVersion(home))
}
