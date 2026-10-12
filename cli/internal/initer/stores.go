package initer

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/exec"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

type StoreResult struct {
	DoltPath  string
	GraphPath string
	Created   bool
}

var gitignoreLines = []string{
	".fabrik/dolt/",
	".fabrik/graph.db",
	".fabrik/export/",
}

func doltDatabaseExists(doltDir string) bool {
	return dirExists(filepath.Join(doltDir, "fabrik"))
}

func InitRepoStores(ctx context.Context, abs string, r exec.Runner) (StoreResult, error) {
	doltDir := store.DoltDir(abs)
	graphPath := store.GraphDBPath(abs)
	res := StoreResult{DoltPath: doltDir, GraphPath: graphPath, Created: !doltDatabaseExists(doltDir)}

	st, err := doltstore.Open(ctx, abs)
	if err != nil {
		return StoreResult{}, err
	}
	defer st.Close()

	if err := configureDolt(ctx, st.DB, abs, r); err != nil {
		return StoreResult{}, err
	}

	if _, err := sqlmigrate.ApplySQLite(ctx, "graph", graphPath); err != nil {
		return StoreResult{}, err
	}

	if err := ensureRepoGitignore(abs); err != nil {
		return StoreResult{}, err
	}
	return res, nil
}

func configureDolt(ctx context.Context, db *sql.DB, abs string, r exec.Runner) error {
	name, _ := r.Run(ctx, "git", "-C", abs, "config", "user.name")
	email, _ := r.Run(ctx, "git", "-C", abs, "config", "user.email")
	name = strings.TrimSpace(firstLine(name))
	email = strings.TrimSpace(firstLine(email))
	if name == "" {
		name = "fabrik"
	}
	if email == "" {
		email = "fabrik@local"
	}
	author := fmt.Sprintf("%s <%s>", name, email)
	_, _ = db.ExecContext(ctx, `SET @@dolt_commit_author = ?`, author)

	origin, _ := r.Run(ctx, "git", "-C", abs, "remote", "get-url", "origin")
	origin = strings.TrimSpace(firstLine(origin))
	if origin == "" {
		return nil
	}
	var cnt int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dolt_remotes WHERE name = 'origin'`).Scan(&cnt)
	if err == nil && cnt > 0 {
		return nil
	}
	_, err = db.ExecContext(ctx, `CALL dolt_remote('add', 'origin', ?)`, origin)
	return err
}

func ensureRepoGitignore(abs string) error {
	path := filepath.Join(abs, ".gitignore")
	var lines []string
	if data, err := os.ReadFile(path); err == nil {
		lines = strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	}
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
	return os.WriteFile(path, []byte(body), 0o644)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
