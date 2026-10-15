package sqlmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/ur-wesley/fabrik/cli/internal/fileutil"
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
		if err := os.MkdirAll(filepath.Dir(dbPath), fileutil.DirPerm); err != nil {
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

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return false, err
	}

	have, err := currentVersion(ctx, db)
	if err != nil {
		_ = db.Close()
		return false, err
	}
	if have > maxBin {
		_ = db.Close()
		return false, &ErrSchemaNewer{Store: kind, Have: have, Binary: maxBin}
	}

	var pending []migrationFile
	for _, f := range files {
		if f.Version > have {
			pending = append(pending, f)
		}
	}
	if len(pending) == 0 {
		_ = db.Close()
		return false, nil
	}

	if !needCreate {
		// WAL safety: checkpoint so all committed frames are in the main
		// file, then CLOSE the pool before touching files. Copying or
		// renaming a live WAL database copies a moving target and strands
		// -wal/-shm state.
		if _, err := db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			_ = db.Close()
			return false, err
		}
		if err := db.Close(); err != nil {
			return false, err
		}
		if err := backupSQLiteFiles(dbPath, backupPath); err != nil {
			return false, err
		}
		backedUp = true
		db, err = sql.Open("sqlite", sqliteutil.DSN(dbPath))
		if err != nil {
			restoreSQLiteFiles(dbPath, backupPath)
			return false, err
		}
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			restoreSQLiteFiles(dbPath, backupPath)
			return false, err
		}
	}

	for _, f := range pending {
		if err := applyOneSQLite(ctx, db, f); err != nil {
			_ = db.Close()
			if backedUp {
				restoreSQLiteFiles(dbPath, backupPath)
			}
			return false, fmt.Errorf("sqlmigrate: %s: %w", f.Filename, err)
		}
	}
	_ = db.Close()
	if backedUp {
		removeSQLiteBackup(backupPath)
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

// sqliteSidecars lists the WAL sidecar suffixes shadowing a main db file.
var sqliteSidecars = []string{"-wal", "-shm"}

// backupSQLiteFiles copies the main database file plus any -wal/-shm
// sidecars. Call only with no open handle on dbPath (after checkpoint+close)
// so the copy is a consistent snapshot.
func backupSQLiteFiles(dbPath, backupPath string) error {
	if err := fileutil.CopyFile(dbPath, backupPath, fileutil.SecretPerm); err != nil {
		return err
	}
	for _, suf := range sqliteSidecars {
		if _, err := os.Stat(dbPath + suf); err != nil {
			continue
		}
		if err := fileutil.CopyFile(dbPath+suf, backupPath+suf, fileutil.SecretPerm); err != nil {
			removeSQLiteBackup(backupPath)
			return err
		}
	}
	return nil
}

// restoreSQLiteFiles swaps the backup (main + sidecars) back over dbPath,
// discarding the failed-migration state first. Call only with no open handle
// on dbPath.
func restoreSQLiteFiles(dbPath, backupPath string) {
	_ = os.Remove(dbPath)
	for _, suf := range sqliteSidecars {
		_ = os.Remove(dbPath + suf)
	}
	_ = os.Rename(backupPath, dbPath)
	for _, suf := range sqliteSidecars {
		if _, err := os.Stat(backupPath + suf); err == nil {
			_ = os.Rename(backupPath+suf, dbPath+suf)
		}
	}
}

func removeSQLiteBackup(backupPath string) {
	_ = os.Remove(backupPath)
	for _, suf := range sqliteSidecars {
		_ = os.Remove(backupPath + suf)
	}
}
