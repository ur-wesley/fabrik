// Package issuestore is the SQLite-backed issue store (modernc.org/sqlite,
// no CGO) replacing the embedded-Dolt doltstore for day-to-day issue CRUD.
//
// Wire format and Store API intentionally mirror internal/doltstore so the
// CLI surface (issue list/ready/show/create/claim/unclaim/close/dep/note,
// history, export) and the MCP issue tools keep working unchanged: the
// issue/dep/note types below are aliases of the doltstore types, and the
// Store methods below keep the same names, Ready semantics, and validation
// rules as doltstore.Store.
//
// Cross-machine sync no longer uses Dolt remotes. The sync unit is the
// git-tracked .fabrik/export/issues.jsonl snapshot (see Record): merges are
// last-write-wins by updated_at for issues, union-by-triple for deps, and
// union-by-id for notes (see sync.go).
package issuestore

import (
	"github.com/ur-wesley/fabrik/cli/internal/doltstore"
)

// Issue, dependency, and note types mirror doltstore exactly so CLI, export,
// and MCP layers can mix issuestore stores with doltstore helpers.
type (
	Issue       = doltstore.Issue
	IssueRef    = doltstore.IssueRef
	Note        = doltstore.Note
	CloseResult = doltstore.CloseResult
	DepResult   = doltstore.DepResult
	ExportIssue = doltstore.ExportIssue
	IssueSlim   = doltstore.IssueSlim
	Dep         = doltstore.Dep
)

// Error values and types mirror doltstore so callers keep a single
// error-handling shape across both stores.
type (
	ErrClaimLost    = doltstore.ErrClaimLost
	ErrNotAssignee  = doltstore.ErrNotAssignee
	ErrIssueStatus  = doltstore.ErrIssueStatus
	ErrIssueBlocked = doltstore.ErrIssueBlocked
	ErrDepSelf      = doltstore.ErrDepSelf
	ErrDepCycle     = doltstore.ErrDepCycle
	ErrNoteTooLong  = doltstore.ErrNoteTooLong
)

var (
	ErrNotFound = doltstore.ErrNotFound
	ErrNoReady  = doltstore.ErrNoReady
)

// Record is one .fabrik/export/issues.jsonl line: the exported issue row
// plus its dependency edges and notes. The embedded ExportIssue keeps every
// line decoding as a plain issue object (old readers ignore the extra keys;
// new readers get the full sync snapshot).
type Record struct {
	ExportIssue
	Deps  []Dep  `json:"deps,omitempty"`
	Notes []Note `json:"notes,omitempty"`
}
