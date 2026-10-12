package memory

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"

	_ "modernc.org/sqlite"

	"github.com/ur-wesley/fabrik/cli/internal/listen"
	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/sqliteutil"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

var (
	sharedPool   bool
	sharedDB     *sql.DB
	sharedDBOnce sync.Once
	sharedDBErr  error
)

func EnableSharedPool(v bool) {
	sharedPool = v
}

func NormalizeActor(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return "default"
	}
	return actor
}

func openDB(ctx context.Context, create bool) (*sql.DB, string, error) {
	path, err := store.MemoryDBPath()
	if err != nil {
		return nil, "", err
	}
	if !create {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, path, nil
		} else if err != nil {
			return nil, "", err
		}
	}
	changed, err := sqlmigrate.ApplySQLite(ctx, "memory", path)
	if err != nil {
		return nil, "", err
	}
	if changed {
		listen.SignalMemory()
	}
	if sharedPool {
		sharedDBOnce.Do(func() {
			sharedDB, sharedDBErr = sql.Open("sqlite", sqliteutil.DSN(path))
			if sharedDBErr == nil {
				sharedDB.SetMaxOpenConns(1)
				sharedDBErr = sharedDB.PingContext(ctx)
			}
		})
		if sharedDBErr != nil {
			return nil, "", sharedDBErr
		}
		return sharedDB, path, nil
	}
	db, err := sql.Open("sqlite", sqliteutil.DSN(path))
	if err != nil {
		return nil, "", err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, "", err
	}
	return db, path, nil
}

func ensureDB(ctx context.Context) (*sql.DB, error) {
	db, _, err := openDB(ctx, true)
	return db, err
}

func releaseDB(db *sql.DB) {
	if sharedPool || db == nil || db == sharedDB {
		return
	}
	_ = db.Close()
}

func OpenReadOnly(ctx context.Context) (*sql.DB, error) {
	db, _, err := openDB(ctx, false)
	if err != nil || db == nil {
		return nil, err
	}
	return db, nil
}
