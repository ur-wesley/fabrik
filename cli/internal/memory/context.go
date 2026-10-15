package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ur-wesley/fabrik/cli/internal/store"
)

var ErrProjectNotFound = errors.New("fabrik: project not found in memory")

type ContextOptions struct {
	Project      string
	Limit        int
	AgentDefault bool
	Cwd          string
	Actor        string
	// Scope mirrors the mem_context scope argument (project|personal).
	// v1: both scopes read the same table; personal is reserved.
	// Accepted for spec compatibility; filtering is unchanged.
	Scope string
}

func Context(ctx context.Context, opt ContextOptions) (ContextResult, error) {
	db, _, err := openDB(ctx, false)
	if err != nil {
		return ContextResult{}, err
	}
	if db == nil {
		return ContextResult{Observations: []Observation{}}, nil
	}
	defer releaseDB(db)

	projects, err := listProjects(ctx, db)
	if err != nil {
		return ContextResult{}, err
	}
	if opt.Project != "" {
		found := false
		for _, p := range projects {
			if p == opt.Project {
				found = true
				break
			}
		}
		if !found {
			return ContextResult{}, fmt.Errorf("%w\nknown: %s", ErrProjectNotFound, strings.Join(projects, ", "))
		}
	}

	res := ContextResult{
		Observations: []Observation{},
		Stats:        ContextStats{Projects: projects},
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM observations WHERE deleted_at IS NULL`).Scan(&res.Stats.ObservationCount); err != nil {
		return ContextResult{}, err
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions`).Scan(&res.Stats.SessionCount); err != nil {
		return ContextResult{}, err
	}

	sess, err := loadContextSession(ctx, db, opt.Actor)
	if err != nil {
		return ContextResult{}, err
	}
	res.Session = sess

	limit := opt.Limit
	if limit <= 0 {
		limit = 20
	}
	var obs []Observation
	if opt.AgentDefault && opt.Project == "" {
		obs, err = loadAgentContextObservations(ctx, db, opt.Cwd, limit)
	} else {
		obs, err = loadRecentObservations(ctx, db, opt.Project, limit)
	}
	if err != nil {
		return ContextResult{}, err
	}
	res.Observations = obs
	return res, nil
}

func loadContextSession(ctx context.Context, db *sql.DB, actor string) (*Session, error) {
	actor = NormalizeActor(actor)
	openQ := `
		SELECT id, actor, started_at, ended_at, goal, discoveries, accomplished, next_steps, files
		FROM sessions WHERE ended_at IS NULL AND actor = ? LIMIT 1`
	s, err := scanSessionRow(ctx, db, openQ, actor)
	if err == nil {
		return s, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	s, err = scanSessionRow(ctx, db, `
		SELECT id, actor, started_at, ended_at, goal, discoveries, accomplished, next_steps, files
		FROM sessions ORDER BY started_at DESC LIMIT 1`)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func loadAgentContextObservations(ctx context.Context, db *sql.DB, cwd string, limit int) ([]Observation, error) {
	cwdProject := ""
	if root, ok := store.RepoRoot(cwd); ok {
		cwdProject = root
	}
	full, err := loadRecentObservations(ctx, db, cwdProject, limit)
	if err != nil {
		return nil, err
	}
	if cwdProject == "" {
		return full, nil
	}
	peek, err := loadRecentObservations(ctx, db, "", limit*3)
	if err != nil {
		return full, nil
	}
	seen := map[string]bool{}
	for _, o := range full {
		seen[o.ID] = true
	}
	for _, o := range peek {
		if o.Project == "" || o.Project == cwdProject || seen[o.ID] {
			continue
		}
		seen[o.ID] = true
		o.Body = ""
		full = append(full, o)
		if len(full) >= limit {
			break
		}
	}
	return full, nil
}

func scanSessionRow(ctx context.Context, db *sql.DB, q string, args ...any) (*Session, error) {
	var s Session
	var ended sql.NullString
	err := db.QueryRowContext(ctx, q, args...).Scan(
		&s.ID, &s.Actor, &s.StartedAt, &ended, &s.Goal, &s.Discoveries, &s.Accomplished, &s.NextSteps, &s.Files,
	)
	if err != nil {
		return nil, err
	}
	if ended.Valid {
		s.EndedAt = &ended.String
	}
	return &s, nil
}

func loadRecentObservations(ctx context.Context, db *sql.DB, project string, limit int) ([]Observation, error) {
	q := `
		SELECT id, session_id, project, kind, title, body, symbol_id, created_at
		FROM observations WHERE deleted_at IS NULL`
	args := []any{}
	if project != "" {
		q += ` AND project = ?`
		args = append(args, project)
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Observation
	for rows.Next() {
		o, err := scanObservation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func scanObservation(rows *sql.Rows) (Observation, error) {
	var o Observation
	var sess, sym sql.NullString
	if err := rows.Scan(&o.ID, &sess, &o.Project, &o.Kind, &o.Title, &o.Body, &sym, &o.CreatedAt); err != nil {
		return Observation{}, err
	}
	if sess.Valid {
		o.SessionID = &sess.String
	}
	if sym.Valid {
		o.SymbolID = &sym.String
	}
	return o, nil
}

func listProjects(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT project FROM observations WHERE project != '' ORDER BY project`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var projects []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func PrintContext(w io.Writer, r ContextResult) {
	if r.Session != nil {
		state := "open"
		if r.Session.EndedAt != nil {
			state = "ended"
		}
		_, _ = fmt.Fprintf(w, "session %s (%s)\n", r.Session.ID, state)
		if r.Session.Goal != "" {
			_, _ = fmt.Fprintf(w, "goal: %s\n", r.Session.Goal)
		}
		if r.Session.NextSteps != "" {
			_, _ = fmt.Fprintf(w, "next: %s\n", r.Session.NextSteps)
		}
		_, _ = fmt.Fprintln(w)
	}
	if len(r.Observations) == 0 {
		_, _ = fmt.Fprintln(w, "no observations")
		return
	}
	for _, o := range r.Observations {
		_, _ = fmt.Fprintf(w, "[%s] %s\n", o.Kind, o.Title)
		if o.Project != "" {
			_, _ = fmt.Fprintf(w, "project: %s\n", o.Project)
		}
		_, _ = fmt.Fprintf(w, "%s\n\n", o.Body)
	}
}

func PrintContextJSON(w io.Writer, r ContextResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
