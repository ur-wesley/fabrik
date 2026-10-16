package doltstore

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ur-wesley/fabrik/cli/internal/fileutil"
)

// snapshotRecord is one .fabrik/export/issues.jsonl line: the exported
// issue row plus its dependency edges and notes. It mirrors
// issuestore.Record without importing it (issuestore imports doltstore
// for the shared types, so the reverse import would cycle).
type snapshotRecord struct {
	ExportIssue
	Deps  []Dep  `json:"deps,omitempty"`
	Notes []Note `json:"notes,omitempty"`
}

// exportSnapshot writes the .fabrik/export/issues.jsonl sync snapshot:
// one rich record per issue (row + dep edges + notes). The file is
// git-tracked: it is the cross-machine sync unit. Returns the issue count.
func (s *Store) exportSnapshot(ctx context.Context) (int, error) {
	issues, err := s.List(ctx, "")
	if err != nil {
		return 0, err
	}
	deps, err := s.IssueDeps(ctx)
	if err != nil {
		return 0, err
	}
	byBlocked := map[string][]Dep{}
	for _, d := range deps {
		byBlocked[d.BlockedID] = append(byBlocked[d.BlockedID], d)
	}
	dir := filepath.Join(s.Repo, ".fabrik", "export")
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
		notes, err := s.IssueNotes(ctx, iss.ID)
		if err != nil {
			return 0, err
		}
		rec := snapshotRecord{ExportIssue: iss.Export(), Deps: byBlocked[iss.ID], Notes: notes}
		if rec.Deps == nil {
			rec.Deps = []Dep{}
		}
		if rec.Notes == nil {
			rec.Notes = []Note{}
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

// ExportPath returns the sync snapshot path for this store.
func (s *Store) ExportPath() string {
	return filepath.Join(s.Repo, ".fabrik", "export", "issues.jsonl")
}
