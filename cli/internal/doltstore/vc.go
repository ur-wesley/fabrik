package doltstore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
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

// git runs git -C repo with args and returns trimmed stdout. Stderr is
// folded into the error so callers can match on git's diagnostics
// (unknown revision, branch not found, nothing to commit, conflicts).
// Stdout and stderr stay separate so warnings on stderr (e.g. CRLF
// notices) never pollute parsed stdout (log lines, numstat, remotes).
func (s *Store) git(ctx context.Context, args ...string) (string, error) {
	full := append([]string{"-C", s.Repo}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		diag := strings.TrimSpace(stderr.String())
		if diag == "" {
			diag = strings.TrimSpace(stdout.String())
		}
		if diag == "" {
			return "", err
		}
		return "", fmt.Errorf("%s: %w", diag, err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (s *Store) gitBranch(ctx context.Context) (string, error) {
	if out, err := s.git(ctx, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		return out, nil
	}
	if out, err := s.git(ctx, "symbolic-ref", "--short", "HEAD"); err == nil {
		return out, nil
	}
	return "", fmt.Errorf("fabrik: not a git repository: %s", s.Repo)
}

func (s *Store) gitHead(ctx context.Context) string {
	out, err := s.git(ctx, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

func (s *Store) VCStatus(ctx context.Context) (VCStatus, error) {
	branch, err := s.gitBranch(ctx)
	if err != nil {
		return VCStatus{}, err
	}
	head := s.gitHead(ctx)
	out, err := s.git(ctx, "status", "--porcelain=v1", "-uall")
	if err != nil {
		return VCStatus{}, err
	}
	var changes []StatusChange
	if strings.TrimSpace(out) != "" {
		for _, line := range strings.Split(out, "\n") {
			if len(line) < 4 {
				continue
			}
			x, y, path := line[0], line[1], strings.TrimSpace(line[3:])
			// Porcelain renames carry "orig -> new"; the working path is new.
			if i := strings.Index(path, " -> "); i >= 0 {
				path = path[i+4:]
			}
			if isLiveDBPath(path) {
				continue
			}
			ch := StatusChange{Table: path, Status: strings.TrimSpace(string([]byte{x, y}))}
			if x != ' ' && x != '?' {
				ch.Staged = true
			}
			if y != ' ' {
				ch.Unstaged = true
			}
			if !ch.Staged && !ch.Unstaged {
				ch.Unstaged = true
			}
			changes = append(changes, ch)
		}
	}
	return VCStatus{
		Branch: branch, Head: head, Dirty: len(changes) > 0, Changes: changes,
	}, nil
}

// isLiveDBPath reports the live SQLite files (.fabrik/issues.db*,
// .fabrik/graph.db*). They are never versioned (init gitignores them);
// the git-tracked sync unit is .fabrik/export/issues.jsonl, so working
// status and commits ignore the live DBs. This keeps `vc status` clean
// right after issue writes, like the old auto-committing Dolt store.
func isLiveDBPath(path string) bool {
	p := strings.ReplaceAll(path, "\\", "/")
	p = strings.TrimPrefix(p, "./")
	return strings.HasPrefix(p, ".fabrik/issues.db") ||
		strings.HasPrefix(p, ".fabrik/graph.db")
}

func (s *Store) VCLog(ctx context.Context, limit int) (VCLog, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	out, err := s.git(ctx, "log", "--format=%H%x1f%ad%x1f%s", "--date=iso-strict",
		"-n", strconv.Itoa(limit))
	if err != nil {
		// No commits yet: git log fails on an unborn HEAD. Shape parity:
		// empty commit list, no error (Dolt returned zero rows likewise).
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "does not have any commits") ||
			strings.Contains(msg, "unknown revision") ||
			strings.Contains(msg, "bad default revision") {
			return VCLog{Commits: []LogEntry{}}, nil
		}
		return VCLog{}, err
	}
	var commits []LogEntry
	if strings.TrimSpace(out) != "" {
		for _, line := range strings.Split(out, "\n") {
			parts := strings.SplitN(line, "\x1f", 3)
			if len(parts) != 3 {
				continue
			}
			commits = append(commits, LogEntry{Hash: parts[0], Date: parts[1], Message: parts[2]})
		}
	}
	if commits == nil {
		commits = []LogEntry{}
	}
	return VCLog{Commits: commits}, nil
}

func (s *Store) VCDiffWorking(ctx context.Context) (VCDiff, error) {
	if s.gitHead(ctx) == "" {
		return VCDiff{Tables: []DiffTable{}}, nil
	}
	out, err := s.git(ctx, "diff", "HEAD", "--numstat")
	if err != nil {
		return VCDiff{}, err
	}
	return numstatDiff(out)
}

func (s *Store) VCDiffRange(ctx context.Context, from, to string) (VCDiff, error) {
	out, err := s.git(ctx, "diff", "--numstat", from, to)
	if err != nil {
		if isUnknownRevisionErr(err) {
			return VCDiff{}, &ErrUnknownCommit{Ref: from}
		}
		return VCDiff{}, err
	}
	return numstatDiff(out)
}

// numstatDiff maps `git diff --numstat` rows onto the VCDiff shape.
// Dolt reported per-table row counts; git reports per-file line counts.
// Field names stay identical so `fabrik vc diff --json` consumers keep
// working; RowsModified has no git equivalent and stays 0.
func numstatDiff(out string) (VCDiff, error) {
	var tables []DiffTable
	if strings.TrimSpace(out) != "" {
		for _, line := range strings.Split(out, "\n") {
			fields := strings.Split(line, "\t")
			if len(fields) != 3 {
				continue
			}
			added, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
			removed, _ := strconv.Atoi(strings.TrimSpace(fields[1]))
			if fields[0] == "-" {
				added = 0
			}
			if fields[1] == "-" {
				removed = 0
			}
			tables = append(tables, DiffTable{
				Name: fields[2], RowsAdded: added, RowsRemoved: removed,
			})
		}
	}
	if tables == nil {
		tables = []DiffTable{}
	}
	return VCDiff{Tables: tables}, nil
}

func (s *Store) VCCommitMessage(ctx context.Context, message string) (VCCommit, error) {
	if _, err := s.exportSnapshot(ctx); err != nil {
		return VCCommit{}, err
	}
	// Stage the snapshot plus tracked updates only (-u): the live DBs
	// stay untracked even where no .gitignore covers them.
	if _, err := s.git(ctx, "add", "--", ".fabrik/export/issues.jsonl"); err != nil {
		return VCCommit{}, err
	}
	if _, err := s.git(ctx, "add", "-u", "--", ".fabrik"); err != nil {
		return VCCommit{}, err
	}
	_, err := s.git(ctx, "-c", "user.name=fabrik", "-c", "user.email=fabrik@local",
		"commit", "-m", message)
	if err != nil {
		if isNothingToCommitErr(err) {
			return VCCommit{}, errNothingToCommit
		}
		return VCCommit{}, err
	}
	return VCCommit{Hash: s.gitHead(ctx), Message: message}, nil
}

var errNothingToCommit = fmt.Errorf("fabrik: nothing to commit")

func IsNothingToCommit(err error) bool {
	return errors.Is(err, errNothingToCommit) || isNothingToCommitErr(err)
}

func isNothingToCommitErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "nothing to commit") ||
		strings.Contains(msg, "no changes added to commit") ||
		strings.Contains(msg, "nothing added to commit")
}

func isUnknownRevisionErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown revision") ||
		strings.Contains(msg, "unknown commit") ||
		strings.Contains(msg, "bad revision") ||
		strings.Contains(msg, "ambiguous argument")
}

func isUnknownCommitErr(err error) bool {
	return isUnknownRevisionErr(err)
}

func (s *Store) VCBranchList(ctx context.Context) (VCBranches, error) {
	current, err := s.gitBranch(ctx)
	if err != nil {
		return VCBranches{}, err
	}
	out, err := s.git(ctx, "branch", "--format=%(refname:short)")
	if err != nil {
		return VCBranches{}, err
	}
	var branches []string
	if strings.TrimSpace(out) != "" {
		for _, line := range strings.Split(out, "\n") {
			name := strings.TrimSpace(line)
			if name != "" {
				branches = append(branches, name)
			}
		}
	}
	sort.Strings(branches)
	if len(branches) == 0 {
		branches = []string{current}
	}
	return VCBranches{Current: current, Branches: branches}, nil
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
	_, err := s.git(ctx, "branch", name)
	return err
}

func (s *Store) VCCheckout(ctx context.Context, name string) error {
	if err := ValidateBranchName(name); err != nil {
		return err
	}
	_, err := s.git(ctx, "checkout", name)
	if err != nil && isBranchNotFoundErr(err) {
		return &ErrBranchNotFound{Name: name}
	}
	return err
}

func isBranchNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "did not match") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "unknown revision")
}

func (s *Store) VCMerge(ctx context.Context, branch, strategy string) (VCMergeResult, error) {
	current, err := s.gitBranch(ctx)
	if err != nil {
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
	_, err = s.git(ctx, "-c", "user.name=fabrik", "-c", "user.email=fabrik@local",
		"merge", "--no-edit", "-X", strategy, branch, "-m", "merge "+branch)
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "conflict") || strings.Contains(msg, "automatic merge failed") {
			conflicts, _ := s.listConflicts(ctx)
			return VCMergeResult{Merged: false, Conflicts: conflicts}, nil
		}
		if isBranchNotFoundErr(err) {
			return VCMergeResult{}, &ErrBranchNotFound{Name: branch}
		}
		return VCMergeResult{}, err
	}
	return VCMergeResult{Merged: true, Hash: s.gitHead(ctx)}, nil
}

func (s *Store) listConflicts(ctx context.Context) ([]string, error) {
	out, err := s.git(ctx, "diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return nil, err
	}
	var files []string
	if strings.TrimSpace(out) != "" {
		for _, line := range strings.Split(out, "\n") {
			if f := strings.TrimSpace(line); f != "" {
				files = append(files, f)
			}
		}
	}
	return files, nil
}

func (s *Store) CountIssues(ctx context.Context) (unfinished, done int, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM issues WHERE status <> 'done'`).Scan(&unfinished)
	if err != nil {
		return 0, 0, err
	}
	err = s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM issues WHERE status = 'done'`).Scan(&done)
	return unfinished, done, err
}
