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

    pub fn reset_for_repo(&mut self) {
        self.reset_query_results();
        self.lookup_status = None;
        self.lookup_status_loading = false;
        self.lookup_build_in_progress = false;
    }

    pub fn clear_all(&mut self) {
        self.lookup_query.clear();
        self.reset_for_repo();
        self.lookup_search_focused = false;
    }
}
