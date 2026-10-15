package doltstore

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var branchNameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

type StatusChange struct {
	Table    string `json:"table"`
	Status   string `json:"status,omitempty"`
	Staged   bool   `json:"staged,omitempty"`
	Unstaged bool   `json:"unstaged,omitempty"`
}

type VCStatus struct {
	Branch  string         `json:"branch"`
	Head    string         `json:"head"`
	Dirty   bool           `json:"dirty"`
	Changes []StatusChange `json:"changes"`
}

type VCCommit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

type LogEntry struct {
	Hash    string `json:"hash"`
	Date    string `json:"date"`
	Message string `json:"message"`
}

type VCLog struct {
	Commits []LogEntry `json:"commits"`
}

type DiffTable struct {
	Name         string `json:"name"`
	RowsAdded    int    `json:"rows_added"`
	RowsRemoved  int    `json:"rows_removed"`
	RowsModified int    `json:"rows_modified"`
}

type VCDiff struct {
	Tables []DiffTable `json:"tables"`
}

type VCBranches struct {
	Current  string   `json:"current"`
	Branches []string `json:"branches"`
}

type VCMergeResult struct {
	Merged    bool     `json:"merged"`
	Hash      string   `json:"hash,omitempty"`
	Conflicts []string `json:"conflicts,omitempty"`
}

func (s *Store) VCStatus(ctx context.Context) (VCStatus, error) {
	var branch, head string
	if err := s.DB.QueryRowContext(ctx, `SELECT active_branch()`).Scan(&branch); err != nil {
		return VCStatus{}, err
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT DOLT_HASHOF('HEAD')`).Scan(&head); err != nil {
		return VCStatus{}, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT table_name, staged, status FROM dolt_status`)
	if err != nil {
		return VCStatus{}, err
	}
	defer rows.Close()
	var changes []StatusChange
	for rows.Next() {
		var table, status string
		var staged int
		if err := rows.Scan(&table, &staged, &status); err != nil {
			return VCStatus{}, err
		}
		if table == "schema_migrations" || table == "meta" {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(status)) {
		case "", "unchanged", "clean", "present":
			continue
		}
		ch := StatusChange{Table: table, Status: status}
		if staged == 1 {
			ch.Staged = true
		} else {
			ch.Unstaged = true
		}
		changes = append(changes, ch)
	}
	if err := rows.Err(); err != nil {
		return VCStatus{}, err
	}
	return VCStatus{
		Branch: branch, Head: head, Dirty: len(changes) > 0, Changes: changes,
	}, nil
}

func (s *Store) VCLog(ctx context.Context, limit int) (VCLog, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT commit_hash, date, message FROM dolt_log
		ORDER BY date DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return VCLog{}, err
	}
	defer rows.Close()
	var commits []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.Hash, &e.Date, &e.Message); err != nil {
			return VCLog{}, err
		}
		commits = append(commits, e)
	}
	if commits == nil {
		commits = []LogEntry{}
	}
	return VCLog{Commits: commits}, rows.Err()
}

func (s *Store) VCDiffWorking(ctx context.Context) (VCDiff, error) {
	return s.vcDiffStat(ctx, "HEAD", "WORKING")
}

func (s *Store) VCDiffRange(ctx context.Context, from, to string) (VCDiff, error) {
	return s.vcDiffStat(ctx, from, to)
}

func (s *Store) vcDiffStat(ctx context.Context, from, to string) (VCDiff, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT table_name, rows_added, rows_deleted, rows_modified
		FROM dolt_diff_stat(?, ?)
	`, from, to)
	if err != nil {
		if isUnknownCommitErr(err) {
			return VCDiff{}, &ErrUnknownCommit{Ref: from}
		}
		return VCDiff{}, err
	}
	defer rows.Close()
	var tables []DiffTable
	for rows.Next() {
		var t DiffTable
		if err := rows.Scan(&t.Name, &t.RowsAdded, &t.RowsRemoved, &t.RowsModified); err != nil {
			return VCDiff{}, err
		}
		tables = append(tables, t)
	}
	if tables == nil {
		tables = []DiffTable{}
	}
	return VCDiff{Tables: tables}, rows.Err()
}

func (s *Store) VCCommitMessage(ctx context.Context, message string) (VCCommit, error) {
	_, err := s.DB.ExecContext(ctx, `CALL DOLT_COMMIT('-m', ?)`, message)
	if err != nil {
		if isNothingToCommitErr(err) {
			return VCCommit{}, errNothingToCommit
		}
		return VCCommit{}, err
	}
	var hash string
	if err := s.DB.QueryRowContext(ctx, `SELECT DOLT_HASHOF('HEAD')`).Scan(&hash); err != nil {
		return VCCommit{}, err
	}
	return VCCommit{Hash: hash, Message: message}, nil
}

var errNothingToCommit = fmt.Errorf("fabrik: nothing to commit")

func IsNothingToCommit(err error) bool {
	return errors.Is(err, errNothingToCommit) || isNothingToCommitErr(err)
}

func isNothingToCommitErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "nothing to commit")
}

func isUnknownCommitErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "unknown commit")
}

func (s *Store) VCBranchList(ctx context.Context) (VCBranches, error) {
	var current string
	if err := s.DB.QueryRowContext(ctx, `SELECT active_branch()`).Scan(&current); err != nil {
		return VCBranches{}, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT name FROM dolt_branches ORDER BY name ASC`)
	if err != nil {
		return VCBranches{}, err
	}
	defer rows.Close()
	var branches []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return VCBranches{}, err
		}
		branches = append(branches, name)
	}
	return VCBranches{Current: current, Branches: branches}, rows.Err()
}

func ValidateBranchName(name string) error {
	if !branchNameRe.MatchString(name) {
		return fmt.Errorf("fabrik: invalid branch name %q", name)
	}
	return nil
}

func (s *Store) VCBranchCreate(ctx context.Context, name string) error {
	if err := ValidateBranchName(name); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, `CALL DOLT_BRANCH('-c', ?)`, name)
	return err
}

func (s *Store) VCCheckout(ctx context.Context, name string) error {
	if err := ValidateBranchName(name); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, `CALL DOLT_CHECKOUT(?)`, name)
	if err != nil && isBranchNotFoundErr(err) {
		return &ErrBranchNotFound{Name: name}
	}
	return err
}

func isBranchNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

func (s *Store) VCMerge(ctx context.Context, branch, strategy string) (VCMergeResult, error) {
	var current string
	if err := s.DB.QueryRowContext(ctx, `SELECT active_branch()`).Scan(&current); err != nil {
		return VCMergeResult{}, err
	}
	if current == branch {
		return VCMergeResult{}, fmt.Errorf("fabrik: cannot merge branch into itself")
	}
	if strategy == "" {
		strategy = "ours"
	}
	if strategy != "ours" && strategy != "theirs" {
		return VCMergeResult{}, fmt.Errorf("fabrik: strategy must be ours or theirs")
	}
	_, err := s.DB.ExecContext(ctx, `CALL DOLT_MERGE(?, '-m', ?, '--strategy', ?)`, branch, "merge "+branch, strategy)
	if err != nil {
		if strings.Contains(err.Error(), "conflict") {
			conflicts, _ := s.listConflicts(ctx)
			return VCMergeResult{Merged: false, Conflicts: conflicts}, nil
		}
		if isBranchNotFoundErr(err) {
			return VCMergeResult{}, &ErrBranchNotFound{Name: branch}
		}
		return VCMergeResult{}, err
	}
	var hash string
	_ = s.DB.QueryRowContext(ctx, `SELECT DOLT_HASHOF('HEAD')`).Scan(&hash)
	return VCMergeResult{Merged: true, Hash: hash}, nil
}

func (s *Store) listConflicts(ctx context.Context) ([]string, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT our_table, our_pk_cols FROM dolt_conflicts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var table, pk string
		if err := rows.Scan(&table, &pk); err != nil {
			return nil, err
		}
		out = append(out, table+"."+pk)
	}
	return out, rows.Err()
}

func (s *Store) CountIssues(ctx context.Context) (unfinished, done int, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM issues WHERE status <> 'done'`).Scan(&unfinished)
	if err != nil {
		return 0, 0, err
	}
	err = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM issues WHERE status = 'done'`).Scan(&done)
	return unfinished, done, err
}
