// Package download fetches release archives and extracts one binary (stdlib only).
package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client is injectable for tests (http.DefaultClient outside tests).
var Client = &http.Client{Timeout: 5 * time.Minute}

// Get downloads url to a temp file, retrying transient failures.
func Get(ctx context.Context, url string) (string, error) {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		p, err := getOnce(ctx, url)
		if err == nil {
			return p, nil
		}
		last = err
	}
	return "", last
}

func getOnce(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %s", url, resp.Status)
	}
	f, err := os.CreateTemp("", "fabrik-dl-*")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

// ExtractBinary unpacks archive (.zip or .tar.gz) into a temp dir and returns
// the path of the first file named binary.
func ExtractBinary(archive, binary string) (string, error) {
	dir, err := os.MkdirTemp("", "fabrik-x-*")
	if err != nil {
		return "", err
	}
	var xerr error
	switch {
	case strings.HasSuffix(archive, ".zip"):
		xerr = unzip(archive, dir)
	case strings.HasSuffix(archive, ".tar.gz"):
		xerr = untar(archive, dir)
	default:
		xerr = fmt.Errorf("unknown archive type: %s", archive)
	}
	if xerr != nil {
		os.RemoveAll(dir)
		return "", xerr
	}
	var found string
	_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == binary {
			found = p
		}
		return nil
	})
	if found == "" {
		os.RemoveAll(dir)
		return "", fmt.Errorf("could not find %s inside archive", binary)
	}
	return found, nil
}

func unzip(src, dir string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if err := writeZipEntry(f, dir); err != nil {
			return err
		}
	}
	return nil
}

func writeZipEntry(f *zip.File, dir string) error {
	dst := filepath.Join(dir, filepath.FromSlash(f.Name))
	if f.FileInfo().IsDir() {
		return os.MkdirAll(dst, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

func untar(src, dir string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, filepath.FromSlash(hdr.Name))
		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		out.Close()
	}
}
