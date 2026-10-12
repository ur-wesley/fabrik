//go:build !cgo

package sqlmigrate

import (
	"context"
	"database/sql"
	"errors"
)

func ApplyDolt(ctx context.Context, db *sql.DB) (bool, error) {
	return false, errors.New("dolt migrations require CGO_ENABLED=1")
}
