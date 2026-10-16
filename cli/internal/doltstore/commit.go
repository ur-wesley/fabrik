package doltstore

import (
	"context"
	"database/sql"
)

// doltCommit is a no-op: the SQLite issue store is durable without a
// version-control commit step. Cross-machine history lives in git
// (see vc.go); per-write commits were a Dolt concept with no SQLite
// equivalent.
func doltCommit(_ context.Context, _ *sql.DB, _ string) error {
	return nil
}
