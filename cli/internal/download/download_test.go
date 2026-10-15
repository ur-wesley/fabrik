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

func writeZipWithNames(t *testing.T, path string, names []string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	w := zip.NewWriter(f)
	for _, n := range names {
		zf, err := w.Create(n)
		require.NoError(t, err)
		_, err = zf.Write([]byte("evil"))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())
}

func writeTarWithHeaders(t *testing.T, path string, hdrs []*tar.Header, bodies [][]byte) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for i, h := range hdrs {
		require.NoError(t, tw.WriteHeader(h))
		if len(bodies[i]) > 0 {
			_, err = tw.Write(bodies[i])
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

func TestUnzipRejectsZipSlip(t *testing.T) {
	parent := t.TempDir()
	arc := filepath.Join(parent, "evil.zip")
	writeZipWithNames(t, arc, []string{"../evil.txt"})
	dir := filepath.Join(parent, "out")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.Error(t, unzip(arc, dir))
	assert.NoFileExists(t, filepath.Join(parent, "evil.txt"))
	assert.NoFileExists(t, filepath.Join(dir, "evil.txt"))
}

func TestUnzipRejectsAbsolutePath(t *testing.T) {
	parent := t.TempDir()
	arc := filepath.Join(parent, "abs.zip")
	writeZipWithNames(t, arc, []string{"/abs.txt"})
	dir := filepath.Join(parent, "out")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.Error(t, unzip(arc, dir))
}

func TestUnzipRejectsSymlink(t *testing.T) {
	parent := t.TempDir()
	arc := filepath.Join(parent, "link.zip")
	f, err := os.Create(arc)
	require.NoError(t, err)
	w := zip.NewWriter(f)
	hdr := &zip.FileHeader{Name: "link", Method: zip.Store}
	hdr.SetMode(os.ModeSymlink | 0o777)
	zf, err := w.CreateHeader(hdr)
	require.NoError(t, err)
	_, err = zf.Write([]byte("/etc/passwd"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	require.NoError(t, f.Close())
	dir := filepath.Join(parent, "out")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.Error(t, unzip(arc, dir))
	assert.NoFileExists(t, filepath.Join(dir, "link"))
}

func TestUntarRejectsZipSlip(t *testing.T) {
	parent := t.TempDir()
	arc := filepath.Join(parent, "evil.tar.gz")
	body := []byte("evil")
	writeTarWithHeaders(t, arc,
		[]*tar.Header{{Name: "../evil.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}},
		[][]byte{body})
	dir := filepath.Join(parent, "out")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.Error(t, untar(arc, dir))
	assert.NoFileExists(t, filepath.Join(parent, "evil.txt"))
}

func TestUntarRejectsSymlinkAndHardlink(t *testing.T) {
	for _, tc := range []struct {
		name string
		hdr  *tar.Header
	}{
		{"symlink", &tar.Header{Name: "link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd", Mode: 0o777}},
		{"hardlink", &tar.Header{Name: "hard", Typeflag: tar.TypeLink, Linkname: "sub/engram", Mode: 0o644}},
		{"fifo", &tar.Header{Name: "pipe", Typeflag: tar.TypeFifo, Mode: 0o644}},
		{"char", &tar.Header{Name: "chr", Typeflag: tar.TypeChar, Mode: 0o644}},
		{"block", &tar.Header{Name: "blk", Typeflag: tar.TypeBlock, Mode: 0o644}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			arc := filepath.Join(parent, "x.tar.gz")
			writeTarWithHeaders(t, arc, []*tar.Header{tc.hdr}, [][]byte{nil})
			dir := filepath.Join(parent, "out")
			require.NoError(t, os.MkdirAll(dir, 0o755))
			require.Error(t, untar(arc, dir))
			assert.NoFileExists(t, filepath.Join(dir, tc.hdr.Name))
		})
	}
}
