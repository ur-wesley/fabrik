package main

import (
	"context"

	"github.com/ur-wesley/fabrik/cli/internal/export"
	"github.com/ur-wesley/fabrik/cli/internal/flow"
	"github.com/ur-wesley/fabrik/cli/internal/issuestore"
	"github.com/ur-wesley/fabrik/cli/internal/store"
)

type issueLoopStore struct {
	st *issuestore.Store
}

func (d *issueLoopStore) Snapshot(ctx context.Context) (int, int, []flow.LoopIssue, error) {
	unfinished, done, err := d.st.CountIssues(ctx)
	if err != nil {
		return 0, 0, nil, err
	}
	ready, err := d.st.Ready(ctx)
	if err != nil {
		return 0, 0, nil, err
	}
	out := make([]flow.LoopIssue, len(ready))
	for i, iss := range ready {
		out[i] = flow.LoopIssue{ID: iss.ID, Title: iss.Title}
	}
	return unfinished, done, out, nil
}

func (d *issueLoopStore) ClaimNext(ctx context.Context, by string) (flow.LoopIssue, error) {
	iss, err := d.st.ClaimNext(ctx, by)
	if err != nil {
		return flow.LoopIssue{}, err
	}
	if err := export.PublishAfterSave(ctx, d.st, iss.ID, "claim"); err != nil {
		return flow.LoopIssue{}, err
	}
	return flow.LoopIssue{ID: iss.ID, Title: iss.Title}, nil
}

func (d *issueLoopStore) Close() error {
	return d.st.Close()
}

func openIssueLoopStore(ctx context.Context, repo string) (flow.LoopStore, error) {
	if _, ok := store.RepoRoot(repo); !ok {
		return nil, store.ErrNoStore
	}
	st, err := issuestore.Open(ctx, repo)
	if err != nil {
		return nil, err
	}
	return &issueLoopStore{st: st}, nil
}
