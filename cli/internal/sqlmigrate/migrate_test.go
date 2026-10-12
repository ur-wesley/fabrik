package sqlmigrate

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestApplySQLiteMemoryIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "memory.db")
	ctx := context.Background()
	if _, err := ApplySQLite(ctx, "memory", dbPath); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplySQLite(ctx, "memory", dbPath); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 migration rows, got %d", n)
	}
}

func TestApplySQLiteGraphCreatesTables(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "graph.db")
	ctx := context.Background()
	if _, err := ApplySQLite(ctx, "graph", dbPath); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM graph_files`).Scan(&n); err != nil {
		t.Fatal(err)
	}
}

func TestMaxVersionMemory(t *testing.T) {
	v, err := MaxVersion("memory")
	if err != nil {
		t.Fatal(err)
	}
	if v < 1 {
		t.Fatalf("expected >= 1, got %d", v)
	}
}

func TestMaxVersionDolt(t *testing.T) {
	v, err := MaxVersion("dolt")
	if err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Fatalf("expected 1, got %d", v)
	}
}

func TestErrSchemaNewer(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "memory.db")
	ctx := context.Background()
	if _, err := ApplySQLite(ctx, "memory", dbPath); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO schema_migrations (version, filename, applied_at) VALUES (999, '999_fake.sql', '2026-01-01T00:00:00Z')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ApplySQLite(ctx, "memory", dbPath)
	if err == nil {
		t.Fatal("expected error")
	}
	var newer *ErrSchemaNewer
	if _, ok := err.(*ErrSchemaNewer); !ok {
		if e, ok := unwrapNewer(err); ok {
			newer = e
		}
	}
	if newer == nil {
		if e, ok := err.(*ErrSchemaNewer); ok {
			newer = e
		}
	}
	if newer == nil {
		t.Fatalf("expected ErrSchemaNewer, got %T %v", err, err)
	}
}

func unwrapNewer(err error) (*ErrSchemaNewer, bool) {
	e, ok := err.(*ErrSchemaNewer)
	return e, ok
}
