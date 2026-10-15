package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type TimelineOptions struct {
	ID      string
	Limit   int
	Project string
}

type TimelineResult struct {
	ID         string        `json:"id"`
	Before     []Observation `json:"before"`
	After      []Observation `json:"after"`
	Observation Observation  `json:"observation"`
}

func Timeline(ctx context.Context, opt TimelineOptions) (TimelineResult, error) {
	id := strings.TrimSpace(opt.ID)
	if id == "" {
		return TimelineResult{}, errors.New("fabrik: observation id is required")
	}
	limit := opt.Limit
	if limit <= 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	o, err := Get(ctx, id)
	if err != nil {
		return TimelineResult{}, err
	}

	db, _, err := openDB(ctx, false)
	if err != nil {
		return TimelineResult{}, err
	}
	if db == nil {
		return TimelineResult{}, fmt.Errorf("fabrik: observation %s not found", id)
	}
	defer releaseDB(db)

	project := strings.TrimSpace(opt.Project)
	before, err := timelineSide(ctx, db, o.CreatedAt, project, true, limit)
	if err != nil {
		return TimelineResult{}, err
	}
	after, err := timelineSide(ctx, db, o.CreatedAt, project, false, limit)
	if err != nil {
		return TimelineResult{}, err
	}
	return TimelineResult{ID: id, Observation: o, Before: before, After: after}, nil
}

func timelineSide(ctx context.Context, db *sql.DB, createdAt, project string, before bool, limit int) ([]Observation, error) {
	q := `
		SELECT id, session_id, project, kind, title, body, topic_key, symbol_id, created_at
		FROM observations WHERE deleted_at IS NULL`
	args := []interface{}{}
	if project != "" {
		q += ` AND project = ?`
		args = append(args, project)
	}
	if before {
		q += ` AND created_at < ? ORDER BY created_at DESC LIMIT ?`
	} else {
		q += ` AND created_at > ? ORDER BY created_at ASC LIMIT ?`
	}
	args = append(args, createdAt, limit)

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	var out []Observation
	defer rows.Close()
	for rows.Next() {
		o, err := scanObservationRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func scanObservationRows(rows *sql.Rows) (Observation, error) {
	var o Observation
	var sess, topic, sym sql.NullString
	if err := rows.Scan(&o.ID, &sess, &o.Project, &o.Kind, &o.Title, &o.Body, &topic, &sym, &o.CreatedAt); err != nil {
		return Observation{}, err
	}
	if sess.Valid {
		o.SessionID = &sess.String
	}
	if topic.Valid {
		o.TopicKey = &topic.String
	}
	if sym.Valid {
		o.SymbolID = &sym.String
	}
	return o, nil
}
