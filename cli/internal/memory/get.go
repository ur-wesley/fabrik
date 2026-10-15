package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func Get(ctx context.Context, id string) (Observation, error) {
	db, _, err := openDB(ctx, false)
	if err != nil {
		return Observation{}, err
	}
	if db == nil {
		return Observation{}, fmt.Errorf("fabrik: observation %s not found", id)
	}
	defer releaseDB(db)

	row := db.QueryRowContext(ctx, `
		SELECT id, session_id, project, kind, title, body, topic_key, symbol_id, created_at
		FROM observations WHERE id = ? AND deleted_at IS NULL`, id)

	o, err := scanObservationRow(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Observation{}, fmt.Errorf("fabrik: observation %s not found", id)
	}
	if err != nil {
		return Observation{}, err
	}
	return o, nil
}

func scanObservationRow(row *sql.Row) (Observation, error) {
	var o Observation
	var sess, topic, sym sql.NullString
	if err := row.Scan(&o.ID, &sess, &o.Project, &o.Kind, &o.Title, &o.Body, &topic, &sym, &o.CreatedAt); err != nil {
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

func PrintGetJSON(w io.Writer, o Observation) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(o)
}

func PrintGet(w io.Writer, o Observation) {
	_, _ = fmt.Fprintf(w, "[%s] %s\n", o.Kind, o.Title)
	if o.Project != "" {
		_, _ = fmt.Fprintf(w, "project: %s\n", o.Project)
	}
	_, _ = fmt.Fprintf(w, "%s\n", o.Body)
}
