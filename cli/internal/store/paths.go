package store

import (
	"os"
	"path/filepath"
)

// FabrikDir returns ~/.fabrik or FABRIK_MEMORY_DIR.
func FabrikDir() (string, error) {
	dir := os.Getenv("FABRIK_MEMORY_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".fabrik")
	}
	return dir, nil
}

// MemoryDBPath returns ~/.fabrik/memory.db or FABRIK_MEMORY_DIR/memory.db.
func MemoryDBPath() (string, error) {
	dir, err := FabrikDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "memory.db"), nil
}

// ListenJSONPath returns ~/.fabrik/listen.json or FABRIK_MEMORY_DIR/listen.json.
func ListenJSONPath() (string, error) {
	dir, err := FabrikDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "listen.json"), nil
}

// RepoRoot finds the Fabrik repo that owns issues for start (git worktrees share main .fabrik/dolt).
func RepoRoot(start string) (string, bool) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	if root, ok := repoRootFromGit(abs); ok {
		return root, true
	}
	for {
		if hasDolt(abs) {
			return abs, true
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return "", false
		}
		abs = parent
	}
}

// GraphDBPath returns .fabrik/graph.db under repo root.
func GraphDBPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".fabrik", "graph.db")
}

// DoltDir returns .fabrik/dolt under repo root.
func DoltDir(repoRoot string) string {
	return filepath.Join(repoRoot, ".fabrik", "dolt")
}
