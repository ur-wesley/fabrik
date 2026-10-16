package store

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func gitTopLevel(start string) (string, error) {
	out, err := exec.Command("git", "-C", start, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitMainWorktree(start string) (string, error) {
	out, err := exec.Command("git", "-C", start, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", err
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "worktree ")), nil
		}
	}
	return "", errors.New("no worktree")
}

func hasDolt(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, ".fabrik", "dolt"))
	return err == nil && st.IsDir()
}

// hasIssuesDB reports the SQLite issue store (.fabrik/issues.db).
func hasIssuesDB(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, ".fabrik", "issues.db"))
	return err == nil && !st.IsDir()
}

// hasIssueStore reports any known issue store: legacy .fabrik/dolt or
// the SQLite .fabrik/issues.db.
func hasIssueStore(dir string) bool {
	return hasDolt(dir) || hasIssuesDB(dir)
}

func repoRootFromGit(start string) (string, bool) {
	top, err := gitTopLevel(start)
	if err != nil {
		return "", false
	}
	if hasIssueStore(top) {
		return top, true
	}
	main, err := gitMainWorktree(start)
	if err == nil && main != "" && main != top && hasIssueStore(main) {
		return main, true
	}
	return "", false
}
