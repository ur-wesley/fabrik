package sqlmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/ur-wesley/fabrik/cli/internal/sqliteutil"
)

// ApplySQLite runs pending migrations for kind (memory or graph) on dbPath.
// The returned bool is true when the database file was created or any migration ran.
func ApplySQLite(ctx context.Context, kind, dbPath string) (bool, error) {
	files, err := listMigrations(kind)
	if err != nil {
		return false, err
	}
	maxBin := 0
	if len(files) > 0 {
		maxBin = files[len(files)-1].Version
	}

	needCreate := false
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		needCreate = true
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	}

	var backedUp bool
	backupPath := dbPath + ".bak"

	db, err := sql.Open("sqlite", sqliteutil.DSN(dbPath))
	if err != nil {
		return false, err
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return false, err
	}

	have, err := currentVersion(ctx, db)
	if err != nil {
		return false, err
	}
	if have > maxBin {
		return false, &ErrSchemaNewer{Store: kind, Have: have, Binary: maxBin}
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

	if !needCreate && !backedUp {
		if err := copyFile(dbPath, backupPath); err != nil {
			return false, err
		}
		backedUp = true
	}

	for _, f := range pending {
		if err := applyOneSQLite(ctx, db, f); err != nil {
			if backedUp {
				_ = db.Close()
				_ = os.Rename(backupPath, dbPath)
			}
			return false, fmt.Errorf("sqlmigrate: %s: %w", f.Filename, err)
		}
	}
	if backedUp {
		_ = os.Remove(backupPath)
	}
	return true, nil
}

// CurrentVersion returns the highest applied schema_migrations version on db.
func CurrentVersion(ctx context.Context, db *sql.DB) (int, error) {
	return currentVersion(ctx, db)
}

func currentVersion(ctx context.Context, db *sql.DB) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM schema_migrations
	`).Scan(&n)
	if err != nil {
		if stringsTableMissing(err) {
			return 0, nil
		}
		return 0, err
	}
	return n, nil
}

func stringsTableMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such table") || strings.Contains(msg, "table not found")
}

func applyOneSQLite(ctx context.Context, db *sql.DB, f migrationFile) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, f.SQL); err != nil {
		_ = tx.Rollback()
		return err
	}
	_, err = tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, filename, applied_at) VALUES (?, ?, ?)`,
		f.Version, f.Filename, AppliedAt(),
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
