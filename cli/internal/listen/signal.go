package listen

import (
	"os"

	"github.com/ur-wesley/fabrik/cli/internal/store"
)

func SignalMemory() {
	cfgPath, err := store.ListenJSONPath()
	if err != nil {
		return
	}
	repo := ""
	wd, err := os.Getwd()
	if err == nil {
		if root, ok := store.RepoRoot(wd); ok {
			repo = root
		}
	}
	NotifyBestEffort(cfgPath, NewEvent("memory", repo))
}

func SignalIssues(repo, issueID, op string) {
	cfgPath, err := store.ListenJSONPath()
	if err != nil {
		return
	}
	e := NewEvent("issues", repo)
	e.ID = issueID
	e.Op = op
	NotifyBestEffort(cfgPath, e)
}

func SignalGraph(repo string) {
	cfgPath, err := store.ListenJSONPath()
	if err != nil {
		return
	}
	NotifyBestEffort(cfgPath, NewEvent("graph", repo))
}
