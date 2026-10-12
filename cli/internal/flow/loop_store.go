package flow

import "context"

type LoopIssue struct {
	ID    string
	Title string
}

type LoopStore interface {
	Snapshot(ctx context.Context) (unfinished int, done int, ready []LoopIssue, err error)
	ClaimNext(ctx context.Context, by string) (LoopIssue, error)
	Close() error
}

var OpenLoopStore func(ctx context.Context, repo string) (LoopStore, error)
