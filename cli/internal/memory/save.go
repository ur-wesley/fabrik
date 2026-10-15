package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ur-wesley/fabrik/cli/internal/listen"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

var (
	validKinds = map[string]bool{
		"decision": true, "bugfix": true, "pattern": true, "discovery": true, "handoff": true, "prompt": true,
	}
)

type SaveOptions struct {
	Title         string
	Body          string
	Kind          string
	TopicKey      string
	CapturePrompt string
	SymbolID      string
	Cwd           string
}

func Save(ctx context.Context, opt SaveOptions) (Observation, error) {
	title := strings.TrimSpace(opt.Title)
	body := strings.TrimSpace(opt.Body)
	if cap := strings.TrimSpace(opt.CapturePrompt); cap != "" && body == "" {
		body = cap
	}
	if title == "" {
		return Observation{}, errors.New("fabrik: --title is required")
	}
	if body == "" {
		return Observation{}, errors.New("fabrik: --body is required")
	}
	if utf8.RuneCountInString(title) > 200 {
		return Observation{}, errors.New("fabrik: --title must be at most 200 runes")
	}
	if utf8.RuneCountInString(body) > 16000 {
		return Observation{}, errors.New("fabrik: --body must be at most 16000 runes")
	}
	kind := opt.Kind
	if kind == "" {
		kind = "discovery"
	}
	if !validKinds[kind] {
		return Observation{}, fmt.Errorf("fabrik: invalid --kind %q", kind)
	}

	db, err := ensureDB(ctx)
	if err != nil {
		return Observation{}, err
	}
	defer releaseDB(db)

	id, err := store.NewHashID("o-")
	if err != nil {
		return Observation{}, err
	}
	created := time.Now().UTC().Format(time.RFC3339)

	project := ""
	cwd := opt.Cwd
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			return Observation{}, err
		}
	}
	if root, ok := store.RepoRoot(cwd); ok {
		project = root
	}

	var sessionID sql.NullString
	var sid string
	err = db.QueryRowContext(ctx, `SELECT id FROM sessions WHERE ended_at IS NULL LIMIT 1`).Scan(&sid)
	if err == nil {
		sessionID = sql.NullString{String: sid, Valid: true}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Observation{}, err
	}

	var sym sql.NullString
	if opt.SymbolID != "" {
		sym = sql.NullString{String: opt.SymbolID, Valid: true}
	}
	var topic sql.NullString
	if opt.TopicKey != "" {
		topic = sql.NullString{String: strings.TrimSpace(opt.TopicKey), Valid: true}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Observation{}, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO observations (id, session_id, project, kind, title, body, topic_key, symbol_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, sessionID, project, kind, title, body, topic, sym, created,
	)
	if err != nil {
		_ = tx.Rollback()
		return Observation{}, err
	}
	var rowid int64
	if err := tx.QueryRowContext(ctx, `SELECT rowid FROM observations WHERE id = ?`, id).Scan(&rowid); err != nil {
		_ = tx.Rollback()
		return Observation{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO observations_fts (rowid, title, body) VALUES (?, ?, ?)`, rowid, title, body); err != nil {
		_ = tx.Rollback()
		return Observation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Observation{}, err
	}

	listen.SignalMemory()

	o := Observation{
		ID: id, Project: project, Kind: kind, Title: title, Body: body, CreatedAt: created,
	}
	if sessionID.Valid {
		o.SessionID = &sessionID.String
	}
	if sym.Valid {
		o.SymbolID = &sym.String
	}
	if topic.Valid {
		o.TopicKey = &topic.String
	}
	return o, nil
}

func PrintObservationJSON(w io.Writer, o Observation) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(o)
}
