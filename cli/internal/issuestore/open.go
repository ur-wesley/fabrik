package issuestore

import (
	"context"
	"database/sql"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/sqliteutil"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

// Store is the SQLite-backed issue store. The CRUD below mirrors
// doltstore.Store (same SQL, same Ready semantics); only the open path and
// the commit step differ: SQLite needs no version-control commit.
type Store struct {
	Repo string
	DB   *sql.DB
}

// CreateInput mirrors doltstore.CreateInput.
type CreateInput = doltstore.CreateInput

// Open opens (creating on demand) the .fabrik/issues.db SQLite store under
// repoRoot and applies pending sql/issues migrations.
func Open(ctx context.Context, repoRoot string) (*Store, error) {
	dbPath := store.IssuesDBPath(repoRoot)
	if _, err := sqlmigrate.ApplySQLite(ctx, "issues", dbPath); err != nil {
		return nil, StoreErr(err)
	}
	db, err := sql.Open("sqlite", sqliteutil.DSN(dbPath))
	if err != nil {
		return nil, StoreErr(err)
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, StoreErr(err)
	}
	if err := checkMetaSchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{Repo: repoRoot, DB: db}, nil
}

// Close releases the store handle. sql.DB.Close is idempotent, so deferred
// double-close after an explicit Close is safe.
func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

// RepoPath returns the repo root owning this store.
func (s *Store) RepoPath() string {
	if s == nil {
		return ""
	}
	return s.Repo
}

// Commit is a no-op kept for API parity with doltstore.Store: SQLite writes
// are durable without a version-control commit step.
func (s *Store) Commit(_ context.Context, _ string) error {
	return nil
}

func checkMetaSchema(ctx context.Context, db *sql.DB) error {
	var val string
	err := db.QueryRowContext(ctx, `SELECT value FROM meta WHERE name = 'schema_version'`).Scan(&val)
	if err != nil {
		if isNoTable(err) {
			return nil
		}
		return StoreErr(err)
	}
	max, err := sqlmigrate.MaxVersion("issues")
	if err != nil {
		return StoreErr(err)
	}
	have, _ := parseInt(val)
	if have > max {
		return &sqlmigrate.ErrSchemaNewer{Store: "issues", Have: have, Binary: max}
	}
	return nil
}

func parseInt(s string) (int, bool) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

func isNoTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such table") || strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "not found")
}
