// Package fileutil holds shared filesystem helpers: named permission
// constants (one audit point for GOOS portability — on Windows the mode bits
// are accepted but largely inert) and the single CopyFile implementation
// used by selfupdate (binary install) and sqlmigrate (DB backup).
package fileutil

import (
	"io"
	"os"
	"path/filepath"
)

// Permission modes used across the CLI. Prefer these over raw literals so
// `grep 0o755` (etc.) only matches here and in test fixtures.
const (
	// DirPerm is for directories we create.
	DirPerm os.FileMode = 0o755
	// ExePerm is for installed executables.
	ExePerm os.FileMode = 0o755
	// ConfigPerm is for config / JSON / markdown files we write.
	ConfigPerm os.FileMode = 0o644
	// SecretPerm is for secrets: lock files, PID files, database backups.
	SecretPerm os.FileMode = 0o600
	// ArchiveFileMask clamps permission bits carried inside downloaded
	// archives: it strips setuid/setgid/sticky and group/other write while
	// preserving the usual layouts (0644 stays 0644, 0755 stays 0755, 0777
	// becomes 0755).
	ArchiveFileMask os.FileMode = 0o755
)

// CopyFile copies src to dst, creating parent dirs (DirPerm) as needed and
// creating/truncating dst with mode. A missing src is an error.
func CopyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), DirPerm); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
