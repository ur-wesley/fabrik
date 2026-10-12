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

func (s *Store) History(ctx context.Context, id string) (HistoryResult, error) {
	table, err := historyTable(id)
	if err != nil {
		return HistoryResult{}, err
	}
	q := fmt.Sprintf(`SELECT * FROM dolt_history_%s WHERE id = ? ORDER BY commit_date ASC`, table)
	rows, err := s.DB.QueryContext(ctx, q, id)
	if err != nil {
		return HistoryResult{}, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return HistoryResult{}, err
	}
	var versions []HistoryVersion
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return HistoryResult{}, err
		}
		ver := HistoryVersion{}
		for i, c := range cols {
			ver[c] = scanHistoryVal(vals[i])
		}
		versions = append(versions, ver)
	}
	if err := rows.Err(); err != nil {
		return HistoryResult{}, err
	}
	if len(versions) == 0 {
		cur, err := s.historyFallback(ctx, id, table)
		if err != nil {
			return HistoryResult{}, err
		}
		versions = []HistoryVersion{cur}
	}
	return HistoryResult{ID: id, Table: table, Versions: versions}, nil
}

func (s *Store) historyFallback(ctx context.Context, id, table string) (HistoryVersion, error) {
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
	return nil, fmt.Errorf("fabrik: %s not found", id)
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
	case nil:
		return nil
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
	err = db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM dolt_history_%s WHERE id = ?`, table), id).Scan(&n)
	return n > 0, err
}
