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
	"runtime"
	"strings"
	"time"

	"github.com/ur-wesley/fabrik/cli/internal/fileutil"
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
	lower := strings.ToLower(archive)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		xerr = unzip(archive, dir)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
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
		if err == nil && !d.IsDir() && strings.EqualFold(d.Name(), binary) {
			found = p
		}
		return nil
	})
	if found == "" {
		os.RemoveAll(dir)
		return "", fmt.Errorf("could not find %s inside archive", binary)
	}
	// Archives (notably zips) often carry no exec bit; ensure the extracted
	// binary is executable on Unix. No-op on Windows (no exec bit there).
	if runtime.GOOS != "windows" {
		if err := os.Chmod(found, fileutil.ExePerm); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
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
	if f.FileInfo().Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("archive entry is a symlink: %q", f.Name)
	}
	dst, err := resolveWithin(dir, f.Name)
	if err != nil {
		return err
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(dst, fileutil.DirPerm)
	}
	if err := os.MkdirAll(filepath.Dir(dst), fileutil.DirPerm); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	// Clamp archive-carried modes: strip setuid/setgid/sticky and
	// group/other write (a hostile archive must not mint setuid files).
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode().Perm()&fileutil.ArchiveFileMask)
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
		switch hdr.Typeflag {
		case tar.TypeReg, tar.TypeRegA, tar.TypeDir:
			// allowed: regular file or directory
		default:
			return fmt.Errorf("archive entry has disallowed type %q: %q", rune(hdr.Typeflag), hdr.Name)
		}
		dst, err := resolveWithin(dir, hdr.Name)
		if err != nil {
			return err
		}
		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(dst, fileutil.DirPerm); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), fileutil.DirPerm); err != nil {
			return err
		}
		// Clamp archive-carried modes (see writeZipEntry).
		out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode().Perm()&fileutil.ArchiveFileMask)
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

// resolveWithin joins an archive entry name to dir and ensures the result
// stays inside dir (Zip-Slip containment). It rejects absolute paths and
// any entry that cleans to something outside dir (e.g. "../evil.txt").
func resolveWithin(dir, name string) (string, error) {
	// Reject Unix-absolute names outright: on Windows "/abs.txt" is not
	// reported by filepath.IsAbs but still escapes on Unix targets.
	if strings.HasPrefix(filepath.ToSlash(name), "/") {
		return "", fmt.Errorf("archive entry escapes target dir: %q", name)
	}
	cleaned := filepath.Clean(filepath.FromSlash(name))
	if cleaned == "." {
		return "", fmt.Errorf("archive entry has empty name")
	}
	if filepath.IsAbs(cleaned) || filepath.VolumeName(cleaned) != "" ||
		strings.HasPrefix(cleaned, string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry escapes target dir: %q", name)
	}
	dst := filepath.Join(dir, cleaned)
	base := filepath.Clean(dir)
	if dst != base && !strings.HasPrefix(dst, base+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry escapes target dir: %q", name)
	}
	return dst, nil
}
