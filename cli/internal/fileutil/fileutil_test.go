package fileutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	require.NoError(t, os.WriteFile(src, []byte("payload"), 0o644))

	dst := filepath.Join(dir, "sub", "deep", "dst.bin")
	require.NoError(t, CopyFile(src, dst, ExePerm))
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(got))
	if runtime.GOOS != "windows" {
		st, err := os.Stat(dst)
		require.NoError(t, err)
		assert.Equal(t, ExePerm, st.Mode().Perm())
	}
}

func TestCopyFileOverwrites(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	require.NoError(t, os.WriteFile(src, []byte("new"), 0o644))
	require.NoError(t, os.WriteFile(dst, []byte("old-longer-body"), 0o644))
	require.NoError(t, CopyFile(src, dst, ConfigPerm))
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "new", string(got))
}

func TestCopyFileMissingSrc(t *testing.T) {
	err := CopyFile(filepath.Join(t.TempDir(), "nope"), filepath.Join(t.TempDir(), "dst"), ConfigPerm)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nope")
}

func TestCopyFileExplicitSecretMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("perm bits are inert on Windows")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "db.sqlite")
	require.NoError(t, os.WriteFile(src, []byte("db-bytes"), 0o644))
	dst := filepath.Join(dir, "db.sqlite.bak")
	require.NoError(t, CopyFile(src, dst, SecretPerm))
	st, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, SecretPerm, st.Mode().Perm())
}
