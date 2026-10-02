package repo

import (
	"os"
	"path/filepath"
)

// IsGitRepo reports whether dir is inside a git work tree (walks up for .git).
func IsGitRepo(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	for {
		if _, err := os.Stat(filepath.Join(abs, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return false
		}
		abs = parent
	}
}
