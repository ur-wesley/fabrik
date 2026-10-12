use std::path::PathBuf;

use crate::model::{Action, Filter, Screen, Snapshot, ViewMode};

use super::AppState;

impl AppState {
    pub(super) fn board_set_filter(&mut self, filter: Filter) -> Vec<Action> {
        self.board.filter = filter;
        self.board.selected = 0;
        self.touch();
        Vec::new()
    }

    pub(super) fn board_set_view(&mut self, mode: ViewMode) -> Vec<Action> {
        self.board.view_mode = mode;
        self.touch();
        Vec::new()
    }

    pub(super) fn board_toggle_lane(&mut self, filter: Filter) -> Vec<Action> {
        if self.board.collapsed_lanes.contains(&filter) {
            self.board.collapsed_lanes.remove(&filter);
        } else {
            self.board.collapsed_lanes.insert(filter);
        }
        self.touch();
        self.persist();
        Vec::new()
    }

    pub(super) fn board_set_search(&mut self, query: String) -> Vec<Action> {
        self.board.search = query;
        self.board.selected = 0;
        self.touch();
        Vec::new()
    }

    pub(super) fn board_focus_search(&mut self) -> Vec<Action> {
        self.board.search_focused = true;
        self.touch();
        Vec::new()
    }

    pub(super) fn board_set_search_focus(&mut self, focused: bool) -> Vec<Action> {
        self.board.search_focused = focused;
        self.touch();
        Vec::new()
    }

    pub(super) fn board_clear_pane_search(&mut self) -> Vec<Action> {
        self.board.search.clear();
        self.board.selected = 0;
        self.touch();
        Vec::new()
    }

    pub(super) fn board_move_selection(&mut self, delta: i32) -> Vec<Action> {
        self.board.move_selection(delta);
        self.touch();
        Vec::new()
    }

    pub(super) fn board_open_selected(&mut self) -> Vec<Action> {
        let id = self
            .board
            .visible_rows()
            .get(self.board.selected)
            .map(|row| row.id.clone());
        id.map_or_else(Vec::new, |id| self.open_detail(id))
    }

    pub(super) fn board_back(&mut self) -> Vec<Action> {
        self.board.close_detail();
        self.workspace.screen = Screen::Board;
        self.touch();
        Vec::new()
    }

    pub(super) fn board_refresh(&mut self) -> Vec<Action> {
        self.board.loading = true;
        self.touch();
        Vec::new()
    }

    pub(super) fn board_snapshot(&mut self, snapshot: Snapshot) -> Vec<Action> {
        self.board
            .apply_snapshot(snapshot, self.workspace.repo.as_ref());
        self.toasts.clear_errors();
        self.touch();
        Vec::new()
    }

    pub(super) fn board_detail_loaded(
        &mut self,
        repo: PathBuf,
        detail: Box<crate::model::IssueDetail>,
    ) -> Vec<Action> {
        self.board.apply_detail_loaded(
            &repo,
            detail,
            self.workspace.repo.as_ref(),
        );
        self.touch();
        Vec::new()
    }

    pub(super) fn board_failed(
        &mut self,
        repo: Option<PathBuf>,
        message: String,
    ) -> Vec<Action> {
        let comment_failure = self.board.comment_posting;
        self.board.apply_failed(
            repo.as_deref(),
            message.clone(),
            self.workspace.repo.as_ref(),
        );
        if !comment_failure {
            self.toasts.push_error(message);
        }
        self.workspace.init_in_progress = false;
        self.memory.mem_loading = false;
        self.lookup.lookup_status_loading = false;
        self.lookup.lookup_build_in_progress = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn board_tick(&mut self) -> Vec<Action> {
        let now = std::time::Instant::now();
        let flash = self.board.apply_tick();
        let toasts = self.toasts.tick(now);
        if flash || toasts {
            self.touch();
        }
        Vec::new()
    }

    pub(super) fn board_post_comment(&mut self) -> Vec<Action> {
        if self.board.comment_draft.trim().is_empty() {
            self.board.comment_error = Some("Note is empty".to_owned());
            self.touch();
            return Vec::new();
        }
        self.board.comment_posting = true;
        self.board.comment_error = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn board_set_screen(&mut self, screen: Screen) -> Vec<Action> {
        if self.workspace.screen == Screen::Detail {
            self.board.close_detail();
        }
        self.workspace.screen = screen;
        self.touch();
        match screen {
            Screen::Memory => vec![Action::RefreshMemory],
            Screen::Lookup => vec![Action::RefreshLookup],
            _ => Vec::new(),
        }
    }
}
