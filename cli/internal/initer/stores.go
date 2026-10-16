package initer

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/fileutil"
	"github.com/ur-wesley/fabrik/cli/internal/issuestore"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

type StoreResult struct {
	// IssuesPath is the SQLite issue store (.fabrik/issues.db).
	IssuesPath string
	// DoltPath is a deprecated alias of IssuesPath, kept so existing
	// callers keep compiling; new code uses IssuesPath.
	DoltPath  string
	GraphPath string
	Created   bool
}

var gitignoreLines = []string{
	".fabrik/issues.db",
	".fabrik/issues.db-wal",
	".fabrik/issues.db-shm",
	".fabrik/graph.db",
}

func issuesDBExists(dbPath string) bool {
	st, err := os.Stat(dbPath)
	return err == nil && !st.IsDir()
}

func InitRepoStores(ctx context.Context, abs string, r exec.Runner) (StoreResult, error) {
	dbPath := store.IssuesDBPath(abs)
	graphPath := store.GraphDBPath(abs)
	res := StoreResult{IssuesPath: dbPath, DoltPath: dbPath, GraphPath: graphPath, Created: !issuesDBExists(dbPath)}

	st, err := issuestore.Open(ctx, abs)
	if err != nil {
		return StoreResult{}, err
	}
	defer st.Close()

	if _, err := sqlmigrate.ApplySQLite(ctx, "graph", graphPath); err != nil {
		return StoreResult{}, err
	}

	if err := ensureRepoGitignore(abs); err != nil {
		return StoreResult{}, err
	}
	return res, nil
}

func ensureRepoGitignore(abs string) error {
	path := filepath.Join(abs, ".gitignore")
	var lines []string
	if data, err := os.ReadFile(path); err == nil {
		lines = strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	}
	// The export snapshot (.fabrik/export/issues.jsonl) is the git-tracked
	// sync unit since cli-a35. A legacy `.fabrik/export/` ignore would
	// silently untrack it and break push/pull, so drop those lines.
	kept := lines[:0]
	for _, l := range lines {
		if t := strings.TrimSpace(l); t == ".fabrik/export/" || t == ".fabrik/export" {
			continue
		}
		kept = append(kept, l)
	}
	lines = kept
	existing := make(map[string]bool)
	for _, l := range lines {
		existing[strings.TrimSpace(l)] = true
	}
	for _, want := range gitignoreLines {
		if !existing[want] {
			lines = append(lines, want)
		}
	}
	if len(lines) == 0 {
		return nil
	}
	body := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(body), fileutil.ConfigPerm)
}
