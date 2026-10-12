package export

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
)

func Issues(ctx context.Context, st *doltstore.Store) (int, error) {
	issues, err := st.List(ctx, "")
	if err != nil {
		return 0, err
	}
	dir := filepath.Join(st.Repo, ".fabrik", "export")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	path := filepath.Join(dir, "issues.jsonl")
	f, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, iss := range issues {
		if err := enc.Encode(iss.Export()); err != nil {
			return 0, err
		}
	}
	return len(issues), nil
}
