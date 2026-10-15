package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ur-wesley/fabrik/cli/internal/listen"
)

type UpdateOptions struct {
	ID       string
	Title    *string
	Body     *string
	Kind     *string
	TopicKey *string
	SymbolID *string
}

func Update(ctx context.Context, opt UpdateOptions) (Observation, error) {
	id := strings.TrimSpace(opt.ID)
	if id == "" {
		return Observation{}, errors.New("fabrik: observation id is required")
	}
	db, err := ensureDB(ctx)
	if err != nil {
		return Observation{}, err
	}
	defer releaseDB(db)

	cur, err := Get(ctx, id)
	if err != nil {
		return Observation{}, err
	}
	title := cur.Title
	body := cur.Body
	kind := cur.Kind
	if opt.Title != nil {
		title = strings.TrimSpace(*opt.Title)
		if title == "" {
			return Observation{}, errors.New("fabrik: title cannot be empty")
		}
		if utf8.RuneCountInString(title) > 200 {
			return Observation{}, errors.New("fabrik: title must be at most 200 runes")
		}
	}
	if opt.Body != nil {
		body = strings.TrimSpace(*opt.Body)
		if body == "" {
			return Observation{}, errors.New("fabrik: body cannot be empty")
		}
		if utf8.RuneCountInString(body) > 16000 {
			return Observation{}, errors.New("fabrik: body must be at most 16000 runes")
		}
	}
	if opt.Kind != nil {
		kind = *opt.Kind
		if !validKinds[kind] {
			return Observation{}, fmt.Errorf("fabrik: invalid kind %q", kind)
		}
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return Observation{}, err
	}
	var topic sql.NullString
	if opt.TopicKey != nil {
		t := strings.TrimSpace(*opt.TopicKey)
		if t != "" {
			topic = sql.NullString{String: t, Valid: true}
		}
	} else if cur.TopicKey != nil {
		topic = sql.NullString{String: *cur.TopicKey, Valid: true}
	}
	var sym sql.NullString
	if opt.SymbolID != nil {
		s := strings.TrimSpace(*opt.SymbolID)
		if s != "" {
			sym = sql.NullString{String: s, Valid: true}
		}
	} else if cur.SymbolID != nil {
		sym = sql.NullString{String: *cur.SymbolID, Valid: true}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE observations SET title = ?, body = ?, kind = ?, topic_key = ?, symbol_id = ?
		WHERE id = ? AND deleted_at IS NULL`,
		title, body, kind, topic, sym, id,
	)
	if err != nil {
		_ = tx.Rollback()
		return Observation{}, err
	}
	if err := replaceFTS(ctx, tx, id, title, body); err != nil {
		_ = tx.Rollback()
		return Observation{}, err
	}
	if err := tx.Commit(); err != nil {
		return Observation{}, err
	}
	listen.SignalMemory()
	return Get(ctx, id)
}
