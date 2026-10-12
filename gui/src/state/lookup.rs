use crate::model::Action;

use super::AppState;

impl AppState {
    pub(super) fn lookup_focus_search(&mut self) -> Vec<Action> {
        self.lookup.lookup_search_focused = true;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_set_search_focus(&mut self, focused: bool) -> Vec<Action> {
        self.lookup.lookup_search_focused = focused;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_clear_pane_search(&mut self) -> Vec<Action> {
        self.lookup.lookup_query.clear();
        self.lookup.reset_for_repo();
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_set_query(&mut self, query: String) -> Vec<Action> {
        self.lookup.lookup_loading = !query.trim().is_empty();
        self.lookup.lookup_query = query;
        self.lookup.reset_query_results();
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_explore_loaded(
        &mut self,
        result: crate::fabrik::ExploreResult,
    ) -> Vec<Action> {
        self.lookup.lookup_explore = Some(result.clone());
        self.lookup.lookup_nodes = crate::fabrik::nodes_from_explore(&result);
        self.lookup.lookup_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_query_loaded(
        &mut self,
        result: crate::fabrik::QueryResult,
    ) -> Vec<Action> {
        self.lookup.lookup_explore = None;
        self.lookup.lookup_nodes = result.nodes;
        self.lookup.lookup_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_clear_path(&mut self) -> Vec<Action> {
        self.lookup.lookup_path = None;
        self.lookup.lookup_explain = None;
        self.lookup.lookup_anchor = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_explain_loaded(
        &mut self,
        result: crate::fabrik::ExplainResult,
    ) -> Vec<Action> {
        self.lookup.lookup_explain = Some(result);
        self.lookup.lookup_path = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_path_loaded(
        &mut self,
        result: crate::fabrik::PathResult,
    ) -> Vec<Action> {
        self.lookup.lookup_path = Some(result);
        self.lookup.lookup_explain = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_select_node(&mut self, id: Option<String>) -> Vec<Action> {
        self.lookup.lookup_anchor = id;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_refresh(&mut self) -> Vec<Action> {
        self.lookup.lookup_status_loading = true;
        if !self.lookup.lookup_query.trim().is_empty() {
            self.lookup.lookup_loading = true;
        }
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_run_graph_build(&mut self) -> Vec<Action> {
        if self.workspace.repo.is_some() && self.workspace.repo_initialized {
            self.lookup.lookup_build_in_progress = true;
            self.board.error = None;
            self.touch();
        }
        Vec::new()
    }

    pub(super) fn lookup_graph_status_loaded(
        &mut self,
        status: crate::fabrik::GraphStatus,
    ) -> Vec<Action> {
        self.lookup.lookup_status = Some(status);
        self.lookup.lookup_status_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_graph_build_done(&mut self, path: std::path::PathBuf) -> Vec<Action> {
        if self.workspace.repo.as_ref() == Some(&path) {
            self.lookup.lookup_build_in_progress = false;
            self.board.error = None;
            self.touch();
            return vec![Action::RefreshLookup];
        }
        self.lookup.lookup_build_in_progress = false;
        self.touch();
        Vec::new()
    }

}
