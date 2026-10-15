use crate::model::{Action, Screen};

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
        let empty = query.trim().is_empty();
        self.lookup.lookup_query = query;
        self.lookup.reset_query_results();
        if empty {
            // Clearing a typed query restores the cached instant overview
            // without a refetch; the single busy flag goes quiet.
            if let Some(cached) = self.lookup.lookup_overview.clone() {
                self.lookup.lookup_nodes = cached.nodes;
            }
            self.lookup.lookup_loading = false;
        } else {
            self.lookup.lookup_loading = true;
        }
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

    /// Instant overview budget shared by the auto-load and the worker call.
    pub(crate) const OVERVIEW_BUDGET: u32 = 60;

    pub(super) fn lookup_fetch_overview(
        &mut self,
        repo: &std::path::Path,
        _budget: u32,
    ) -> Vec<Action> {
        if self.workspace.repo.as_deref() == Some(repo) {
            self.lookup.lookup_loading = true;
            self.touch();
        }
        Vec::new()
    }

    pub(super) fn lookup_overview_loaded(
        &mut self,
        repo: std::path::PathBuf,
        result: crate::fabrik::QueryResult,
    ) -> Vec<Action> {
        // Stale answer from a repo switch: the guard was reset, drop it.
        if self.workspace.repo.as_ref() != Some(&repo) {
            return Vec::new();
        }
        // A typed query supersedes the overview; only fill the empty view.
        if !self.lookup.lookup_query.trim().is_empty() {
            self.lookup.lookup_overview = Some(result);
            self.lookup.overview_loaded_for = Some(repo);
            self.lookup.lookup_overview_error = None;
            self.touch();
            return Vec::new();
        }
        self.lookup.lookup_explore = None;
        self.lookup.lookup_nodes.clone_from(&result.nodes);
        self.lookup.lookup_overview = Some(result);
        self.lookup.overview_loaded_for = Some(repo);
        self.lookup.lookup_overview_error = None;
        self.lookup.lookup_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_overview_failed(
        &mut self,
        repo: &std::path::Path,
        message: String,
    ) -> Vec<Action> {
        // Stale answer from a repo switch: the guard was reset, drop it.
        if self.workspace.repo.as_deref() != Some(repo) {
            return Vec::new();
        }
        // Persistent card: full message plus the failing binary path so an
        // old `fabrik` on PATH is diagnosable in place. Toast path
        // unchanged; board state untouched (this is lookup-scoped).
        let binary = crate::fabrik::fabrik_bin();
        self.lookup.lookup_overview_error = Some(format!("{message} (binary: {binary})"));
        self.lookup.lookup_loading = false;
        self.toasts.push_error(message);
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

    pub(super) fn lookup_path_loaded(&mut self, result: crate::fabrik::PathResult) -> Vec<Action> {
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
        } else if self.lookup.lookup_nodes.is_empty() {
            // Route-enter mirrors SetLookupQuery(empty): restore the cached
            // instant overview so the canvas shows without a Rebuild.
            if let Some(cached) = self.lookup.lookup_overview.clone() {
                self.lookup.lookup_nodes = cached.nodes;
                self.lookup.lookup_loading = false;
            }
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
        // Instant overview: a built, fresh graph with an empty query and an
        // empty canvas fetches the degree-ordered overview once per repo.
        // Typed queries and anchors keep their search-driven flow.
        let fresh = self
            .lookup
            .lookup_status
            .as_ref()
            .is_some_and(|s| s.indexed && s.stale_files == 0);
        if fresh
            && self.lookup.lookup_query.trim().is_empty()
            && self.lookup.lookup_nodes.is_empty()
            && self.lookup.lookup_anchor.is_none()
            && let Some(repo) = self.workspace.repo.clone()
            && (self.lookup.overview_loaded_for.as_ref() != Some(&repo)
                || self.lookup.lookup_overview.is_none())
        {
            return vec![Action::FetchGraphOverview {
                repo,
                budget: Self::OVERVIEW_BUDGET,
            }];
        }
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

    pub(super) fn lookup_fetch_context(&mut self, symbol: String) -> Vec<Action> {
        let trimmed = symbol.trim().to_owned();
        if trimmed.is_empty() {
            return Vec::new();
        }
        // Symbol-only gate (mirrors the `is_issue_id` write guards): file
        // anchors (`file:…`) fail live with `node sym:file:… not found`,
        // so never mark loading / never spawn — the header shows path
        // info + hint instead.
        if !crate::model::is_symbol_anchor(&trimmed) {
            return Vec::new();
        }
        self.lookup.lookup_context_symbol = Some(trimmed);
        self.lookup.lookup_inspect_loading = true;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_context_loaded(
        &mut self,
        result: crate::fabrik::GraphContextResult,
    ) -> Vec<Action> {
        self.lookup.lookup_context = Some(result);
        self.lookup.lookup_inspect_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_fetch_impact(&mut self, symbol: String, depth: u32) -> Vec<Action> {
        let trimmed = symbol.trim().to_owned();
        if trimmed.is_empty() {
            return Vec::new();
        }
        // Same symbol-only gate as context: file anchors never load.
        if !crate::model::is_symbol_anchor(&trimmed) {
            return Vec::new();
        }
        self.lookup.lookup_impact_symbol = Some(trimmed);
        self.lookup.lookup_impact_depth = depth.clamp(1, 10);
        self.lookup.lookup_inspect_loading = true;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_impact_loaded(
        &mut self,
        result: crate::fabrik::GraphImpactResult,
    ) -> Vec<Action> {
        self.lookup.lookup_impact = Some(result);
        self.lookup.lookup_inspect_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_fetch_detect(&mut self, base_ref: String) -> Vec<Action> {
        let trimmed = base_ref.trim().to_owned();
        self.lookup.lookup_detect_ref = Some(trimmed);
        self.lookup.lookup_inspect_loading = true;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_detect_loaded(
        &mut self,
        result: crate::fabrik::DetectChangesResult,
    ) -> Vec<Action> {
        self.lookup.lookup_detect = Some(result);
        self.lookup.lookup_inspect_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_explore_issue_seed(
        &mut self,
        issue: String,
        query: String,
    ) -> Vec<Action> {
        let id = issue.trim().to_owned();
        if id.is_empty() {
            return Vec::new();
        }
        self.lookup.lookup_issue_seed = Some(id);
        self.lookup.lookup_loading = true;
        if !query.trim().is_empty() {
            self.lookup.lookup_query = query;
        }
        // Seed-from-issue jumps straight to the graph view.
        self.workspace.screen = Screen::Lookup;
        self.touch();
        Vec::new()
    }

    pub(super) fn lookup_clear_inspect(&mut self) -> Vec<Action> {
        self.lookup.reset_inspect();
        self.touch();
        Vec::new()
    }
}
