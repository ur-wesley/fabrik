package platform

import (
	"strings"
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
	assert.Len(t, ReleaseTargets, 5)
	assert.True(t, IsReleaseTarget("windows_amd64"))
	assert.True(t, IsReleaseTarget("linux_amd64"))
	assert.True(t, IsReleaseTarget("linux_arm64"))
	assert.True(t, IsReleaseTarget("darwin_amd64"))
	assert.True(t, IsReleaseTarget("darwin_arm64"))
	assert.False(t, IsReleaseTarget("windows_arm64"))
}

func TestCurrentResolves(t *testing.T) {
	k, err := Current()
	require.NoError(t, err)
	assert.Contains(t, k, "_")
}

// cli-4ag/cli-699: single table parity — AllKeys holds all 6 supported
// variants, ReleaseTargets the 5 fabrik binaries the release workflow builds
// (windows_arm64 is the documented exclusion), every target resolves via
// Key, and both asset-name helpers agree on the suffixes.
func TestPlatformTableParity(t *testing.T) {
	assert.Len(t, AllKeys, 6)
	assert.Len(t, ReleaseTargets, 5)
	for _, k := range ReleaseTargets {
		assert.Contains(t, AllKeys, k)
		goos, goarch, ok := splitKey(k)
		require.True(t, ok, "release target %s must be <goos>_<goarch>", k)
		got, err := Key(goos, goarch)
		require.NoError(t, err)
		assert.Equal(t, k, got)
	}
	// documented exclusion: resolves via Key (dep assets cover it) but has
	// no fabrik binary and is not a release target.
	got, err := Key("windows", "arm64")
	require.NoError(t, err)
	assert.Equal(t, "windows_arm64", got)
	assert.False(t, IsReleaseTarget("windows_arm64"))
	_, err = SelfUpdateAssetName("windows_arm64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "go install")
}

func splitKey(k string) (string, string, bool) {
	for _, goos := range []string{"windows", "linux", "darwin"} {
		if goarch, ok := strings.CutPrefix(k, goos+"_"); ok {
			return goos, goarch, true
		}
	}
	return "", "", false
}

// cli-699: one canonical asset-name impl — table-driven over every variant.
func TestCanonicalAssetNames(t *testing.T) {
	for _, k := range AllKeys {
		suffix, err := Suffix(k)
		require.NoError(t, err, "Suffix(%s)", k)
		// dep archives: <tool>_<ver>_<key> + windows->.zip else .tar.gz
		dep := DepAssetName("beads", "1.4.0", k)
		if strings.HasPrefix(k, "windows") {
			assert.Equal(t, "beads_1.4.0_"+k+".zip", dep)
		} else {
			assert.Equal(t, "beads_1.4.0_"+k+".tar.gz", dep)
		}
		// fabrik binaries: fabrik-<suffix>[.exe], except the exclusion
		name, err := SelfUpdateAssetName(k)
		if k == "windows_arm64" {
			require.Error(t, err)
			continue
		}
		require.NoError(t, err)
		want := "fabrik-" + suffix
		if strings.HasPrefix(k, "windows") {
			want += ".exe"
		}
		assert.Equal(t, want, name)
	}
	_, err := SelfUpdateAssetName("plan9_amd64")
	require.Error(t, err)
	_, err = Suffix("plan9_amd64")
	require.Error(t, err)
}
