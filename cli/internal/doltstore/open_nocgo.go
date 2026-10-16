package doltstore

import (
	"context"
	"database/sql"

	_ "modernc.org/sqlite"

	"github.com/ur-wesley/fabrik/cli/internal/sqliteutil"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

// HasCGO stays as a compatibility constant (always false now): the
// SQLite issue store needs no CGO. Callers that gated on the embedded
// Dolt engine (migrater beads leg, reseter, cgo_skip_test) keep compiling;
// the beads leg now runs unconditionally against SQLite.
const HasCGO = false

func openDB(ctx context.Context, repoRoot string) (*sql.DB, error) {
	dbPath := store.IssuesDBPath(repoRoot)
	if _, err := sqlmigrate.ApplySQLite(ctx, "issues", dbPath); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", sqliteutil.DSN(dbPath))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(_ context.Context, _ *sql.DB) (bool, error) {
	return false, nil
}
