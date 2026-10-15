//go:build cgo

package sqlmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/dolthub/driver"
)

func ApplyDolt(ctx context.Context, db *sql.DB) (bool, error) {
	files, err := listMigrations("dolt")
	if err != nil {
		return false, err
	}
	maxBin := 0
	if len(files) > 0 {
		maxBin = files[len(files)-1].Version
	}

	have, err := currentVersion(ctx, db)
	if err != nil {
		return false, err
	}
	if have > maxBin {
		return false, &ErrSchemaNewer{Store: "dolt", Have: have, Binary: maxBin}
	}

	var pending []migrationFile
	for _, f := range files {
		if f.Version > have {
			pending = append(pending, f)
		}
	}
	if len(pending) == 0 {
		return false, nil
	}

	for _, f := range pending {
		if err := applyOneDolt(ctx, db, f); err != nil {
			return false, fmt.Errorf("sqlmigrate: %s: %w", f.Filename, err)
		}
	}
	return true, nil
}

func applyOneDolt(ctx context.Context, db *sql.DB, f migrationFile) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, stmt := range splitSQLStatements(f.SQL) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, filename, applied_at) VALUES (?, ?, ?)`,
		f.Version, f.Filename, AppliedAt(),
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	msg := fmt.Sprintf("schema migration %s", f.Filename)
	_, err = db.ExecContext(ctx, `CALL DOLT_COMMIT('-am', ?)`, msg)
	if err != nil && !strings.Contains(err.Error(), "nothing to commit") {
		return err
	}
	return nil
}

func splitSQLStatements(sql string) []string {
	var out []string
	for _, part := range strings.Split(sql, ";") {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		out = append(out, stmt)
	}
	return out
}
