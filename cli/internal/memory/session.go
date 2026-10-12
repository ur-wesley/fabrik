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

	"github.com/ur-wesley/fabrik/cli/internal/clierr"
	"github.com/ur-wesley/fabrik/cli/internal/listen"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

var ErrNoOpenSession = errors.New("fabrik: no open session")

type SessionStartResult struct {
	Session
	Created bool `json:"created"`
}

type SessionSummaryOptions struct {
	Goal         string
	Discoveries  string
	Accomplished string
	NextSteps    string
	Files        string
	GoalSet      bool
	DiscoveriesSet      bool
	AccomplishedSet     bool
	NextStepsSet        bool
	FilesSet            bool
	Cwd                 string
	Actor               string
}

func SessionStart(ctx context.Context, actor string) (SessionStartResult, error) {
	actor = NormalizeActor(actor)
	db, err := ensureDB(ctx)
	if err != nil {
		return SessionStartResult{}, err
	}
	defer releaseDB(db)

	s, err := scanSessionRow(ctx, db, `
		SELECT id, actor, started_at, ended_at, goal, discoveries, accomplished, next_steps, files
		FROM sessions WHERE ended_at IS NULL AND actor = ? LIMIT 1`, actor)
	if err == nil {
		return SessionStartResult{Session: *s, Created: false}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return SessionStartResult{}, err
	}

	id, err := store.NewHashID("s-")
	if err != nil {
		return SessionStartResult{}, err
	}
	started := time.Now().UTC().Format(time.RFC3339)
	_, err = db.ExecContext(ctx, `INSERT INTO sessions (id, actor, started_at) VALUES (?, ?, ?)`, id, actor, started)
	if err != nil {
		return SessionStartResult{}, err
	}
	listen.SignalMemory()
	return SessionStartResult{
		Session: Session{ID: id, Actor: actor, StartedAt: started},
		Created: true,
	}, nil
}

func SessionSummary(ctx context.Context, opt SessionSummaryOptions) (Session, error) {
	actor := NormalizeActor(opt.Actor)
	if !opt.GoalSet && !opt.DiscoveriesSet && !opt.AccomplishedSet && !opt.NextStepsSet && !opt.FilesSet {
		return Session{}, clierr.Usage("fabrik: at least one of --goal, --discoveries, --accomplished, --next, or --files is required")
	}

	db, err := ensureDB(ctx)
	if err != nil {
		return Session{}, err
	}
	defer releaseDB(db)

	s, err := scanSessionRow(ctx, db, `
		SELECT id, actor, started_at, ended_at, goal, discoveries, accomplished, next_steps, files
		FROM sessions WHERE ended_at IS NULL AND actor = ? LIMIT 1`, actor)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNoOpenSession
	}
	if err != nil {
		return Session{}, err
	}

	if opt.GoalSet {
		s.Goal = opt.Goal
	}
	if opt.DiscoveriesSet {
		s.Discoveries = opt.Discoveries
	}
	if opt.AccomplishedSet {
		s.Accomplished = opt.Accomplished
	}
	if opt.NextStepsSet {
		s.NextSteps = opt.NextSteps
	}
	if opt.FilesSet {
		s.Files = opt.Files
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Session{}, err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE sessions SET goal = ?, discoveries = ?, accomplished = ?, next_steps = ?, files = ?
		WHERE id = ?`,
		s.Goal, s.Discoveries, s.Accomplished, s.NextSteps, s.Files, s.ID,
	)
	if err != nil {
		_ = tx.Rollback()
		return Session{}, err
	}

	title := "Session handoff"
	if strings.TrimSpace(s.Goal) != "" {
		title = strings.TrimSpace(s.Goal)
	}
	body := formatHandoffBody(*s)
	obsID, err := store.NewHashID("o-")
	if err != nil {
		_ = tx.Rollback()
		return Session{}, err
	}
	created := time.Now().UTC().Format(time.RFC3339)

	project := ""
	cwd := opt.Cwd
	if cwd == "" {
		cwd, err = os.Getwd()
		if err != nil {
			_ = tx.Rollback()
			return Session{}, err
		}
	}
	if root, ok := store.RepoRoot(cwd); ok {
		project = root
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO observations (id, session_id, project, kind, title, body, created_at)
		VALUES (?, ?, ?, 'handoff', ?, ?, ?)`,
		obsID, s.ID, project, title, body, created,
	)
	if err != nil {
		_ = tx.Rollback()
		return Session{}, err
	}
	var rowid int64
	if err := tx.QueryRowContext(ctx, `SELECT rowid FROM observations WHERE id = ?`, obsID).Scan(&rowid); err != nil {
		_ = tx.Rollback()
		return Session{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO observations_fts (rowid, title, body) VALUES (?, ?, ?)`, rowid, title, body); err != nil {
		_ = tx.Rollback()
		return Session{}, err
	}
	if err := tx.Commit(); err != nil {
		return Session{}, err
	}

	listen.SignalMemory()
	return *s, nil
}

func SessionEnd(ctx context.Context, actor string) (Session, error) {
	actor = NormalizeActor(actor)
	db, err := ensureDB(ctx)
	if err != nil {
		return Session{}, err
	}
	defer releaseDB(db)

	s, err := scanSessionRow(ctx, db, `
		SELECT id, actor, started_at, ended_at, goal, discoveries, accomplished, next_steps, files
		FROM sessions WHERE ended_at IS NULL AND actor = ? LIMIT 1`, actor)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrNoOpenSession
	}
	if err != nil {
		return Session{}, err
	}

	ended := time.Now().UTC().Format(time.RFC3339)
	_, err = db.ExecContext(ctx, `UPDATE sessions SET ended_at = ? WHERE id = ?`, ended, s.ID)
	if err != nil {
		return Session{}, err
	}
	s.EndedAt = &ended
	listen.SignalMemory()
	return *s, nil
}

func formatHandoffBody(s Session) string {
	var b strings.Builder
	if s.Goal != "" {
		b.WriteString("Goal: ")
		b.WriteString(s.Goal)
		b.WriteByte('\n')
	}
	if s.Discoveries != "" {
		b.WriteString("Discoveries: ")
		b.WriteString(s.Discoveries)
		b.WriteByte('\n')
	}
	if s.Accomplished != "" {
		b.WriteString("Accomplished: ")
		b.WriteString(s.Accomplished)
		b.WriteByte('\n')
	}
	if s.NextSteps != "" {
		b.WriteString("Next: ")
		b.WriteString(s.NextSteps)
		b.WriteByte('\n')
	}
	if s.Files != "" {
		b.WriteString("Files:\n")
		b.WriteString(s.Files)
		if !strings.HasSuffix(s.Files, "\n") {
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func PrintSessionStartJSON(w io.Writer, r SessionStartResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func PrintSessionJSON(w io.Writer, s Session) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

func PrintSessionStart(w io.Writer, r SessionStartResult) {
	state := "existing"
	if r.Created {
		state = "created"
	}
	_, _ = fmt.Fprintf(w, "session %s (%s)\n", r.ID, state)
}

func PrintSession(w io.Writer, s Session) {
	state := "open"
	if s.EndedAt != nil {
		state = "closed"
	}
	_, _ = fmt.Fprintf(w, "session %s (%s)\n", s.ID, state)
}
