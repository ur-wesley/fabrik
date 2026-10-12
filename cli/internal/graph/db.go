package graph

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/sqliteutil"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

type DB struct {
	Repo string
	sql  *sql.DB
}

func Open(ctx context.Context, repoRoot string) (*DB, error) {
	root, err := store.RequireRepo(repoRoot)
	if err != nil {
		return nil, err
	}
	dbPath := store.GraphDBPath(root)
	if _, err := sqlmigrate.ApplySQLite(ctx, "graph", dbPath); err != nil {
		return nil, err
	}
	sqlDB, err := sql.Open("sqlite", sqliteutil.DSN(dbPath))
	if err != nil {
		return nil, err
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return &DB{Repo: root, sql: sqlDB}, nil
}

func (d *DB) Close() error {
	if d == nil || d.sql == nil {
		return nil
	}
	return d.sql.Close()
}

func (d *DB) requireIndexed(ctx context.Context) error {
	var n int
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_files`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrEmpty
	}
	return nil
}

func (d *DB) indexed(ctx context.Context) (bool, error) {
	var n int
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(*) FROM graph_files`).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (d *DB) loadNode(ctx context.Context, id string) (Node, error) {
	var n Node
	var kind string
	err := d.sql.QueryRowContext(ctx, `
		SELECT id, kind, name, path, line, lang FROM graph_nodes WHERE id = ?`, id).
		Scan(&n.ID, &kind, &n.Name, &n.Path, &n.Line, &n.Lang)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Node{}, fmt.Errorf("fabrik: node %s not found", id)
		}
		return Node{}, err
	}
	n.Kind = kind
	return n, nil
}
