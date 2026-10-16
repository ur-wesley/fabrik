package export

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
	"github.com/ur-wesley/fabrik/cli/internal/fileutil"
	"github.com/ur-wesley/fabrik/cli/internal/issuestore"
)

// Store is the issue-store surface the export snapshot needs. Both the
// SQLite issuestore and the legacy doltstore satisfy it, so `dolt pull`
// keeps exporting while the day-to-day path uses SQLite.
type Store interface {
	List(ctx context.Context, statusFilter string) ([]doltstore.Issue, error)
	IssueDeps(ctx context.Context) ([]doltstore.Dep, error)
	IssueNotes(ctx context.Context, issueID string) ([]doltstore.Note, error)
	RepoPath() string
}

// Issues writes the .fabrik/export/issues.jsonl sync snapshot: one rich
// record per issue (row + dep edges + notes). Every line still decodes as a
// plain issue object; the extra deps/notes keys drive the last-write-wins
// merge on import (see issuestore.ImportFile). The file is git-tracked: it
// is the cross-machine sync unit now that Dolt remotes are gone.
func Issues(ctx context.Context, st Store) (int, error) {
	issues, err := st.List(ctx, "")
	if err != nil {
		return 0, err
	}
	deps, err := st.IssueDeps(ctx)
	if err != nil {
		return 0, err
	}
	byBlocked := map[string][]doltstore.Dep{}
	for _, d := range deps {
		byBlocked[d.BlockedID] = append(byBlocked[d.BlockedID], d)
	}
	dir := filepath.Join(st.RepoPath(), ".fabrik", "export")
	if err := os.MkdirAll(dir, fileutil.DirPerm); err != nil {
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
		notes, err := st.IssueNotes(ctx, iss.ID)
		if err != nil {
			return 0, err
		}
		rec := issuestore.Record{ExportIssue: iss.Export(), Deps: byBlocked[iss.ID], Notes: notes}
		if rec.Deps == nil {
			rec.Deps = []doltstore.Dep{}
		}
		if rec.Notes == nil {
			rec.Notes = []doltstore.Note{}
		}
		for i := range rec.Notes {
			rec.Notes[i].IssueID = iss.ID
		}
		if err := enc.Encode(rec); err != nil {
			return 0, err
		}
	}
	return len(issues), nil
}
