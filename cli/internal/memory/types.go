package memory

type Session struct {
	ID           string  `json:"id"`
	Actor        string  `json:"actor,omitempty"`
	StartedAt    string  `json:"started_at"`
	EndedAt      *string `json:"ended_at"`
	Goal         string  `json:"goal"`
	Discoveries  string  `json:"discoveries"`
	Accomplished string  `json:"accomplished"`
	NextSteps    string  `json:"next_steps"`
	Files        string  `json:"files"`
}

type Observation struct {
	ID        string  `json:"id"`
	SessionID *string `json:"session_id"`
	Project   string  `json:"project"`
	Kind      string  `json:"kind"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	TopicKey  *string `json:"topic_key,omitempty"`
	SymbolID  *string `json:"symbol_id,omitempty"`
	CreatedAt string  `json:"created_at"`
}

type ContextStats struct {
	ObservationCount int      `json:"observation_count"`
	SessionCount     int      `json:"session_count"`
	Projects         []string `json:"projects"`
}

type ContextResult struct {
	Session      *Session      `json:"session"`
	Observations []Observation `json:"observations"`
	Stats        ContextStats  `json:"stats"`
}
