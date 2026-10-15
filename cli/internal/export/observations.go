package export

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ur-wesley/fabrik/cli/internal/fileutil"
	"github.com/ur-wesley/fabrik/cli/internal/memory"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

type ObservationsResult struct {
	Count int
	Path  string
}

func Observations(ctx context.Context) (ObservationsResult, error) {
	db, err := memory.OpenReadOnly(ctx)
	if err != nil {
		return ObservationsResult{}, err
	}
	if db == nil {
		dir, err := store.FabrikDir()
		if err != nil {
			return ObservationsResult{}, err
		}
		path := filepath.Join(dir, "export", "observations.jsonl")
		if err := os.MkdirAll(filepath.Dir(path), fileutil.DirPerm); err != nil {
			return ObservationsResult{}, err
		}
		f, err := os.Create(path)
		if err != nil {
			return ObservationsResult{}, err
		}
		_ = f.Close()
		return ObservationsResult{Count: 0, Path: path}, nil
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, `
		SELECT id, session_id, project, kind, title, body, topic_key, symbol_id, created_at
		FROM observations ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return ObservationsResult{}, err
	}
	defer rows.Close()

	dir, err := store.FabrikDir()
	if err != nil {
		return ObservationsResult{}, err
	}
	exportDir := filepath.Join(dir, "export")
	if err := os.MkdirAll(exportDir, fileutil.DirPerm); err != nil {
		return ObservationsResult{}, err
	}
	path := filepath.Join(exportDir, "observations.jsonl")
	f, err := os.Create(path)
	if err != nil {
		return ObservationsResult{}, err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	count := 0
	for rows.Next() {
		var o memory.Observation
		var sessionID, topicKey, symbolID *string
		if err := rows.Scan(&o.ID, &sessionID, &o.Project, &o.Kind, &o.Title, &o.Body, &topicKey, &symbolID, &o.CreatedAt); err != nil {
			return ObservationsResult{}, err
		}
		o.SessionID = sessionID
		o.TopicKey = topicKey
		o.SymbolID = symbolID
		if err := enc.Encode(o); err != nil {
			return ObservationsResult{}, err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return ObservationsResult{}, err
	}
	return ObservationsResult{Count: count, Path: path}, nil
}
