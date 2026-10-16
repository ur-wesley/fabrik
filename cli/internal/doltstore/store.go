package doltstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ur-wesley/fabrik/cli/internal/sqlmigrate"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

type Store struct {
	Repo string
	DB   *sql.DB
}

func Open(ctx context.Context, repoRoot string) (*Store, error) {
	db, err := openDB(ctx, repoRoot)
	if err != nil {
		return nil, StoreErr(err)
	}
	if err := checkMetaSchema(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{Repo: repoRoot, DB: db}, nil
}

func (s *Store) Close() error {
	if s.DB == nil {
		return nil
	}
	// sql.DB.Close is idempotent, so deferred double-close after an
	// explicit Close (e.g. to release the Dolt lock before reopening)
	// is safe. s.DB stays non-nil so use-after-close keeps returning
	// "sql: database is closed" instead of panicking.
	return s.DB.Close()
}

func (s *Store) Commit(ctx context.Context, message string) error {
	if err := doltCommit(ctx, s.DB, message); err != nil {
		return StoreErr(err)
	}
	return nil
}

func checkMetaSchema(ctx context.Context, db *sql.DB) error {
	var val string
	err := db.QueryRowContext(ctx, `SELECT value FROM meta WHERE name = 'schema_version'`).Scan(&val)
	if err != nil {
		if isNoTable(err) {
			return nil
		}
		return StoreErr(err)
	}
	max, err := sqlmigrate.MaxVersion("issues")
	if err != nil {
		return StoreErr(err)
	}
	have, _ := parseInt(val)
	if have > max {
		return &sqlmigrate.ErrSchemaNewer{Store: "issues", Have: have, Binary: max}
	}
	return nil
}

func (s *Store) nextSeq(ctx context.Context) (int64, error) {
	var n sql.NullInt64
	if err := s.DB.QueryRowContext(ctx, `SELECT MAX(seq) FROM issue_notes`).Scan(&n); err != nil {
		return 0, err
	}
	if !n.Valid {
		return 1, nil
	}
	return n.Int64 + 1, nil
}

func parseInt(s string) (int, bool) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	return n, true
}

func isNoTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "doesn't exist") || strings.Contains(msg, "not found")
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func nullableStr(ns sql.NullString) *string {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	return &ns.String
}

func scanIssue(row interface{ Scan(...any) error }, ready bool) (Issue, error) {
	var i Issue
	var assignee, started, closed sql.NullString
	err := row.Scan(
		&i.ID, &i.Title, &i.Body, &i.Acceptance, &i.Status, &i.Priority, &i.Kind,
		&assignee, &i.CreatedAt, &i.UpdatedAt, &started, &closed,
	)
	if err != nil {
		return Issue{}, err
	}
	i.Assignee = nullableStr(assignee)
	i.StartedAt = nullableStr(started)
	i.ClosedAt = nullableStr(closed)
	i.Ready = ready
	return i, nil
}

func scanIssueWithReady(row interface{ Scan(...any) error }) (Issue, error) {
	var i Issue
	var assignee, started, closed sql.NullString
	var ready int
	err := row.Scan(
		&i.ID, &i.Title, &i.Body, &i.Acceptance, &i.Status, &i.Priority, &i.Kind,
		&assignee, &i.CreatedAt, &i.UpdatedAt, &started, &closed, &ready,
	)
	if err != nil {
		return Issue{}, err
	}
	i.Assignee = nullableStr(assignee)
	i.StartedAt = nullableStr(started)
	i.ClosedAt = nullableStr(closed)
	i.Ready = ready != 0
	return i, nil
}

const issueReadySQL = `CASE WHEN i.status = 'open' AND NOT EXISTS (
	SELECT 1 FROM issue_deps d
	INNER JOIN issues b ON b.id = d.blocker_id
	WHERE d.blocked_id = i.id AND d.kind = 'blocks' AND b.status <> 'done'
) THEN 1 ELSE 0 END`

const issueReadyWhere = `i.status = 'open' AND NOT EXISTS (
	SELECT 1 FROM issue_deps d
	INNER JOIN issues b ON b.id = d.blocker_id
	WHERE d.blocked_id = i.id AND d.kind = 'blocks' AND b.status <> 'done'
)`

const defaultStaleActive = 6 * time.Hour

func (s *Store) computeReady(ctx context.Context, id string, status string) (bool, error) {
	if status != "open" {
		return false, nil
	}
	var n int
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM issue_deps d
		INNER JOIN issues b ON b.id = d.blocker_id
		WHERE d.blocked_id = ? AND d.kind = 'blocks' AND b.status <> 'done'
	`, id).Scan(&n)
	if err != nil {
		return false, err
	}
	return n == 0, nil
}

func (s *Store) enrichReady(ctx context.Context, i Issue) (Issue, error) {
	r, err := s.computeReady(ctx, i.ID, i.Status)
	if err != nil {
		return Issue{}, err
	}
	i.Ready = r
	return i, nil
}

func (s *Store) List(ctx context.Context, statusFilter string) ([]Issue, error) {
	q := `SELECT i.id, i.title, i.body, i.acceptance, i.status, i.priority, i.kind, i.assignee,
			i.created_at, i.updated_at, i.started_at, i.closed_at, ` + issueReadySQL + ` AS ready_flag
		FROM issues i`
	var args []any
	if statusFilter != "" {
		q += ` WHERE i.status = ?`
		args = append(args, statusFilter)
	}
	q += ` ORDER BY i.priority ASC, i.id ASC`

	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Issue
	for rows.Next() {
		i, err := scanIssueWithReady(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if out == nil {
		out = []Issue{}
	}
	return out, rows.Err()
}

type CreateInput struct {
	Title      string
	Body       string
	Acceptance string
	Priority   *int
	Kind       string
}

func (s *Store) Create(ctx context.Context, in CreateInput) (Issue, error) {
	if err := normalizeCreateInput(&in); err != nil {
		return Issue{}, err
	}
	id, err := store.NewHashID("f-")
	if err != nil {
		return Issue{}, err
	}
	ts := nowRFC3339()
	priority := *in.Priority
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO issues (id, title, body, acceptance, status, priority, kind, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'open', ?, ?, ?, ?)
	`, id, in.Title, in.Body, in.Acceptance, priority, in.Kind, ts, ts)
	if err != nil {
		return Issue{}, err
	}
	if err := s.Commit(ctx, fmt.Sprintf("issue create %s", id)); err != nil {
		return Issue{}, err
	}
	return Issue{
		ID: id, Title: in.Title, Body: in.Body, Acceptance: in.Acceptance,
		Status: "open", Priority: priority, Kind: in.Kind,
		CreatedAt: ts, UpdatedAt: ts, Ready: true,
	}, nil
}

func (s *Store) Get(ctx context.Context, id string) (Issue, error) {
	row := s.DB.QueryRowContext(ctx, `
		SELECT id, title, body, acceptance, status, priority, kind, assignee,
			created_at, updated_at, started_at, closed_at
		FROM issues WHERE id = ?
	`, id)
	i, err := scanIssue(row, false)
	if err == sql.ErrNoRows {
		return Issue{}, ErrNotFound
	}
	if err != nil {
		return Issue{}, err
	}
	return s.enrichReady(ctx, i)
}

func (s *Store) Show(ctx context.Context, id string) (Issue, error) {
	i, err := s.Get(ctx, id)
	if err != nil {
		return Issue{}, err
	}
	blockers, err := s.loadBlockers(ctx, id)
	if err != nil {
		return Issue{}, err
	}
	blocking, err := s.loadBlocking(ctx, id)
	if err != nil {
		return Issue{}, err
	}
	notes, err := s.loadNotes(ctx, id)
	if err != nil {
		return Issue{}, err
	}
	i.Blockers = blockers
	i.Blocking = blocking
	i.Notes = notes
	return i, nil
}

func (s *Store) loadRefs(ctx context.Context, q string, id string) ([]IssueRef, error) {
	rows, err := s.DB.QueryContext(ctx, q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IssueRef
	for rows.Next() {
		var r IssueRef
		if err := rows.Scan(&r.ID, &r.Title, &r.Status, &r.Priority, &r.Kind); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) loadBlockers(ctx context.Context, id string) ([]IssueRef, error) {
	return s.loadRefs(ctx, `
		SELECT i.id, i.title, i.status, i.priority, i.kind
		FROM issue_deps d
		INNER JOIN issues i ON i.id = d.blocker_id
		WHERE d.blocked_id = ?
		ORDER BY i.id ASC
	`, id)
}

func (s *Store) loadBlocking(ctx context.Context, id string) ([]IssueRef, error) {
	return s.loadRefs(ctx, `
		SELECT i.id, i.title, i.status, i.priority, i.kind
		FROM issue_deps d
		INNER JOIN issues i ON i.id = d.blocked_id
		WHERE d.blocker_id = ?
		ORDER BY i.id ASC
	`, id)
}

func (s *Store) loadNotes(ctx context.Context, id string) ([]Note, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, body, created_at FROM issue_notes
		WHERE issue_id = ?
		ORDER BY created_at ASC, COALESCE(seq, 9223372036854775807) ASC, rowid ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Note
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Body, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) Ready(ctx context.Context) ([]Issue, error) {
	q := `SELECT i.id, i.title, i.body, i.acceptance, i.status, i.priority, i.kind, i.assignee,
			i.created_at, i.updated_at, i.started_at, i.closed_at, 1 AS ready_flag
		FROM issues i
		WHERE i.status = 'open' AND NOT EXISTS (
			SELECT 1 FROM issue_deps d
			INNER JOIN issues b ON b.id = d.blocker_id
			WHERE d.blocked_id = i.id AND d.kind = 'blocks' AND b.status <> 'done'
		)
		ORDER BY i.priority ASC, i.id ASC`
	rows, err := s.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Issue
	for rows.Next() {
		i, err := scanIssueWithReady(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	if out == nil {
		out = []Issue{}
	}
	return out, rows.Err()
}

func (s *Store) firstIncompleteBlocker(ctx context.Context, blockedID string) (string, error) {
	var id string
	err := s.DB.QueryRowContext(ctx, `
		SELECT b.id FROM issue_deps d
		INNER JOIN issues b ON b.id = d.blocker_id
		WHERE d.blocked_id = ? AND d.kind = 'blocks' AND b.status <> 'done'
		ORDER BY b.id ASC
		LIMIT 1
	`, blockedID).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) firstBlockerID(ctx context.Context, blockedID string) (string, error) {
	var id string
	err := s.DB.QueryRowContext(ctx, `
		SELECT b.id FROM issue_deps d
		INNER JOIN issues b ON b.id = d.blocker_id
		WHERE d.blocked_id = ? AND d.kind = 'blocks'
		ORDER BY b.id ASC
		LIMIT 1
	`, blockedID).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) ReclaimStaleActive(ctx context.Context, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = defaultStaleActive
	}
	cutoff := time.Now().UTC().Add(-ttl).Format(time.RFC3339)
	res, err := s.DB.ExecContext(ctx, `
		UPDATE issues SET status = 'open', assignee = NULL, updated_at = ?
		WHERE status = 'active' AND updated_at < ?
	`, nowRFC3339(), cutoff)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil
	}
	return s.Commit(ctx, "issue reclaim stale active")
}

func (s *Store) firstReadyID(ctx context.Context) (string, error) {
	var id string
	err := s.DB.QueryRowContext(ctx, `
		SELECT i.id FROM issues i
		WHERE `+issueReadyWhere+`
		ORDER BY i.priority ASC, i.id ASC
		LIMIT 1
	`).Scan(&id)
	if err == sql.ErrNoRows {
		return "", ErrNoReady
	}
	return id, err
}

func (s *Store) ClaimNext(ctx context.Context, by string) (Issue, error) {
	_ = s.ReclaimStaleActive(ctx, defaultStaleActive)
	id, err := s.firstReadyID(ctx)
	if err != nil {
		return Issue{}, err
	}
	return s.Claim(ctx, id, by)
}

func (s *Store) Claim(ctx context.Context, id, by string) (Issue, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return s.ClaimNext(ctx, by)
	}
	if err := validateIssueID(id); err != nil {
		return Issue{}, err
	}
	by = strings.TrimSpace(by)
	if by == "" {
		by = "fabrik"
	}
	if err := validateAssignee(by); err != nil {
		return Issue{}, err
	}
	i, err := s.Get(ctx, id)
	if err != nil {
		return Issue{}, err
	}
	switch i.Status {
	case "active":
		return Issue{}, &ErrIssueStatus{ID: id, Status: "active"}
	case "done":
		return Issue{}, &ErrIssueStatus{ID: id, Status: "done"}
	}
	if !i.Ready {
		blocker, err := s.firstIncompleteBlocker(ctx, id)
		if err != nil {
			return Issue{}, err
		}
		if blocker == "" {
			blocker, err = s.firstBlockerID(ctx, id)
			if err != nil {
				return Issue{}, err
			}
		}
		return Issue{}, &ErrIssueBlocked{ID: id, BlockerID: blocker}
	}
	ts := nowRFC3339()
	started := ts
	if i.StartedAt != nil && *i.StartedAt != "" {
		started = *i.StartedAt
	}
	res, err := s.DB.ExecContext(ctx, `
		UPDATE issues SET status = 'active', assignee = ?, started_at = ?, updated_at = ?
		WHERE id = ? AND status = 'open'
	`, by, started, ts, id)
	if err != nil {
		return Issue{}, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return Issue{}, &ErrClaimLost{ID: id}
	}
	if err := s.Commit(ctx, fmt.Sprintf("issue claim %s", id)); err != nil {
		return Issue{}, err
	}
	i.Status = "active"
	i.Assignee = &by
	i.StartedAt = &started
	i.UpdatedAt = ts
	i.Ready = false
	return i, nil
}

func (s *Store) Unclaim(ctx context.Context, id, by string, force bool) (Issue, error) {
	if err := validateIssueID(id); err != nil {
		return Issue{}, err
	}
	by = strings.TrimSpace(by)
	if by == "" {
		by = "fabrik"
	}
	i, err := s.Get(ctx, id)
	if err != nil {
		return Issue{}, err
	}
	if i.Status != "active" {
		return Issue{}, &ErrIssueStatus{ID: id, Status: i.Status}
	}
	if !force && i.Assignee != nil && *i.Assignee != by {
		return Issue{}, &ErrNotAssignee{ID: id, Assignee: *i.Assignee}
	}
	ts := nowRFC3339()
	res, err := s.DB.ExecContext(ctx, `
		UPDATE issues SET status = 'open', assignee = NULL, updated_at = ?
		WHERE id = ? AND status = 'active'
	`, ts, id)
	if err != nil {
		return Issue{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Issue{}, &ErrClaimLost{ID: id}
	}
	if err := s.Commit(ctx, fmt.Sprintf("issue unclaim %s", id)); err != nil {
		return Issue{}, err
	}
	return s.Get(ctx, id)
}

func (s *Store) allBlockersDone(ctx context.Context, blockedID string) (bool, error) {
	blocker, err := s.firstIncompleteBlocker(ctx, blockedID)
	if err != nil {
		return false, err
	}
	return blocker == "", nil
}

func (s *Store) unblockDependents(ctx context.Context, blockerID string, ts string) error {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT d.blocked_id FROM issue_deps d
		WHERE d.blocker_id = ? AND d.kind = 'blocks'
	`, blockerID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var dependents []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		dependents = append(dependents, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, depID := range dependents {
		var status string
		err := s.DB.QueryRowContext(ctx, `SELECT status FROM issues WHERE id = ?`, depID).Scan(&status)
		if err != nil {
			return err
		}
		if status != "blocked" {
			continue
		}
		ok, err := s.allBlockersDone(ctx, depID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		_, err = s.DB.ExecContext(ctx, `
			UPDATE issues SET status = 'open', started_at = NULL, updated_at = ? WHERE id = ?
		`, ts, depID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CloseIssue(ctx context.Context, id string, reason string) (CloseResult, error) {
	return s.CloseIssueBy(ctx, id, reason, "", false)
}

func (s *Store) CloseIssueBy(ctx context.Context, id, reason, by string, force bool) (CloseResult, error) {
	if err := validateIssueID(id); err != nil {
		return CloseResult{}, err
	}
	reason = strings.TrimSpace(reason)
	if err := validateCloseReason(reason); err != nil {
		return CloseResult{}, err
	}
	by = strings.TrimSpace(by)
	i, err := s.Get(ctx, id)
	if err != nil {
		return CloseResult{}, err
	}
	if i.Status == "done" {
		return CloseResult{}, &ErrIssueStatus{ID: id, Status: "done"}
	}
	if i.Status == "active" && by != "" && !force {
		if i.Assignee == nil || *i.Assignee != by {
			assignee := ""
			if i.Assignee != nil {
				assignee = *i.Assignee
			}
			return CloseResult{}, &ErrNotAssignee{ID: id, Assignee: assignee}
		}
	}
	ts := nowRFC3339()
	noteID := ""
	if reason != "" {
		noteID, err = store.NewHashID("n-")
		if err != nil {
			return CloseResult{}, err
		}
		seq, err := s.nextSeq(ctx)
		if err != nil {
			return CloseResult{}, err
		}
		_, err = s.DB.ExecContext(ctx, `
			INSERT INTO issue_notes (id, issue_id, body, created_at, seq) VALUES (?, ?, ?, ?, ?)
		`, noteID, id, reason, ts, seq)
		if err != nil {
			return CloseResult{}, err
		}
	}
	_, err = s.DB.ExecContext(ctx, `
		UPDATE issues SET status = 'done', closed_at = ?, updated_at = ? WHERE id = ?
	`, ts, ts, id)
	if err != nil {
		return CloseResult{}, err
	}
	if err := s.unblockDependents(ctx, id, ts); err != nil {
		return CloseResult{}, err
	}
	if err := s.Commit(ctx, fmt.Sprintf("issue close %s", id)); err != nil {
		return CloseResult{}, err
	}
	i.Status = "done"
	i.ClosedAt = &ts
	i.UpdatedAt = ts
	i.Ready = false
	return CloseResult{Issue: i, NoteID: noteID}, nil
}

func (s *Store) depWouldCycle(ctx context.Context, blocked, blocker string) (string, error) {
	seen := map[string]bool{}
	var dfs func(string) (string, error)
	dfs = func(id string) (string, error) {
		if seen[id] {
			return id, nil
		}
		seen[id] = true
		rows, err := s.DB.QueryContext(ctx, `
			SELECT blocker_id FROM issue_deps WHERE blocked_id = ? AND kind = 'blocks'
		`, id)
		if err != nil {
			return "", err
		}
		defer rows.Close()
		for rows.Next() {
			var bid string
			if err := rows.Scan(&bid); err != nil {
				return "", err
			}
			if bid == blocker {
				return blocker, nil
			}
			c, err := dfs(bid)
			if err != nil {
				return "", err
			}
			if c != "" {
				return c, nil
			}
		}
		if err := rows.Err(); err != nil {
			return "", err
		}
		return "", nil
	}
	return dfs(blocked)
}

func (s *Store) Dep(ctx context.Context, blocked, blocker, kind string) (DepResult, error) {
	if err := validateIssueID(blocked); err != nil {
		return DepResult{}, err
	}
	if err := validateIssueID(blocker); err != nil {
		return DepResult{}, err
	}
	if kind == "" {
		kind = "blocks"
	}
	if err := validateDepKind(kind); err != nil {
		return DepResult{}, err
	}
	if blocked == blocker {
		return DepResult{}, &ErrDepSelf{}
	}
	if _, err := s.Get(ctx, blocked); err != nil {
		return DepResult{}, err
	}
	if _, err := s.Get(ctx, blocker); err != nil {
		return DepResult{}, err
	}
	var exists int
	err := s.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM issue_deps WHERE blocker_id = ? AND blocked_id = ? AND kind = ?
	`, blocker, blocked, kind).Scan(&exists)
	if err != nil {
		return DepResult{}, err
	}
	if exists > 0 {
		i, err := s.Get(ctx, blocked)
		if err != nil {
			return DepResult{}, err
		}
		return DepResult{Issue: i, AlreadyExists: true}, nil
	}
	if kind == "blocks" {
		c, err := s.depWouldCycle(ctx, blocked, blocker)
		if err != nil {
			return DepResult{}, err
		}
		if c != "" {
			return DepResult{}, &ErrDepCycle{ID: c}
		}
	}
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO issue_deps (blocker_id, blocked_id, kind) VALUES (?, ?, ?)
	`, blocker, blocked, kind)
	if err != nil {
		return DepResult{}, err
	}
	ts := nowRFC3339()
	if kind == "blocks" {
		var blockerStatus, blockedStatus string
		err = s.DB.QueryRowContext(ctx, `SELECT status FROM issues WHERE id = ?`, blocker).Scan(&blockerStatus)
		if err != nil {
			return DepResult{}, err
		}
		err = s.DB.QueryRowContext(ctx, `SELECT status FROM issues WHERE id = ?`, blocked).Scan(&blockedStatus)
		if err != nil {
			return DepResult{}, err
		}
		if blockerStatus != "done" && blockedStatus == "open" {
			_, err = s.DB.ExecContext(ctx, `
				UPDATE issues SET status = 'blocked', updated_at = ? WHERE id = ?
			`, ts, blocked)
			if err != nil {
				return DepResult{}, err
			}
		}
	}
	if err := s.Commit(ctx, fmt.Sprintf("issue dep %s %s", blocked, blocker)); err != nil {
		return DepResult{}, err
	}
	i, err := s.Get(ctx, blocked)
	if err != nil {
		return DepResult{}, err
	}
	return DepResult{Issue: i, AlreadyExists: false}, nil
}

func (s *Store) Note(ctx context.Context, issueID, body string) (Note, error) {
	if err := validateIssueID(issueID); err != nil {
		return Note{}, err
	}
	body = strings.TrimSpace(body)
	if err := validateNoteBody(body); err != nil {
		return Note{}, err
	}
	if _, err := s.Get(ctx, issueID); err != nil {
		return Note{}, err
	}
	id, err := store.NewHashID("n-")
	if err != nil {
		return Note{}, err
	}
	ts := nowRFC3339()
	seq, err := s.nextSeq(ctx)
	if err != nil {
		return Note{}, err
	}
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO issue_notes (id, issue_id, body, created_at, seq) VALUES (?, ?, ?, ?, ?)
	`, id, issueID, body, ts, seq)
	if err != nil {
		return Note{}, err
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE issues SET updated_at = ? WHERE id = ?`, ts, issueID)
	if err != nil {
		return Note{}, err
	}
	if err := s.Commit(ctx, fmt.Sprintf("issue note %s", id)); err != nil {
		return Note{}, err
	}
	return Note{ID: id, IssueID: issueID, Body: body, CreatedAt: ts}, nil
}
