package doltstore

type Issue struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Body        string  `json:"body"`
	Acceptance  string  `json:"acceptance"`
	Status      string  `json:"status"`
	Priority    int     `json:"priority"`
	Kind        string  `json:"kind"`
	Assignee    *string `json:"assignee"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
	StartedAt   *string `json:"started_at"`
	ClosedAt    *string `json:"closed_at"`
	Ready       bool    `json:"ready"`
	Blockers    []IssueRef `json:"blockers,omitempty"`
	Blocking    []IssueRef `json:"blocking,omitempty"`
	Notes       []Note     `json:"notes,omitempty"`
}

type IssueRef struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
	Kind     string `json:"kind"`
}

type Note struct {
	ID        string `json:"id"`
	IssueID   string `json:"issue_id,omitempty"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

// Dep is one issue_deps edge. It backs the export snapshot and the sync
// merge (union by triple); added for the SQLite sync path, also implemented
// by the legacy Dolt store so `dolt pull` keeps exporting.
type Dep struct {
	BlockerID string `json:"blocker_id"`
	BlockedID string `json:"blocked_id"`
	Kind      string `json:"kind"`
}

type CloseResult struct {
	Issue  Issue
	NoteID string
}

type DepResult struct {
	Issue         Issue
	AlreadyExists bool
}

type ExportIssue struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Body       string  `json:"body"`
	Acceptance string  `json:"acceptance"`
	Status     string  `json:"status"`
	Priority   int     `json:"priority"`
	Kind       string  `json:"kind"`
	Assignee   *string `json:"assignee"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
	StartedAt  *string `json:"started_at"`
	ClosedAt   *string `json:"closed_at"`
}

type IssueSlim struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
	Kind     string `json:"kind"`
	Ready    bool   `json:"ready,omitempty"`
}

func (i Issue) Slim() IssueSlim {
	return IssueSlim{
		ID: i.ID, Title: i.Title, Status: i.Status, Priority: i.Priority, Kind: i.Kind, Ready: i.Ready,
	}
}

func SlimIssues(issues []Issue) []IssueSlim {
	out := make([]IssueSlim, len(issues))
	for i, iss := range issues {
		out[i] = iss.Slim()
	}
	return out
}

func (i Issue) Export() ExportIssue {
	return ExportIssue{
		ID: i.ID, Title: i.Title, Body: i.Body, Acceptance: i.Acceptance,
		Status: i.Status, Priority: i.Priority, Kind: i.Kind, Assignee: i.Assignee,
		CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt, StartedAt: i.StartedAt, ClosedAt: i.ClosedAt,
	}
}
