package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"

	_ "modernc.org/sqlite"

	"github.com/ur-wesley/fabrik/cli/internal/listen"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

type CheckResult struct {
	Memory       string   `json:"memory"`
	MemoryPath   string   `json:"memory_path"`
	SchemaMemory int      `json:"schema_memory,omitempty"`
	Observations int      `json:"observations,omitempty"`
	Sessions     int      `json:"sessions,omitempty"`
	Projects     []string `json:"projects,omitempty"`
}

func Check(ctx context.Context) (CheckResult, error) {
	path, err := store.MemoryDBPath()
	if err != nil {
		return CheckResult{}, err
	}
	res := CheckResult{MemoryPath: path}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		res.Memory = "missing"
		return res, nil
	} else if err != nil {
		return CheckResult{}, err
	}

	changed, err := sqlmigrate.ApplySQLite(ctx, "memory", path)
	if err != nil {
		return CheckResult{}, err
	}
	if changed {
		listen.SignalMemory()
	}

	res.Memory = "ok"

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return CheckResult{}, err
	}
	defer releaseDB(db)
	if v, err := sqlmigrate.CurrentVersion(ctx, db); err == nil {
		res.SchemaMemory = v
	}

	if err := db.PingContext(ctx); err != nil {
		return CheckResult{}, err
	}
	if err := countTable(ctx, db, "observations", &res.Observations); err != nil {
		return CheckResult{}, err
	}
	if err := countTable(ctx, db, "sessions", &res.Sessions); err != nil {
		return CheckResult{}, err
	}
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT project FROM observations WHERE project != '' ORDER BY project`)
	if err != nil {
		return CheckResult{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return CheckResult{}, err
		}
		res.Projects = append(res.Projects, p)
	}
	return res, rows.Err()
}

func countTable(ctx context.Context, db *sql.DB, table string, out *int) error {
	return db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(out)
}

func PrintCheck(w io.Writer, r CheckResult) {
	if r.Memory == "missing" {
		_, _ = io.WriteString(w, "memory missing\n")
		return
	}
	_, _ = fmt.Fprintf(w, "memory ok  observations %d  projects %d\n", r.Observations, len(r.Projects))
}

func PrintCheckJSON(w io.Writer, r CheckResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
