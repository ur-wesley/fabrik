package doltstore

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

// SyncResult is the `fabrik sync` outcome: a pull (git + ImportFile merge)
// followed by a push (export + git commit + push).
type SyncResult struct {
	Remote    string   `json:"remote"`
	Pulled    bool     `json:"pulled"`
	Pushed    bool     `json:"pushed"`
	Commits   int      `json:"commits"`
	Conflicts []string `json:"conflicts"`
}

var ErrNoDoltRemote = fmt.Errorf("fabrik: no dolt remote (fabrik dolt remote add origin <url>)")

func (s *Store) ListRemotes(ctx context.Context) ([]RemoteInfo, error) {
	out, err := s.git(ctx, "remote", "-v")
	if err != nil {
		return nil, err
	}
	seen := map[string]string{}
	var order []string
	if strings.TrimSpace(out) != "" {
		for _, line := range strings.Split(out, "\n") {
			fields := strings.Fields(line)
			if len(fields) < 3 || fields[2] != "(fetch)" {
				continue
			}
			if _, ok := seen[fields[0]]; !ok {
				order = append(order, fields[0])
			}
			seen[fields[0]] = fields[1]
		}
	}
	var remotes []RemoteInfo
	for _, name := range order {
		remotes = append(remotes, RemoteInfo{Name: name, URL: seen[name]})
	}
	if remotes == nil {
		remotes = []RemoteInfo{}
	}
	return remotes, nil
}

func (s *Store) RemoteAdd(ctx context.Context, name, url string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(url) == "" {
		return fmt.Errorf("fabrik: usage: fabrik dolt remote add <name> <url>")
	}
	_, err := s.git(ctx, "remote", "add", name, url)
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

// commitSnapshot exports the issue store and commits the .fabrik snapshot
// when it changed. It reports whether a new commit was created. Only the
// export snapshot plus tracked updates are staged (-u): the live DBs stay
// untracked even where no .gitignore covers them.
func (s *Store) commitSnapshot(ctx context.Context, message string) (bool, error) {
	if _, err := s.exportSnapshot(ctx); err != nil {
		return false, err
	}
	if _, err := s.git(ctx, "add", "--", ".fabrik/export/issues.jsonl"); err != nil {
		return false, err
	}
	if _, err := s.git(ctx, "add", "-u", "--", ".fabrik"); err != nil {
		return false, err
	}
	_, err := s.git(ctx, "-c", "user.name=fabrik", "-c", "user.email=fabrik@local",
		"commit", "-m", message)
	if err != nil {
		if isNothingToCommitErr(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isUpToDate(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "up to date") ||
		strings.Contains(lower, "up-to-date") ||
		strings.Contains(lower, "everything up-to-date")
}

func (s *Store) DoltPush(ctx context.Context) (PushResult, error) {
	remote, err := s.defaultRemote(ctx)
	if err != nil {
		return PushResult{}, err
	}
	branch, err := s.gitBranch(ctx)
	if err != nil {
		return PushResult{}, err
	}
	committed, err := s.commitSnapshot(ctx, "sync: export issues")
	if err != nil {
		return PushResult{}, fmt.Errorf("fabrik: dolt push: %w", err)
	}
	ahead := s.aheadCount(ctx, remote, branch)
	if _, err := s.git(ctx, "push", remote, branch); err != nil {
		if isUpToDate(err.Error()) {
			return PushResult{Remote: remote, Pushed: false, Commits: 0}, nil
		}
		return PushResult{}, fmt.Errorf("fabrik: dolt push: %w", err)
	}
	// Push succeeded. Nothing was pushed when we committed nothing and
	// HEAD was already at the remote (ahead == 0). Unknown counts
	// (no remote branch yet) count as pushed.
	if !committed && ahead == 0 {
		return PushResult{Remote: remote, Pushed: false, Commits: 0}, nil
	}
	commits := ahead
	if commits < 0 {
		commits = 0
	}
	if committed {
		commits++
	}
	if commits == 0 {
		commits = 1
	}
	return PushResult{Remote: remote, Pushed: true, Commits: commits}, nil
}

// aheadCount reports how many local commits the remote branch lacks.
// -1 means unknown (e.g. the remote branch does not exist yet).
func (s *Store) aheadCount(ctx context.Context, remote, branch string) int {
	out, err := s.git(ctx, "rev-list", "--count", remote+"/"+branch+"..HEAD")
	if err != nil {
		return -1
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return -1
	}
	return n
}

func (s *Store) DoltPull(ctx context.Context) (PullResult, error) {
	remote, err := s.defaultRemote(ctx)
	if err != nil {
		return PullResult{}, err
	}
	branch, err := s.gitBranch(ctx)
	if err != nil {
		return PullResult{}, err
	}
	before := s.gitHead(ctx)
	out, err := s.git(ctx, "pull", "--no-rebase", remote, branch)
	if err != nil {
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "conflict") ||
			strings.Contains(strings.ToLower(msg), "automatic merge failed") {
			conflicts, _ := s.listConflicts(ctx)
			return PullResult{Remote: remote, Pulled: false, Commits: 0, Conflicts: conflicts},
				fmt.Errorf("fabrik: dolt pull: %w", err)
		}
		// A fresh repo with no upstream history reports "couldn't find
		// remote ref": nothing to pull, then merge the local snapshot
		// state (no-op when the file is absent).
		if strings.Contains(strings.ToLower(msg), "couldn't find remote ref") {
			return PullResult{Remote: remote, Pulled: false, Commits: 0, Conflicts: []string{}}, nil
		}
		return PullResult{}, fmt.Errorf("fabrik: dolt pull: %w", err)
	}
	pulled := !isUpToDate(out) || s.gitHead(ctx) != before
	commits := 0
	if pulled {
		commits = 1
	}
	if _, err := s.ImportFile(ctx, s.ExportPath()); err != nil {
		return PullResult{}, fmt.Errorf("fabrik: dolt pull: %w", err)
	}
	if conflicts, _ := s.listConflicts(ctx); len(conflicts) > 0 {
		return PullResult{Remote: remote, Pulled: pulled, Commits: commits, Conflicts: conflicts},
			fmt.Errorf("fabrik: dolt pull: merge has conflicts")
	}
	return PullResult{Remote: remote, Pulled: pulled, Commits: commits, Conflicts: []string{}}, nil
}

// Sync pulls (git + snapshot merge) then pushes (export + commit + push).
// It is the `fabrik sync` trigger sharing the dolt push/pull seams.
func (s *Store) Sync(ctx context.Context) (SyncResult, error) {
	pull, err := s.DoltPull(ctx)
	if err != nil {
		return SyncResult{}, err
	}
	push, err := s.DoltPush(ctx)
	if err != nil {
		return SyncResult{}, err
	}
	return SyncResult{
		Remote: pull.Remote, Pulled: pull.Pulled, Pushed: push.Pushed,
		Commits: pull.Commits + push.Commits, Conflicts: pull.Conflicts,
	}, nil
}
