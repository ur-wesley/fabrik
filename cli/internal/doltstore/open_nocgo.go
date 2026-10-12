//go:build !cgo

package doltstore

import (
	"context"
	"database/sql"
	"errors"
)

const HasCGO = false

func openDB(ctx context.Context, doltDir string) (*sql.DB, error) {
	return nil, errors.New("embedded Dolt requires CGO_ENABLED=1")
}

func migrate(ctx context.Context, db *sql.DB) (bool, error) {
	return false, errors.New("embedded Dolt requires CGO_ENABLED=1")
}
