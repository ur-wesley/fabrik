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
	"github.com/ur-wesley/fabrik/cli/internal/memory"
	"github.com/ur-wesley/fabrik/cli/internal/plugin"
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
	// Memory stats are text-output only (spec v2/check.md: repo JSON has
	// dolt/graph only; mem check JSON is separate). json:"-" keeps
	// PrintJSONTo output unchanged.
	Memory       string `json:"-"`
	Observations int    `json:"-"`
	Sessions     int    `json:"-"`
	Projects     int    `json:"-"`
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
		fillMemory(ctx, &res)
		return res, 1
	}
	res.Dolt = "ok"

	st, err := doltstore.Open(ctx, abs)
	if err != nil {
		res.Dolt = "missing"
		res.Graph = "missing"
		fillMemory(ctx, &res)
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
		fillMemory(ctx, &res)
		return res, 0
	}
	res.Graph = "ok"

	db, err := sql.Open("sqlite", res.GraphPath)
	if err != nil {
		res.Graph = "missing"
		fillMemory(ctx, &res)
		return res, 0
	}
	defer db.Close()
	if v, err := sqlmigrate.CurrentVersion(ctx, db); err == nil {
		res.SchemaGraph = v
	}
	_ = countTable(ctx, db, "graph_files", &res.GraphFiles)
	fillMemory(ctx, &res)
	return res, 0
}

// fillMemory populates the text-only memory stats via memory.Check.
// Errors are ignored: an unavailable memory.db prints as "memory missing".
func fillMemory(ctx context.Context, res *RepoResult) {
	res.Memory = "missing"
	if mem, err := memory.Check(ctx); err == nil {
		res.Memory = mem.Memory
		res.Observations = mem.Observations
		res.Sessions = mem.Sessions
		res.Projects = len(mem.Projects)
	}
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
	if r.Memory == "ok" {
		fmt.Fprintf(out, "memory ok  observations %d  projects %d\n", r.Observations, r.Projects)
	} else {
		fmt.Fprintln(out, "memory missing")
	}
}

func PrintJSONTo(out io.Writer, r RepoResult) error {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// PrintPluginAdvisory prints a machine-level freshness warning when the
// global Antigravity/Gemini CLI plugin copy (~/.gemini/config/plugins/fabrik)
// is missing or older than the embedded source. Advisory only: it never
// affects the check exit code. Unresolvable home is silently skipped.
func PrintPluginAdvisory(out io.Writer) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	if adv := plugin.Advisory(home); adv != "" {
		fmt.Fprintln(out, adv)
	}
}
