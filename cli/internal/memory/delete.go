package memory

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ur-wesley/fabrik/cli/internal/listen"
)

func Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("fabrik: observation id is required")
	}
	db, err := ensureDB(ctx)
	if err != nil {
		return err
	}
	defer releaseDB(db)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	deleted := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.ExecContext(ctx, `
		UPDATE observations SET deleted_at = ? WHERE id = ? AND deleted_at IS NULL`, deleted, id)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		_ = tx.Rollback()
		return errors.New("fabrik: observation not found")
	}
	if err := removeFTS(ctx, tx, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	listen.SignalMemory()
	return nil
}
