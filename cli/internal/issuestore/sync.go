package issuestore

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// ImportResult counts one ImportRecords run. Issues carry their own
// already-present count (losing last-write-wins comparisons); deps and notes
// likewise; skipped counts rows outside the valid id shapes.
type ImportResult struct {
	Issues        int `json:"issues"`
	IssuesPresent int `json:"issues_present"`
	Deps          int `json:"deps"`
	DepsPresent   int `json:"deps_present"`
	Notes         int `json:"notes"`
	NotesPresent  int `json:"notes_present"`
}

// ImportFile merges a .fabrik/export/issues.jsonl snapshot into the store.
// Transport is git (the file is git-tracked); merge rules are:
//   - issues: upsert by id, last-write-wins on max(updated_at)
//   - deps: union by (blocker_id, blocked_id, kind) triple
//   - notes: union by id
func (s *Store) ImportFile(ctx context.Context, path string) (ImportResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return ImportResult{}, err
	}
	defer f.Close()
	var records []Record
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" {
			continue
		}
		var rec Record
		if err := json.Unmarshal([]byte(raw), &rec); err != nil {
			return ImportResult{}, fmt.Errorf("fabrik: import %s line %d: %w", path, line, err)
		}
		if strings.TrimSpace(rec.ID) == "" {
			return ImportResult{}, fmt.Errorf("fabrik: import %s line %d: record without id", path, line)
		}
		records = append(records, rec)
	}
	if err := sc.Err(); err != nil {
		return ImportResult{}, err
	}
	return s.ImportRecords(ctx, records)
}

// ImportRecords merges decoded snapshot records with the same rules as
// ImportFile.
func (s *Store) ImportRecords(ctx context.Context, records []Record) (ImportResult, error) {
	var res ImportResult
	now := nowRFC3339()
	for _, rec := range records {
		applied, err := s.importIssue(ctx, rec)
		if err != nil {
			return res, err
		}
		if applied {
			res.Issues++
		} else {
			res.IssuesPresent++
		}
	}
	for _, rec := range records {
		for _, d := range rec.Deps {
			blocker := strings.TrimSpace(d.BlockerID)
			blocked := strings.TrimSpace(d.BlockedID)
			kind := strings.TrimSpace(d.Kind)
			if kind == "" {
				kind = "blocks"
			}
			if blocker == "" || blocked == "" {
				continue
			}
			var have int
			if err := s.DB.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM issue_deps WHERE blocker_id = ? AND blocked_id = ? AND kind = ?`,
				blocker, blocked, kind).Scan(&have); err != nil {
				return res, err
			}
			if have > 0 {
				res.DepsPresent++
				continue
			}
			if _, err := s.DB.ExecContext(ctx, `
				INSERT INTO issue_deps (blocker_id, blocked_id, kind) VALUES (?, ?, ?)`,
				blocker, blocked, kind); err != nil {
				return res, fmt.Errorf("fabrik: import dep %s -> %s: %w", blocker, blocked, err)
			}
			res.Deps++
			if kind == "blocks" {
				if err := applyImportedBlock(ctx, s.DB, blocked, blocker, now); err != nil {
					return res, err
				}
			}
		}
	}
	for _, rec := range records {
		for _, n := range rec.Notes {
			if strings.TrimSpace(n.Body) == "" || strings.TrimSpace(n.ID) == "" {
				continue
			}
			issueID := strings.TrimSpace(n.IssueID)
			if issueID == "" {
				issueID = rec.ID
			}
			var have string
			if err := s.DB.QueryRowContext(ctx, `SELECT id FROM issue_notes WHERE id = ?`, n.ID).Scan(&have); err == nil {
				res.NotesPresent++
				continue
			}
			when := strings.TrimSpace(n.CreatedAt)
			if when == "" {
				when = now
			}
			seq, err := s.nextSeq(ctx)
			if err != nil {
				return res, err
			}
			if _, err := s.DB.ExecContext(ctx, `
				INSERT INTO issue_notes (id, issue_id, body, created_at, seq) VALUES (?, ?, ?, ?, ?)`,
				n.ID, issueID, n.Body, when, seq); err != nil {
				return res, fmt.Errorf("fabrik: import note for %s: %w", issueID, err)
			}
			res.Notes++
		}
	}
	return res, nil
}

// importIssue upserts one record by id with last-write-wins on
// max(updated_at). It reports true when the row was inserted or replaced.
func (s *Store) importIssue(ctx context.Context, rec Record) (bool, error) {
	e := rec.ExportIssue
	assignee := sql.NullString{}
	if e.Assignee != nil && strings.TrimSpace(*e.Assignee) != "" {
		assignee = sql.NullString{String: strings.TrimSpace(*e.Assignee), Valid: true}
	}
	started := sql.NullString{}
	if e.StartedAt != nil && strings.TrimSpace(*e.StartedAt) != "" {
		started = sql.NullString{String: strings.TrimSpace(*e.StartedAt), Valid: true}
	}
	closed := sql.NullString{}
	if e.ClosedAt != nil && strings.TrimSpace(*e.ClosedAt) != "" {
		closed = sql.NullString{String: strings.TrimSpace(*e.ClosedAt), Valid: true}
	}
	var have string
	var haveUpdated string
	err := s.DB.QueryRowContext(ctx, `SELECT id, updated_at FROM issues WHERE id = ?`, e.ID).Scan(&have, &haveUpdated)
	if err == sql.ErrNoRows {
		_, err = s.DB.ExecContext(ctx, `
			INSERT INTO issues (id, title, body, acceptance, status, priority, kind, assignee,
				created_at, updated_at, started_at, closed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			e.ID, e.Title, e.Body, e.Acceptance, e.Status, e.Priority, e.Kind,
			assignee, e.CreatedAt, e.UpdatedAt, started, closed,
		)
		if err != nil {
			return false, fmt.Errorf("fabrik: import issue %s: %w", e.ID, err)
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if !newerThan(e.UpdatedAt, haveUpdated) {
		return false, nil
	}
	_, err = s.DB.ExecContext(ctx, `
		UPDATE issues SET title = ?, body = ?, acceptance = ?, status = ?, priority = ?,
			kind = ?, assignee = ?, created_at = ?, updated_at = ?,
			started_at = ?, closed_at = ? WHERE id = ?`,
		e.Title, e.Body, e.Acceptance, e.Status, e.Priority, e.Kind,
		assignee, e.CreatedAt, e.UpdatedAt, started, closed, e.ID,
	)
	if err != nil {
		return false, fmt.Errorf("fabrik: import issue %s: %w", e.ID, err)
	}
	return true, nil
}

// newerThan reports whether a is strictly newer than b. RFC3339 timestamps
// compare directly; unparsable values fall back to lexicographic order.
func newerThan(a, b string) bool {
	ta, ea := time.Parse(time.RFC3339, a)
	tb, eb := time.Parse(time.RFC3339, b)
	if ea == nil && eb == nil {
		return ta.After(tb)
	}
	return a > b
}

// applyImportedBlock mirrors Dep for kind=blocks: when the edge is new and
// the blocker is not done and the blocked issue is open, the blocked issue
// becomes blocked. Missing endpoints are skipped (the edge row is still
// kept for the union).
func applyImportedBlock(ctx context.Context, db *sql.DB, blocked, blocker, now string) error {
	var blockerStatus, blockedStatus string
	if err := db.QueryRowContext(ctx, `SELECT status FROM issues WHERE id = ?`, blocker).Scan(&blockerStatus); err != nil {
		return nil
	}
	if err := db.QueryRowContext(ctx, `SELECT status FROM issues WHERE id = ?`, blocked).Scan(&blockedStatus); err != nil {
		return nil
	}
	if blockerStatus != "done" && blockedStatus == "open" {
		_, err := db.ExecContext(ctx, `UPDATE issues SET status = 'blocked', updated_at = ? WHERE id = ?`, now, blocked)
		return err
	}
	return nil
}
