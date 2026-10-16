package doltstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type HistoryVersion map[string]interface{}

type HistoryResult struct {
	ID       string           `json:"id"`
	Table    string           `json:"table"`
	Versions []HistoryVersion `json:"versions"`
}

// History returns the current row for an issue (f-) or note (n-) id.
// The SQLite issue store keeps no per-commit version history (that was
// Dolt's dolt_history_* system tables); the shape stays compatible with
// the `fabrik history` text/JSON output, with git log carrying the
// cross-machine timeline (see VCLog).
func (s *Store) History(ctx context.Context, id string) (HistoryResult, error) {
	table, err := historyTable(id)
	if err != nil {
		return HistoryResult{}, err
	}
	cur, err := s.historyCurrent(ctx, id, table)
	if err != nil {
		return HistoryResult{}, err
	}
	return HistoryResult{ID: id, Table: table, Versions: []HistoryVersion{cur}}, nil
}

func (s *Store) historyCurrent(ctx context.Context, id, table string) (HistoryVersion, error) {
	if table == "issues" {
		i, err := s.Get(ctx, id)
		if err != nil {
			if err == ErrNotFound {
				return nil, fmt.Errorf("fabrik: %s not found", id)
			}
			return nil, err
		}
		return HistoryVersion{
			"id": i.ID, "title": i.Title, "status": i.Status, "priority": i.Priority,
		}, nil
	}
	var n Note
	err := s.DB.QueryRowContext(ctx, `SELECT id, issue_id, body, created_at FROM issue_notes WHERE id = ?`, id).
		Scan(&n.ID, &n.IssueID, &n.Body, &n.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fabrik: %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return HistoryVersion{
		"id": n.ID, "issue_id": n.IssueID, "body": n.Body, "created_at": n.CreatedAt,
	}, nil
}

func historyTable(id string) (string, error) {
	if strings.HasPrefix(id, "f-") {
		return "issues", nil
	}
	if strings.HasPrefix(id, "n-") {
		return "issue_notes", nil
	}
	return "", fmt.Errorf("fabrik: history expects f- or n- id")
}

func scanHistoryVal(v interface{}) interface{} {
	switch x := v.(type) {
	case []byte:
		return string(x)
	default:
		return x
	}
}

func HistoryExists(ctx context.Context, db *sql.DB, id string) (bool, error) {
	table, err := historyTable(id)
	if err != nil {
		return false, err
	}
	var n int
	q := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE id = ?`, table)
	if err := db.QueryRowContext(ctx, q, id).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}
