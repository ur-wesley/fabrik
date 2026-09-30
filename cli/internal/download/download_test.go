package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestZip(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	w := zip.NewWriter(f)
	zf, err := w.Create("sub/bd")
	require.NoError(t, err)
	_, err = zf.Write([]byte("fake-binary"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())
}

func writeTestTarGz(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	body := []byte("fake-binary")
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "sub/engram", Mode: 0o755, Size: int64(len(body))}))
	_, err = tw.Write(body)
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

func TestExtractZip(t *testing.T) {
	arc := filepath.Join(t.TempDir(), "a.zip")
	writeTestZip(t, arc)
	found, err := ExtractBinary(arc, "bd")
	require.NoError(t, err)
	b, err := os.ReadFile(found)
	require.NoError(t, err)
	assert.Equal(t, "fake-binary", string(b))
}

func TestExtractTarGz(t *testing.T) {
	arc := filepath.Join(t.TempDir(), "a.tar.gz")
	writeTestTarGz(t, arc)
	found, err := ExtractBinary(arc, "engram")
	require.NoError(t, err)
	b, err := os.ReadFile(found)
	require.NoError(t, err)
	assert.Equal(t, "fake-binary", string(b))
}

func TestExtractMissing(t *testing.T) {
	arc := filepath.Join(t.TempDir(), "a.zip")
	writeTestZip(t, arc)
	_, err := ExtractBinary(arc, "nope")
	require.Error(t, err)
}

func TestGetRoundTrip(t *testing.T) {
	writeTestZip(t, filepath.Join(t.TempDir(), "x.zip"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(t.TempDir(), "missing"))
	}))
	defer srv.Close()

	old := Client
	Client = srv.Client()
	defer func() { Client = old }()

	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	defer ok.Close()
	p, err := Get(context.Background(), ok.URL)
	require.NoError(t, err)
	b, err := os.ReadFile(p)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))
	os.Remove(p)

	_, err = Get(context.Background(), srv.URL+"/nope")
	require.Error(t, err)
}
