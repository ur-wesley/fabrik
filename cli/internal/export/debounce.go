package export

import (
	"context"
	"sync"
	"time"

	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/listen"
)

const exportDebounce = 200 * time.Millisecond

type pendingExport struct {
	repo    string
	issueID string
	op      string
}

var (
	exportMu      sync.Mutex
	exportPending *pendingExport
	exportTimer   *time.Timer
)

func PublishAfterSave(ctx context.Context, st *doltstore.Store, issueID string, op string) error {
	exportMu.Lock()
	defer exportMu.Unlock()
	exportPending = &pendingExport{repo: st.Repo, issueID: issueID, op: op}
	if exportTimer != nil {
		exportTimer.Stop()
	}
	exportTimer = time.AfterFunc(exportDebounce, func() {
		exportMu.Lock()
		p := exportPending
		exportPending = nil
		exportTimer = nil
		exportMu.Unlock()
		if p == nil || p.repo == "" {
			return
		}
		bg := context.Background()
		st2, err := doltstore.Open(bg, p.repo)
		if err != nil {
			return
		}
		_, _ = Issues(bg, st2)
		_ = st2.Close()
		listen.SignalIssues(p.repo, p.issueID, p.op)
	})
	return nil
}

func PublishIssues(ctx context.Context, st *doltstore.Store) error {
	if _, err := Issues(ctx, st); err != nil {
		return err
	}
	listen.SignalIssues(st.Repo, "", "export")
	return nil
}
