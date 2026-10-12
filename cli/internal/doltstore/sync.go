package doltstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type RemoteInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type PushResult struct {
	Remote  string `json:"remote"`
	Pushed  bool   `json:"pushed"`
	Commits int    `json:"commits"`
}

type PullResult struct {
	Remote    string   `json:"remote"`
	Pulled    bool     `json:"pulled"`
	Commits   int      `json:"commits"`
	Conflicts []string `json:"conflicts"`
}

var ErrNoDoltRemote = fmt.Errorf("fabrik: no dolt remote (fabrik dolt remote add origin <url>)")

func (s *Store) ListRemotes(ctx context.Context) ([]RemoteInfo, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT name, url FROM dolt_remotes ORDER BY name ASC`)
	if err != nil {
		if isUnsupported(err) {
			return nil, fmt.Errorf("fabrik: dolt remote: %w", err)
		}
		return nil, err
	}
	defer rows.Close()
	var out []RemoteInfo
	for rows.Next() {
		var r RemoteInfo
		if err := rows.Scan(&r.Name, &r.URL); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if out == nil {
		out = []RemoteInfo{}
	}
	return out, rows.Err()
}

func (s *Store) RemoteAdd(ctx context.Context, name, url string) error {
	_, err := s.DB.ExecContext(ctx, `CALL dolt_remote('add', ?, ?)`, name, url)
	if isUnsupported(err) {
		return fmt.Errorf("fabrik: dolt remote add: driver does not support remotes (%v)", err)
	}
	return err
}

func (s *Store) defaultRemote(ctx context.Context) (string, error) {
	remotes, err := s.ListRemotes(ctx)
	if err != nil {
		return "", err
	}
	if len(remotes) == 0 {
		return "", ErrNoDoltRemote
	}
	for _, r := range remotes {
		if r.Name == "origin" {
			return "origin", nil
		}
	}
	return remotes[0].Name, nil
}

func (s *Store) DoltPush(ctx context.Context) (PushResult, error) {
	st, err := s.VCStatus(ctx)
	if err != nil {
		return PushResult{}, err
	}
	if st.Dirty {
		return PushResult{}, fmt.Errorf("fabrik: dolt push: uncommitted changes")
	}
	remote, err := s.defaultRemote(ctx)
	if err != nil {
		return PushResult{}, err
	}
	branch := st.Branch
	var rows *sql.Rows
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}
		r, err := s.DB.QueryContext(ctx, `CALL DOLT_PUSH(?, ?)`, remote, branch)
		if isUnsupported(err) {
			return PushResult{}, fmt.Errorf("fabrik: dolt push: driver does not support DOLT_PUSH (%v)", err)
		}
		if err != nil && pushContention(err) && attempt < 3 {
			lastErr = err
			continue
		}
		if err != nil {
			return PushResult{}, fmt.Errorf("fabrik: dolt push: %w", err)
		}
		rows = r
		break
	}
	if rows == nil {
		if lastErr != nil {
			return PushResult{}, fmt.Errorf("fabrik: dolt push: %w", lastErr)
		}
		return PushResult{}, fmt.Errorf("fabrik: dolt push: failed")
	}
	defer rows.Close()
	commits := 0
	pushed := false
	for rows.Next() {
		var ref, status string
		if err := rows.Scan(&ref, &status); err == nil {
			if strings.Contains(strings.ToLower(status), "success") || status == "0" {
				pushed = true
				commits++
			}
		}
	}
	if !pushed && commits == 0 {
		return PushResult{Remote: remote, Pushed: false, Commits: 0}, nil
	}
	return PushResult{Remote: remote, Pushed: pushed || commits > 0, Commits: commits}, nil
}

func (s *Store) DoltPull(ctx context.Context) (PullResult, error) {
	remote, err := s.defaultRemote(ctx)
	if err != nil {
		return PullResult{}, err
	}
	rows, err := s.DB.QueryContext(ctx, `CALL DOLT_PULL(?)`, remote)
	if isUnsupported(err) {
		return PullResult{}, fmt.Errorf("fabrik: dolt pull: driver does not support DOLT_PULL (%v)", err)
	}
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "conflict") {
			_, _ = s.DB.ExecContext(ctx, `CALL DOLT_RESET('--hard')`)
			conflicts, _ := s.listConflicts(ctx)
			return PullResult{Remote: remote, Pulled: false, Commits: 0, Conflicts: conflicts}, fmt.Errorf("fabrik: dolt pull: %w", err)
		}
		return PullResult{}, fmt.Errorf("fabrik: dolt pull: %w", err)
	}
	defer rows.Close()
	commits := 0
	pulled := false
	for rows.Next() {
		pulled = true
		commits++
	}
	if !pulled {
		return PullResult{Remote: remote, Pulled: false, Commits: 0, Conflicts: []string{}}, nil
	}
	return PullResult{Remote: remote, Pulled: true, Commits: commits, Conflicts: []string{}}, nil
}

func pushContention(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "lease") || strings.Contains(msg, "contention") ||
		strings.Contains(msg, "busy") || strings.Contains(msg, "timeout")
}

func isUnsupported(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not supported") || strings.Contains(msg, "unsupported") || strings.Contains(msg, "nocgo")
}
