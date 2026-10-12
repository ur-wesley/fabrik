package memory

import (
	"context"
	"database/sql"
)

func replaceFTS(ctx context.Context, tx *sql.Tx, obsID string, title, body string) error {
	var rowid int64
	if err := tx.QueryRowContext(ctx, `SELECT rowid FROM observations WHERE id = ?`, obsID).Scan(&rowid); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM observations_fts WHERE rowid = ?`, rowid); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO observations_fts (rowid, title, body) VALUES (?, ?, ?)`, rowid, title, body)
	return err
}

func removeFTS(ctx context.Context, tx *sql.Tx, obsID string) error {
	var rowid int64
	if err := tx.QueryRowContext(ctx, `SELECT rowid FROM observations WHERE id = ?`, obsID).Scan(&rowid); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM observations_fts WHERE rowid = ?`, rowid)
	return err
}
