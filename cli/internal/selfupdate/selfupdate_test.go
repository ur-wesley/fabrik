package selfupdate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewer(t *testing.T) {
	assert.True(t, newer("1.0.0", "v1.1.0"))
	assert.True(t, newer("v1.0.0", "v1.0.1"))
	assert.False(t, newer("v1.1.0", "v1.1.0"))
	assert.False(t, newer("v2.0.0", "v1.9.9"))
	assert.True(t, newer("dev", "v1.0.0"), "dev builds always update")
	assert.False(t, newer("v1.0.0", "nonsense"))
}

func TestAssetName(t *testing.T) {
	for _, tc := range [][2]string{
		{"windows_amd64", "fabrik-windows-amd64.exe"},
		{"linux_amd64", "fabrik-linux-amd64"},
		{"darwin_arm64", "fabrik-darwin-arm64"},
	} {
		got, err := assetName(tc[0])
		require.NoError(t, err)
		assert.Equal(t, tc[1], got)
	}
	_, err := assetName("linux_arm64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "go install")
}

// stubRelease serves a fake GitHub API + asset endpoints.
func stubRelease(t *testing.T, tag, binary string, corruptSum, omitSum bool) (*httptest.Server, string) {
	t.Helper()
	body := []byte("new-binary-" + tag)
	sum := sha256.Sum256(body)
	sumHex := hex.EncodeToString(sum[:])
	if corruptSum {
		sumHex = hex.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	}
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/ur-wesley/fabrik/releases/latest", "/repos/ur-wesley/fabrik/releases/tags/" + tag:
			assets := fmt.Sprintf(`[{"name":%q,"browser_download_url":%q},{"name":%q,"browser_download_url":%q}]`,
				binary, srv.URL+"/dl/"+binary, binary+".sha256", srv.URL+"/dl/"+binary+".sha256")
			if omitSum {
				assets = fmt.Sprintf(`[{"name":%q,"browser_download_url":%q}]`, binary, srv.URL+"/dl/"+binary)
			}
			fmt.Fprintf(w, `{"tag_name":%q,"assets":%s}`, tag, assets)
		case "/dl/" + binary:
			w.Write(body)
		case "/dl/" + binary + ".sha256":
			fmt.Fprintf(w, "%s  %s\n", sumHex, binary)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, string(body)
}

func fakeExe(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fabrik")
	require.NoError(t, os.WriteFile(p, []byte(content), 0o755))
	return p
}

func TestUpdateFullSwap(t *testing.T) {
	srv, wantBody := stubRelease(t, "v1.1.0", "fabrik-linux-amd64", false, false)
	var out bytes.Buffer
	exe := fakeExe(t, "old-binary")
	require.NoError(t, Update(context.Background(), Config{Yes: true}, Deps{
		Out:         &out,
		APIBase:     srv.URL,
		Client:      srv.Client(),
		ExecPath:    exe,
		PlatformKey: func() (string, error) { return "linux_amd64", nil },
		Current:     "1.0.0",
	}))
	got, err := os.ReadFile(exe)
	require.NoError(t, err)
	assert.Equal(t, wantBody, string(got))
	rollback, err := os.ReadFile(exe + ".old")
	require.NoError(t, err)
	assert.Equal(t, "old-binary", string(rollback))
	assert.Contains(t, out.String(), "Updated to v1.1.0")
}

func TestUpdateAlreadyCurrent(t *testing.T) {
	srv, _ := stubRelease(t, "v1.0.0", "fabrik-linux-amd64", false, false)
	var out bytes.Buffer
	exe := fakeExe(t, "old-binary")
	require.NoError(t, Update(context.Background(), Config{Yes: true}, Deps{
		Out:         &out,
		APIBase:     srv.URL,
		Client:      srv.Client(),
		ExecPath:    exe,
		PlatformKey: func() (string, error) { return "linux_amd64", nil },
		Current:     "1.0.0",
	}))
	assert.Contains(t, out.String(), "up to date")
	got, _ := os.ReadFile(exe)
	assert.Equal(t, "old-binary", string(got))
}

func TestUpdateChecksumMismatchFailsClosed(t *testing.T) {
	srv, _ := stubRelease(t, "v1.1.0", "fabrik-linux-amd64", true, false)
	var out bytes.Buffer
	exe := fakeExe(t, "old-binary")
	err := Update(context.Background(), Config{Yes: true}, Deps{
		Out:         &out,
		APIBase:     srv.URL,
		Client:      srv.Client(),
		ExecPath:    exe,
		PlatformKey: func() (string, error) { return "linux_amd64", nil },
		Current:     "1.0.0",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
	got, _ := os.ReadFile(exe)
	assert.Equal(t, "old-binary", string(got), "binary must be untouched")
}

func TestUpdateMissingChecksumRefused(t *testing.T) {
	srv, _ := stubRelease(t, "v1.1.0", "fabrik-linux-amd64", false, true)
	var out bytes.Buffer
	exe := fakeExe(t, "old-binary")
	err := Update(context.Background(), Config{Yes: true}, Deps{
		Out:         &out,
		APIBase:     srv.URL,
		Client:      srv.Client(),
		ExecPath:    exe,
		PlatformKey: func() (string, error) { return "linux_amd64", nil },
		Current:     "1.0.0",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no checksum")
}

func TestUpdateCheckOnly(t *testing.T) {
	srv, _ := stubRelease(t, "v2.0.0", "fabrik-linux-amd64", false, false)
	var out bytes.Buffer
	exe := fakeExe(t, "old-binary")
	require.NoError(t, Update(context.Background(), Config{Yes: true, CheckOnly: true}, Deps{
		Out:         &out,
		APIBase:     srv.URL,
		Client:      srv.Client(),
		ExecPath:    exe,
		PlatformKey: func() (string, error) { return "linux_amd64", nil },
		Current:     "1.0.0",
	}))
	assert.Contains(t, out.String(), "1.0.0 -> v2.0.0")
	got, _ := os.ReadFile(exe)
	assert.Equal(t, "old-binary", string(got))
}

func TestUpdateUnsupportedPlatform(t *testing.T) {
	srv, _ := stubRelease(t, "v1.1.0", "fabrik-linux-amd64", false, false)
	var out bytes.Buffer
	err := Update(context.Background(), Config{Yes: true}, Deps{
		Out:         &out,
		APIBase:     srv.URL,
		Client:      srv.Client(),
		ExecPath:    fakeExe(t, "x"),
		PlatformKey: func() (string, error) { return "linux_arm64", nil },
		Current:     "1.0.0",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "go install")
}
