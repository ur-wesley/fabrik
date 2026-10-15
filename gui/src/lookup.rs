pub struct LookupStore {
    pub lookup_query: String,
    pub lookup_explore: Option<crate::fabrik::ExploreResult>,
    pub lookup_nodes: Vec<crate::fabrik::GraphNode>,
    pub lookup_explain: Option<crate::fabrik::ExplainResult>,
    pub lookup_path: Option<crate::fabrik::PathResult>,
    pub lookup_anchor: Option<String>,
    pub lookup_loading: bool,
    pub lookup_status_loading: bool,
    pub lookup_status: Option<crate::fabrik::GraphStatus>,
    pub lookup_build_in_progress: bool,
    pub lookup_search_focused: bool,
    pub lookup_context: Option<crate::fabrik::GraphContextResult>,
    pub lookup_context_symbol: Option<String>,
    pub lookup_impact: Option<crate::fabrik::GraphImpactResult>,
    pub lookup_impact_symbol: Option<String>,
    pub lookup_impact_depth: u32,
    pub lookup_detect: Option<crate::fabrik::DetectChangesResult>,
    pub lookup_detect_ref: Option<String>,
    pub lookup_issue_seed: Option<String>,
    pub lookup_inspect_loading: bool,
    /// Cached instant overview (`graph overview --json`) shown when the
    /// query is empty. Survives typed queries so clearing the search
    /// restores it without a refetch; cleared on repo switch.
    pub lookup_overview: Option<crate::fabrik::QueryResult>,
    /// Persistent instant-overview failure (full message plus the failing
    /// binary path). Set on overview fetch failure, cleared on overview
    /// success or repo switch. The Lookup pane renders it as an error card
    /// with Retry; board snapshots never clear it (the card — not the
    /// toast — is the source of truth for overview failures).
    pub lookup_overview_error: Option<String>,
    /// Repo the cached overview was fetched for. Guards the auto-load so
    /// each repo fetches its overview exactly once.
    pub overview_loaded_for: Option<std::path::PathBuf>,
}

impl LookupStore {
    pub fn new() -> Self {
        Self {
            lookup_query: String::new(),
            lookup_explore: None,
            lookup_nodes: Vec::new(),
            lookup_explain: None,
            lookup_path: None,
            lookup_anchor: None,
            lookup_loading: false,
            lookup_status_loading: false,
            lookup_status: None,
            lookup_build_in_progress: false,
            lookup_search_focused: false,
            lookup_context: None,
            lookup_context_symbol: None,
            lookup_impact: None,
            lookup_impact_symbol: None,
            lookup_impact_depth: 3,
            lookup_detect: None,
            lookup_detect_ref: None,
            lookup_issue_seed: None,
            lookup_inspect_loading: false,
            lookup_overview: None,
            lookup_overview_error: None,
            overview_loaded_for: None,
        }
    }

    pub fn needs_graph_build(&self) -> bool {
        match &self.lookup_status {
            None => true,
            Some(s) => !s.indexed,
        }
    }

    pub fn graph_stale(&self) -> bool {
        self.lookup_status
            .as_ref()
            .is_some_and(|s| s.indexed && s.stale_files > 0)
    }

    pub fn reset_query_results(&mut self) {
        self.lookup_anchor = None;
        self.lookup_path = None;
        self.lookup_explain = None;
        self.lookup_explore = None;
        self.lookup_nodes.clear();
        self.lookup_loading = false;
    }

    pub fn reset_inspect(&mut self) {
        self.lookup_context = None;
        self.lookup_context_symbol = None;
        self.lookup_impact = None;
        self.lookup_impact_symbol = None;
        self.lookup_detect = None;
        self.lookup_detect_ref = None;
        self.lookup_issue_seed = None;
        self.lookup_inspect_loading = false;
    }

    pub fn reset_for_repo(&mut self) {
        self.reset_query_results();
        self.reset_inspect();
        self.lookup_status = None;
        self.lookup_status_loading = false;
        self.lookup_build_in_progress = false;
        self.lookup_overview = None;
        self.lookup_overview_error = None;
        self.overview_loaded_for = None;
    }

    pub fn clear_all(&mut self) {
        self.lookup_query.clear();
        self.reset_for_repo();
        self.lookup_search_focused = false;
        self.lookup_impact_depth = 3;
    }
}

/// Old-binary classifier: the shelled CLI predates `graph overview`, so the
/// failure reads like a cobra usage error instead of a graph problem.
/// Matches case-insensitively so wrapped/prefixed messages still classify.
#[must_use]
pub fn is_unknown_command(message: &str) -> bool {
    let lower = message.to_ascii_lowercase();
    ["unknown command", "unknown subcommand", "unknown flag"]
        .iter()
        .any(|needle| lower.contains(needle))
}
