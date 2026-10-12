//go:build cgo

package doltstore

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v4"
	embedded "github.com/dolthub/driver"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
)

const HasCGO = true

func openDB(ctx context.Context, doltDir string) (*sql.DB, error) {
	if err := os.MkdirAll(doltDir, 0o755); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(doltDir)
	if err != nil {
		return nil, err
	}
	if err := ensureFabrikDatabase(ctx, abs); err != nil {
		return nil, err
	}
	return connectDolt(ctx, abs, "fabrik")
}

func ensureFabrikDatabase(ctx context.Context, absDir string) error {
	db, err := connectDolt(ctx, absDir, "mysql")
	if err != nil {
		db, err = connectDolt(ctx, absDir, "dolt")
		if err != nil {
			return err
		}
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `CREATE DATABASE IF NOT EXISTS fabrik`)
	return err
}

func connectDolt(ctx context.Context, absDir, database string) (*sql.DB, error) {
	dsn, err := doltDSN(absDir, database)
	if err != nil {
		return nil, err
	}
	cfg, err := embedded.ParseDSN(dsn)
	if err != nil {
		return nil, err
	}
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 50 * time.Millisecond
	bo.MaxElapsedTime = 500 * time.Millisecond
	cfg.BackOff = bo
	connector, err := embedded.NewConnector(cfg)
	if err != nil {
		return nil, err
	}
	db := sql.OpenDB(connector)
	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func doltDSN(dir, database string) (string, error) {
	u := &url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(dir),
	}
	q := url.Values{}
	q.Set("commitname", "fabrik")
	q.Set("commitemail", "fabrik@local")
	q.Set("database", database)
	q.Set("multistatements", "true")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func migrate(ctx context.Context, db *sql.DB) (bool, error) {
	return sqlmigrate.ApplyDolt(ctx, db)
}
