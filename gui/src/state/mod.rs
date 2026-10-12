use std::collections::HashSet;
use std::path::PathBuf;

use crate::board::BoardStore;
use crate::discover::is_fabrik_repo;
use crate::lookup::LookupStore;
use crate::memory::MemoryStore;
use crate::model::{Action, Filter, Screen};
use crate::toast::ToastStore;
use crate::workspace::{WorkspaceStore, load_persisted};

mod board;
mod listen;
mod lookup;
mod memory;
mod tabs;

#[cfg(test)]
mod tests;

pub struct AppState {
    pub workspace: WorkspaceStore,
    pub board: BoardStore,
    pub memory: MemoryStore,
    pub lookup: LookupStore,
    pub toasts: ToastStore,
    pub revision: u64,
}

pub type Store = AppState;

impl AppState {
    pub fn new() -> Self {
        let persisted = load_persisted();
        let collapsed_lanes: HashSet<Filter> = persisted
            .collapsed_lanes
            .iter()
            .filter_map(|key| Filter::from_key(key))
            .collect();
        Self {
            workspace: WorkspaceStore::from_persisted(&persisted),
            board: BoardStore::new(collapsed_lanes),
            memory: MemoryStore::new(persisted.mem_filter_project),
            lookup: LookupStore::new(),
            toasts: ToastStore::default(),
            revision: 0,
        }
    }

    pub fn mem_project_filter(&self) -> Option<PathBuf> {
        self.memory.project_filter(&self.workspace.repo)
    }

    pub fn tab_entries(&self) -> Vec<crate::model::RecentRepo> {
        self.workspace.tab_entries()
    }

    pub fn recent_entries(&self) -> Vec<crate::model::RecentRepo> {
        self.workspace.recent_entries()
    }

    pub fn persist(&self) {
        self.workspace.persist_snapshot(
            &self.board.collapsed_lane_keys(),
            self.memory.mem_filter_project,
        );
    }

    pub(super) fn touch(&mut self) {
        self.revision = self.revision.saturating_add(1);
    }

    pub(super) fn set_workspace_guarded(&mut self, path: PathBuf) -> Vec<Action> {
        let Some(path) = crate::discover::workspace_from_picked(path) else {
            let message = "Not a valid folder".to_owned();
            self.board.error = Some(message.clone());
            self.board.loading = false;
            self.toasts.push_error(message);
            self.touch();
            return Vec::new();
        };
        self.apply_repo(path);
        if self.workspace.repo_initialized {
            vec![Action::Refresh]
        } else {
            Vec::new()
        }
    }

    pub(super) fn apply_repo(&mut self, path: PathBuf) {
        let initialized = is_fabrik_repo(&path);
        self.workspace.set_repo_meta(path);
        self.workspace.repo_initialized = initialized;
        self.board
            .reset_for_repo(self.workspace.repo.as_ref(), initialized);
        self.memory.reset_for_repo();
        self.lookup.reset_for_repo();
        self.touch();
        self.persist();
    }

    pub(super) fn open_detail(&mut self, id: String) -> Vec<Action> {
        self.board.open_detail(&id, self.workspace.repo.as_ref());
        self.workspace.screen = Screen::Detail;
        self.touch();
        Vec::new()
    }

    pub fn reduce(&mut self, action: Action) -> Vec<Action> {
        match action {
            Action::RepoPicked(path) => self.set_workspace_guarded(path),
            Action::PickRecent(path) => self.set_workspace_guarded(path),
            Action::SetFilter(filter) => self.board_set_filter(filter),
            Action::SetView(mode) => self.board_set_view(mode),
            Action::ToggleLane(filter) => self.board_toggle_lane(filter),
            Action::SetSearch(query) => self.board_set_search(query),
            Action::FocusSearch => self.board_focus_search(),
            Action::SetBoardSearchFocus(focused) => self.board_set_search_focus(focused),
            Action::FocusMemSearch => self.memory_focus_search(),
            Action::SetMemSearchFocus(focused) => self.memory_set_search_focus(focused),
            Action::FocusLookupSearch => self.lookup_focus_search(),
            Action::SetLookupSearchFocus(focused) => self.lookup_set_search_focus(focused),
            Action::ClearPaneSearch => match self.workspace.screen {
                Screen::Memory => self.memory_clear_pane_search(),
                Screen::Lookup => self.lookup_clear_pane_search(),
                _ => self.board_clear_pane_search(),
            },
            Action::MoveSelection(delta) => self.board_move_selection(delta),
            Action::OpenSelected => self.board_open_selected(),
            Action::OpenDetail(id) => self.open_detail(id),
            Action::Back => self.board_back(),
            Action::Refresh => self.board_refresh(),
            Action::Snapshot(snapshot) => self.board_snapshot(snapshot),
            Action::DetailLoaded(repo, detail) => self.board_detail_loaded(repo, detail),
            Action::Failed(repo, message) => self.board_failed(repo, message),
            Action::Tick => self.board_tick(),
            Action::CloseTab(path) => self.tabs_close_tab(path),
            Action::NextTab => self.tabs_step_tab(1),
            Action::PrevTab => self.tabs_step_tab(-1),
            Action::SelectTab(slot) => self.tabs_select_tab(slot),
            Action::ReopenTab => self.tabs_reopen_tab(),
            Action::ClosePicker => self.tabs_close_picker(),
            Action::TogglePicker => self.tabs_toggle_picker(),
            Action::SetPickerQuery(query) => self.tabs_set_picker_query(query),
            Action::MovePicker(delta) => self.tabs_move_picker(delta),
            Action::PostComment => self.board_post_comment(),
            Action::SetScreen(screen) => self.board_set_screen(screen),
            Action::RefreshMemory => self.memory_refresh(),
            Action::MemContextLoaded(ctx) => self.memory_context_loaded(ctx),
            Action::MemSearchLoaded(rows) => self.memory_search_loaded(rows),
            Action::ToggleMemProjectFilter => self.memory_toggle_project_filter(),
            Action::MemSelect(id) => self.memory_select(id),
            Action::MemGetLoaded(obs) => self.memory_get_loaded(obs),
            Action::MemClearDetail => self.memory_clear_detail(),
            Action::SetMemSearch(query) => self.memory_set_search(query),
            Action::SetLookupQuery(query) => self.lookup_set_query(query),
            Action::LookupExploreLoaded(result) => self.lookup_explore_loaded(result),
            Action::LookupQueryLoaded(result) => self.lookup_query_loaded(result),
            Action::LookupClearPath => self.lookup_clear_path(),
            Action::LookupExplainLoaded(result) => self.lookup_explain_loaded(result),
            Action::LookupPathLoaded(result) => self.lookup_path_loaded(result),
            Action::LookupSelectNode(id) => self.lookup_select_node(id),
            Action::ListenConnected(live) => self.listen_connected(live),
            Action::ListenUnavailable(message) => self.listen_unavailable(message),
            Action::LiveEvent { topic, id, op } => self.listen_live_event(topic, id, op),
            Action::RefreshLookup => self.lookup_refresh(),
            Action::RunGraphBuild => self.lookup_run_graph_build(),
            Action::GraphStatusLoaded(status) => self.lookup_graph_status_loaded(status),
            Action::GraphBuildDone(path) => self.lookup_graph_build_done(path),
            Action::ToggleInitApp(app) => self.listen_toggle_init_app(app),
            Action::RunProjectInit => self.listen_run_project_init(),
            Action::ProjectInitDone(path) => self.listen_project_init_done(path),
        }
    }
}
