package mcpserver

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/export"
	"github.com/ur-wesley/fabrik/cli/internal/graph"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

func openIssueStore(ctx context.Context, projectPath string) (*doltstore.Store, error) {
	start := projectPath
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		start = wd
	}
	root, err := store.RequireRepo(start)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	st, err := doltstore.Open(ctx, root)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	return st, nil
}

func afterIssueWrite(ctx context.Context, st *doltstore.Store, issueID string, publish bool) error {
	if !publish {
		return nil
	}
	if err := export.PublishAfterSave(ctx, st, issueID, "write"); err != nil {
		return mapStoreErr(err)
	}
	return nil
}

func openGraphDB(ctx context.Context, projectPath string) (*graph.DB, error) {
	start := projectPath
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		start = wd
	}
	db, err := graph.Open(ctx, start)
	if err != nil {
		return nil, mapStoreErr(err)
	}
	return db, nil
}

func mapStoreErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, store.ErrNoStore) {
		return errors.New("no store (run fabrik init)")
	}
	if isStoreBusy(err) {
		return errors.New("store busy (another fabrik process holds the lock)")
	}
	var save *export.SaveExportError
	if errors.As(err, &save) {
		return save
	}
	return err
}

func isStoreBusy(err error) bool {
	if errors.Is(err, sql.ErrConnDone) {
		return false
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "database is locked") || strings.Contains(msg, "sqlite_busy") {
		return true
	}
	if strings.Contains(msg, "lock wait timeout") || strings.Contains(msg, "cannot acquire lock") {
		return true
	}
	return false
}
