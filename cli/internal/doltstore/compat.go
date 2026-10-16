package doltstore

import (
	"context"
)

// RepoPath returns the repo root owning this store.
func (s *Store) RepoPath() string {
	if s == nil {
		return ""
	}
	return s.Repo
}

// IssueNotes returns the notes of one issue in display order. It backs the
// export snapshot shared with the SQLite sync path.
func (s *Store) IssueNotes(ctx context.Context, issueID string) ([]Note, error) {
	return s.loadNotes(ctx, issueID)
}

// IssueDeps returns every dependency edge with its kind. It backs the export
// snapshot shared with the SQLite sync path (union by triple).
func (s *Store) IssueDeps(ctx context.Context) ([]Dep, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT blocker_id, blocked_id, kind FROM issue_deps
		ORDER BY blocked_id ASC, blocker_id ASC, kind ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Dep
	for rows.Next() {
		var d Dep
		if err := rows.Scan(&d.BlockerID, &d.BlockedID, &d.Kind); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	if out == nil {
		out = []Dep{}
	}
	return out, rows.Err()
}
