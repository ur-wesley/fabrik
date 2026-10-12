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
        Screen::Board | Screen::Detail => (
            state.board.search.clone(),
            state.board.search_focused,
        ),
        Screen::Settings => (String::new(), false),
    };
    let lookup_ready = screen == Screen::Lookup && state.workspace.repo_initialized;
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
        show_rebuild: lookup_ready,
        rebuild_label,
        rebuild_busy: state.lookup.lookup_build_in_progress,
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
        repo_selected: state.workspace.repo.is_some(),
        detail: state.memory.mem_detail.as_ref(),
        detail_loading: state.memory.mem_detail_id.is_some()
            && state.memory.mem_detail.is_none()
            && state.memory.mem_loading,
        selected_id: state.memory.mem_detail_id.as_deref(),
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
    crate::ui::lookup::LookupView {
        active_path: state.workspace.repo.clone(),
        tabs: state.workspace.tab_entries(),
        last_ok_label: state.board.last_ok_label(),
        live: state.workspace.listen_live && state.board.last_ok.is_some(),
        issue_count: state.board.issue_count(),
        repo_path: state.workspace.repo_path_label(),
        loading: state.board.loading,
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
        lookup_loading: state.lookup.lookup_loading || state.lookup.lookup_status_loading,
        has_query: !state.lookup.lookup_query.trim().is_empty(),
        anchor: state.lookup.lookup_anchor.as_deref(),
    }
}
