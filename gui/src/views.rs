use crate::model::Screen;
use crate::state::AppState;
use crate::ui::chrome::RouteToolbarView;

pub fn window_nav_screen(state: &AppState) -> Screen {
    match state.workspace.screen {
        Screen::Detail => Screen::Board,
        other => other,
    }
}

pub fn route_toolbar_view(state: &AppState) -> RouteToolbarView {
    let screen = window_nav_screen(state);
    let (search, search_focused) = match screen {
        Screen::Memory => (
            state.memory.mem_search.clone(),
            state.memory.mem_search_focused,
        ),
        Screen::Lookup => (
            state.lookup.lookup_query.clone(),
            state.lookup.lookup_search_focused,
        ),
        Screen::Board | Screen::Detail => (state.board.search.clone(), state.board.search_focused),
        Screen::Settings => (String::new(), false),
    };
    let lookup_ready = screen == Screen::Lookup && state.workspace.repo_initialized;
    let show_graph_setup = state.workspace.repo_initialized
        && (state.lookup.needs_graph_build() || state.lookup.graph_stale());
    let rebuild_label = if state.lookup.needs_graph_build() {
        "Build"
    } else {
        "Rebuild"
    };
    RouteToolbarView {
        screen,
        view_mode: state.board.view_mode,
        search,
        search_focused,
        search_hint: state.workspace.pane_search_hint(),
        show_list_kanban: matches!(screen, Screen::Board | Screen::Detail),
        // Single Build entry: banner owns unbuilt/stale, toolbar only when built+fresh.
        show_rebuild: lookup_ready && !show_graph_setup,
        rebuild_label,
        rebuild_busy: state.lookup.lookup_build_in_progress,
        show_create: matches!(screen, Screen::Board | Screen::Detail)
            && state.workspace.repo_initialized,
    }
}

pub fn settings_pane_view(state: &AppState) -> crate::ui::settings::SettingsView {
    crate::ui::settings::SettingsView {
        active_path: state.workspace.repo.clone(),
        tabs: state.workspace.tab_entries(),
        last_ok_label: state.board.last_ok_label(),
        live: state.workspace.listen_live && state.board.last_ok.is_some(),
        issue_count: state.board.issue_count(),
        repo_path: state.workspace.repo_path_label(),
        loading: state.board.loading,
        stale: state.board.stale,
        init_apps: state.workspace.init_apps.clone(),
    }
}

pub fn apply_route_toolbar(state: &mut AppState, view: &RouteToolbarView) {
    match view.screen {
        Screen::Memory => {
            state.memory.mem_search = view.search.clone();
            state.memory.mem_search_focused = view.search_focused;
        }
        Screen::Lookup => {
            state.lookup.lookup_query = view.search.clone();
            state.lookup.lookup_search_focused = view.search_focused;
        }
        Screen::Board | Screen::Detail => {
            state.board.search = view.search.clone();
            state.board.search_focused = view.search_focused;
        }
        Screen::Settings => {}
    }
}

pub fn board_view(state: &AppState) -> crate::ui::board::BoardView {
    let lanes = state.board.lanes();
    let rows = lanes
        .iter()
        .find(|lane| lane.filter == state.board.filter)
        .map_or_else(Vec::new, |lane| lane.cards.clone());
    crate::ui::board::BoardView {
        current_repo: Some(state.workspace.repo_label()),
        active_path: state.workspace.repo.clone(),
        tabs: state.workspace.tab_entries(),
        last_ok_label: state.board.last_ok_label(),
        live: state.workspace.listen_live && state.board.last_ok.is_some(),
        view_mode: state.board.view_mode,
        filter: state.board.filter,
        selected: state.board.selected,
        search: state.board.search.clone(),
        rows,
        lanes,
        collapsed: state.board.collapsed_vec(),
        repo_path: state.workspace.repo_path_label(),
        issue_count: state.board.issue_count(),
        loading: state.board.loading,
        stale: state.board.stale,
        needs_init: state.workspace.needs_project_init(),
        init_in_progress: state.workspace.init_in_progress,
        git_repo: state
            .workspace
            .repo
            .as_deref()
            .is_some_and(crate::discover::is_git_repo),
    }
}

pub fn memory_pane_view(state: &AppState) -> crate::ui::memory::MemoryView<'_> {
    let mut kinds: Vec<String> = state
        .memory
        .mem_context
        .iter()
        .flat_map(|ctx| ctx.observations.iter().map(|o| o.kind.clone()))
        .chain(state.memory.mem_results.iter().map(|o| o.kind.clone()))
        .filter(|k| !k.trim().is_empty())
        .collect::<std::collections::BTreeSet<String>>()
        .into_iter()
        .collect();
    kinds.sort();
    crate::ui::memory::MemoryView {
        active_path: state.workspace.repo.clone(),
        tabs: state.workspace.tab_entries(),
        last_ok_label: state.board.last_ok_label(),
        live: state.workspace.listen_live && state.board.last_ok.is_some(),
        issue_count: state.board.issue_count(),
        repo_path: state.workspace.repo_path_label(),
        loading: state.board.loading,
        stale: state.board.stale,
        context: state.memory.mem_context.as_ref(),
        results: &state.memory.mem_results,
        mem_loading: state.memory.mem_loading,
        mem_searching: !state.memory.mem_search.trim().is_empty(),
        filter_project: state.memory.mem_filter_project,
        // Resolve-then-filter: the label is the tab's repo root from the
        // current-project result, so filter and save agree. Empty while
        // pending or on non-Fabrik tabs (see `project_filter_note`).
        project_label: state.memory.resolved_project(),
        project_filter_enabled: state.workspace.repo.is_some()
            && state.memory.resolved_project().is_some(),
        project_filter_note: state
            .memory
            .project_filter_note(state.workspace.repo.as_ref()),
        detail: state.memory.mem_detail.as_ref(),
        detail_loading: state.memory.mem_detail_id.is_some()
            && state.memory.mem_detail.is_none()
            && state.memory.mem_loading,
        selected_id: state.memory.mem_detail_id.as_deref(),
        check: state.memory.mem_check.as_ref(),
        stats: state.memory.mem_stats.as_ref(),
        timeline: state.memory.mem_timeline.as_ref(),
        timeline_id: state.memory.mem_timeline_id.as_deref(),
        current_project: state.memory.mem_current_project.as_ref(),
        diagnostics_loading: state.memory.mem_diagnostics_loading,
        diagnostics_summary: state.memory.diagnostics_summary(),
        form_open: state.memory.mem_form_open,
        editing: state.memory.mem_editing.as_deref(),
        write_busy: state.memory.mem_write_busy,
        write_error: state.memory.mem_write_error.clone(),
        delete_confirm: state.memory.mem_delete_confirm.clone(),
        prompt_open: state.memory.mem_prompt_open,
        kind_filter: state.memory.mem_kind_filter.clone(),
        kinds,
        diagnostics_open: state.memory.mem_diagnostics_open,
    }
}

pub fn lookup_pane_view(state: &AppState) -> crate::ui::lookup::LookupView<'_> {
    let graph_body = state
        .lookup
        .lookup_explore
        .as_ref()
        .filter(|e| !e.indexed && !e.hint.is_empty())
        .map(|e| e.hint.clone())
        .unwrap_or_else(|| "Run a build to index symbols and calls in this repo.".to_owned());
    let (graph_columns, graph_edges) = {
        // Instant overview edges only apply to the empty-query canvas:
        // a typed query replaces the nodes, so a stale cached overview
        // must not shadow the search results.
        let overview = if state.lookup.lookup_query.trim().is_empty() {
            state.lookup.lookup_overview.as_ref()
        } else {
            None
        };
        crate::ui::graph_view::build_columns(
            &state.lookup.lookup_nodes,
            state.lookup.lookup_anchor.as_deref(),
            state.lookup.lookup_context.as_ref(),
            state.lookup.lookup_impact.as_ref(),
            state.lookup.lookup_path.as_ref(),
            overview,
        )
    };
    crate::ui::lookup::LookupView {
        active_path: state.workspace.repo.clone(),
        tabs: state.workspace.tab_entries(),
        last_ok_label: state.board.last_ok_label(),
        live: state.workspace.listen_live && state.board.last_ok.is_some(),
        issue_count: state.board.issue_count(),
        repo_path: state.workspace.repo_path_label(),
        // Single busy flag (footer only): every loader OR-ed here so the
        // pane shows exactly one loading indicator and disables actions.
        loading: state.board.loading
            || state.lookup.lookup_loading
            || state.lookup.lookup_status_loading
            || state.lookup.lookup_inspect_loading
            || state.lookup.lookup_build_in_progress,
        stale: state.board.stale,
        needs_init: state.workspace.needs_project_init(),
        init_in_progress: state.workspace.init_in_progress,
        git_repo: state
            .workspace
            .repo
            .as_deref()
            .is_some_and(crate::discover::is_git_repo),
        show_graph_setup: state.workspace.repo_initialized
            && (state.lookup.needs_graph_build() || state.lookup.graph_stale()),
        graph_setup_rebuild: state.lookup.graph_stale()
            || state
                .lookup
                .lookup_status
                .as_ref()
                .is_some_and(|s| s.indexed),
        graph_setup_busy: state.lookup.lookup_build_in_progress,
        graph_setup_body: graph_body,
        nodes: &state.lookup.lookup_nodes,
        explain: state.lookup.lookup_explain.as_ref(),
        path: state.lookup.lookup_path.as_ref(),
        has_query: !state.lookup.lookup_query.trim().is_empty(),
        anchor: state.lookup.lookup_anchor.as_deref(),
        context: state.lookup.lookup_context.as_ref(),
        context_symbol: state.lookup.lookup_context_symbol.as_deref(),
        impact: state.lookup.lookup_impact.as_ref(),
        impact_symbol: state.lookup.lookup_impact_symbol.as_deref(),
        impact_depth: state.lookup.lookup_impact_depth,
        detect: state.lookup.lookup_detect.as_ref(),
        detect_ref: state.lookup.lookup_detect_ref.as_deref(),
        issue_seed: state.lookup.lookup_issue_seed.as_deref(),
        stale_files: state
            .lookup
            .lookup_status
            .as_ref()
            .map_or(0, |s| s.stale_files),
        status_indexed: state
            .lookup
            .lookup_status
            .as_ref()
            .is_some_and(|s| s.indexed),
        status_files: state.lookup.lookup_status.as_ref().map_or(0, |s| s.files),
        status_symbols: state.lookup.lookup_status.as_ref().map_or(0, |s| s.symbols),
        status_last_build_at: state
            .lookup
            .lookup_status
            .as_ref()
            .map_or_else(String::new, |s| s.last_build_at.clone()),
        explore: state.lookup.lookup_explore.as_ref(),
        graph_columns,
        graph_edges,
        overview_error: state.lookup.lookup_overview_error.clone(),
        overview_budget: AppState::OVERVIEW_BUDGET,
    }
}

#[cfg(test)]
mod tests {
    use std::path::PathBuf;

    use crate::fabrik::GraphStatus;
    use crate::model::{Action, Screen};
    use crate::state::AppState;

    use super::{lookup_pane_view, route_toolbar_view};

    fn lookup_screen_store() -> AppState {
        let mut state = AppState::new();
        state.workspace.open_tabs = vec![PathBuf::from("r")];
        state.workspace.repo = Some(PathBuf::from("r"));
        state.workspace.repo_initialized = true;
        state.workspace.screen = Screen::Lookup;
        state
    }

    #[test]
    fn single_build_unbuilt_toolbar_hidden_banner_shown() {
        let state = lookup_screen_store();
        assert!(state.lookup.needs_graph_build());
        let toolbar = route_toolbar_view(&state);
        let pane = lookup_pane_view(&state);
        assert!(!toolbar.show_rebuild);
        assert!(pane.show_graph_setup);
    }

    #[test]
    fn single_build_stale_banner_shown_toolbar_hidden() {
        let mut state = lookup_screen_store();
        state.reduce(Action::GraphStatusLoaded(GraphStatus {
            indexed: true,
            files: 10,
            symbols: 100,
            stale_files: 3,
            last_build_at: String::new(),
        }));
        let toolbar = route_toolbar_view(&state);
        let pane = lookup_pane_view(&state);
        assert!(pane.show_graph_setup);
        assert!(!toolbar.show_rebuild);
    }

    #[test]
    fn single_build_fresh_toolbar_shown_banner_hidden() {
        let mut state = lookup_screen_store();
        state.reduce(Action::GraphStatusLoaded(GraphStatus {
            indexed: true,
            files: 10,
            symbols: 100,
            stale_files: 0,
            last_build_at: String::new(),
        }));
        let toolbar = route_toolbar_view(&state);
        let pane = lookup_pane_view(&state);
        assert!(toolbar.show_rebuild);
        assert!(!pane.show_graph_setup);
        assert_eq!(toolbar.rebuild_label, "Rebuild");
    }

    #[test]
    fn lookup_pane_view_carries_overview_error_and_budget() {
        let mut state = lookup_screen_store();
        state.reduce(Action::LookupOverviewFailed {
            repo: PathBuf::from("r"),
            message: "fabrik exited 1: boom".to_owned(),
        });
        let pane = lookup_pane_view(&state);
        assert_eq!(pane.overview_error, state.lookup.lookup_overview_error);
        assert!(
            pane.overview_error
                .as_deref()
                .is_some_and(|card| card.contains("boom"))
        );
        assert_eq!(pane.overview_budget, AppState::OVERVIEW_BUDGET);
    }
}
