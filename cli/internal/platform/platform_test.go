package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKey(t *testing.T) {
	cases := map[[2]string]string{
		{"windows", "amd64"}: "windows_amd64",
		{"windows", "arm64"}: "windows_arm64",
		{"linux", "amd64"}:   "linux_amd64",
		{"linux", "x86_64"}:  "linux_amd64",
		{"linux", "aarch64"}: "linux_arm64",
		{"darwin", "amd64"}:  "darwin_amd64",
		{"darwin", "arm64"}:  "darwin_arm64",
	}
	for in, want := range cases {
		got, err := Key(in[0], in[1])
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
}

func TestKeyUnsupported(t *testing.T) {
	_, err := Key("freebsd", "amd64")
	require.Error(t, err)
	_, err = Key("linux", "386")
	require.Error(t, err)
}

func TestBinaryName(t *testing.T) {
	assert.Equal(t, "bd.exe", BinaryName("windows", "bd"))
	assert.Equal(t, "bd", BinaryName("linux", "bd"))
	assert.Equal(t, "bd", BinaryName("darwin", "bd"))
}

func TestReleaseTargets(t *testing.T) {
	assert.Len(t, ReleaseTargets, 3)
	assert.True(t, IsReleaseTarget("windows_amd64"))
	assert.True(t, IsReleaseTarget("linux_amd64"))
	assert.True(t, IsReleaseTarget("darwin_arm64"))
	assert.False(t, IsReleaseTarget("darwin_amd64"))
	assert.False(t, IsReleaseTarget("linux_arm64"))
}

func TestCurrentResolves(t *testing.T) {
	k, err := Current()
	require.NoError(t, err)
	assert.Contains(t, k, "_")
}
