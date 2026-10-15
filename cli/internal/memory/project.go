package memory

import (
	"os"

	"github.com/ur-wesley/fabrik/cli/internal/store"
)

type CurrentProjectResult struct {
	Project string `json:"project"`
	Source  string `json:"source"`
}

func CurrentProject(cwd string) CurrentProjectResult {
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if root, ok := store.RepoRoot(cwd); ok {
		return CurrentProjectResult{Project: root, Source: "cwd"}
	}
	if v := os.Getenv("FABRIK_PROJECT"); v != "" {
		return CurrentProjectResult{Project: v, Source: "env"}
	}
	return CurrentProjectResult{Source: "none"}
}
