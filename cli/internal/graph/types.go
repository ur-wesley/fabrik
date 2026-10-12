package graph

type Node struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	Name string `json:"name"`
	Path string `json:"path"`
	Line int    `json:"line"`
	Lang string `json:"lang"`
}

type Edge struct {
	Src        string `json:"src"`
	Dst        string `json:"dst"`
	Kind       string `json:"kind"`
	Label      string `json:"label"`
	Confidence string `json:"confidence,omitempty"`
}

type BuildResult struct {
	Files   int `json:"files"`
	Parsed  int `json:"parsed"`
	Skipped int `json:"skipped"`
	Nodes   int `json:"nodes"`
	Edges   int `json:"edges"`
}

type QueryResult struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type ExplainResult struct {
	Node      Node   `json:"node"`
	Edges     []Edge `json:"edges"`
	Truncated bool   `json:"truncated"`
}

type StatusResult struct {
	Indexed     bool   `json:"indexed"`
	Files       int    `json:"files,omitempty"`
	Symbols     int    `json:"symbols,omitempty"`
	StaleFiles  int    `json:"stale_files,omitempty"`
	LastBuildAt string `json:"last_build_at,omitempty"`
}

type Snippet struct {
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Symbol    string `json:"symbol"`
	Text      string `json:"text"`
}

type ExploreFile struct {
	Path     string    `json:"path"`
	Snippets []Snippet `json:"snippets"`
}

type ExplorePath struct {
	From string   `json:"from"`
	To   string   `json:"to"`
	Hops []string `json:"hops"`
}

type BlastRadius struct {
	Callers         []string `json:"callers"`
	DependentsCount int      `json:"dependents_count"`
}

type ExploreResult struct {
	Indexed      bool          `json:"indexed"`
	Hint         string        `json:"hint,omitempty"`
	Files        []ExploreFile `json:"files,omitempty"`
	Paths        []ExplorePath `json:"paths,omitempty"`
	BlastRadius  *BlastRadius  `json:"blast_radius,omitempty"`
	Candidates   []string      `json:"candidates,omitempty"`
}

type ContextResult struct {
	Symbol       Node                `json:"symbol"`
	Callers      []Node              `json:"callers"`
	Callees      []Edge              `json:"callees"`
	Observations []ContextObservation `json:"observations"`
	Tests        []ContextTest       `json:"tests"`
}

type ContextObservation struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind"`
}

type ContextTest struct {
	Path   string `json:"path"`
	Symbol string `json:"symbol"`
}

type ImpactResult struct {
	Root    string           `json:"root"`
	ByDepth map[string][]Node `json:"by_depth"`
	Total   int              `json:"total"`
}

type DetectChangesResult struct {
	ChangedFiles     []string `json:"changed_files"`
	TouchedSymbols   []string `json:"touched_symbols"`
	ImpactedSymbols  []string `json:"impacted_symbols"`
	SuggestedChecks  []string `json:"suggested_checks"`
}
