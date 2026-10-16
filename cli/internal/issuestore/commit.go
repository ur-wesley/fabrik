package issuestore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
)

// doltCommit is a no-op kept for API parity with doltstore: SQLite issue
// writes need no Dolt version-control commit. The signature is unchanged so
// the CRUD below ports line-for-line.
func doltCommit(_ context.Context, _ *sql.DB, _ string) error {
	return nil
}

// StoreErr mirrors doltstore.StoreErr.
func StoreErr(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*sqlmigrate.ErrSchemaNewer); ok {
		return err
	}
	return fmt.Errorf("fabrik: store: %w", err)
}
