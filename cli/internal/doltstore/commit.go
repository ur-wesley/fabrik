package doltstore

import (
	"context"
	"database/sql"
	"strings"
)

func doltCommit(ctx context.Context, db *sql.DB, message string) error {
	_, _ = db.ExecContext(ctx, `CALL DOLT_ADD('-A')`)
	_, err := db.ExecContext(ctx, `CALL DOLT_COMMIT('-am', ?)`, message)
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "nothing to commit") {
		return nil
	}
	return err
}
