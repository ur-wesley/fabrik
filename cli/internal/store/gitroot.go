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

func repoRootFromGit(start string) (string, bool) {
	top, err := gitTopLevel(start)
	if err != nil {
		return "", false
	}
	if hasDolt(top) {
		return top, true
	}
	main, err := gitMainWorktree(start)
	if err == nil && main != "" && main != top && hasDolt(main) {
		return main, true
	}
	return "", false
}
