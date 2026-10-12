// Package check implements `fabrik check`.
package check

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return "fabrik check: store not ready"
}

type RepoResult struct {
	Dolt        string `json:"dolt"`
	DoltPath    string `json:"dolt_path"`
	Graph       string `json:"graph"`
	GraphPath   string `json:"graph_path"`
	SchemaDolt  int    `json:"schema_dolt"`
	SchemaGraph int    `json:"schema_graph"`
	Issues      int    `json:"issues"`
	GraphFiles  int    `json:"graph_files"`
}

func Run(ctx context.Context, repoRoot string) (RepoResult, int) {
	if repoRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return RepoResult{}, 1
		}
		repoRoot = cwd
	}
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return RepoResult{}, 1
	}

	res := RepoResult{
		DoltPath:  store.DoltDir(abs),
		GraphPath: store.GraphDBPath(abs),
	}

	if !dirExists(res.DoltPath) {
		res.Dolt = "missing"
		res.Graph = "missing"
		return res, 1
	}
	res.Dolt = "ok"

	st, err := doltstore.Open(ctx, abs)
	if err != nil {
		res.Dolt = "missing"
		res.Graph = "missing"
		return res, 1
	}
	defer st.Close()
	if v, err := sqlmigrate.CurrentVersion(ctx, st.DB); err == nil {
		res.SchemaDolt = v
	}
	if err := countTable(ctx, st.DB, "issues", &res.Issues); err != nil {
		res.Issues = 0
	}

	if _, err := os.Stat(res.GraphPath); err != nil {
		res.Graph = "missing"
		return res, 0
	}
	res.Graph = "ok"

	db, err := sql.Open("sqlite", res.GraphPath)
	if err != nil {
		res.Graph = "missing"
		return res, 0
	}
	defer db.Close()
	if v, err := sqlmigrate.CurrentVersion(ctx, db); err == nil {
		res.SchemaGraph = v
	}
	_ = countTable(ctx, db, "graph_files", &res.GraphFiles)
	return res, 0
}

func countTable(ctx context.Context, db *sql.DB, table string, out *int) error {
	return db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(out)
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func Print(out io.Writer, r RepoResult) {
	fmt.Fprintf(out, "dolt %s  schema %d  issues %d\n", r.Dolt, r.SchemaDolt, r.Issues)
	fmt.Fprintf(out, "graph %s  graph_files %d\n", r.Graph, r.GraphFiles)
}

func PrintJSONTo(out io.Writer, r RepoResult) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
