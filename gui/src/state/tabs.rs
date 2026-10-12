use std::path::PathBuf;

use crate::model::Action;

use super::AppState;

impl AppState {
    pub(super) fn tabs_close_tab(&mut self, path: PathBuf) -> Vec<Action> {
        let next = self.workspace.close_tab(&path);
        if self.workspace.repo.is_none() {
            self.board.clear_all();
            self.memory.clear_all();
            self.lookup.clear_all();
            self.touch();
            self.persist();
            return Vec::new();
        }
        if let Some(next_path) = next {
            self.apply_repo(next_path);
            return vec![Action::Refresh];
        }
        self.touch();
        self.persist();
        Vec::new()
    }

    pub(super) fn tabs_step_tab(&mut self, delta: i32) -> Vec<Action> {
        if let Some(path) = self.workspace.step_tab(delta) {
            self.apply_repo(path);
            return vec![Action::Refresh];
        }
        Vec::new()
    }

    pub(super) fn tabs_select_tab(&mut self, slot: u8) -> Vec<Action> {
        if let Some(path) = self.workspace.select_tab(slot) {
            self.apply_repo(path);
            return vec![Action::Refresh];
        }
        Vec::new()
    }

    pub(super) fn tabs_reopen_tab(&mut self) -> Vec<Action> {
        if let Some(path) = self.workspace.reopen_tab() {
            self.apply_repo(path);
            return vec![Action::Refresh];
        }
        self.touch();
        Vec::new()
    }

    pub(super) fn tabs_close_picker(&mut self) -> Vec<Action> {
        self.workspace.set_picker(false);
        self.touch();
        Vec::new()
    }

    pub(super) fn tabs_toggle_picker(&mut self) -> Vec<Action> {
        let open = !self.workspace.picker_open;
        self.workspace.set_picker(open);
        self.touch();
        Vec::new()
    }

    pub(super) fn tabs_set_picker_query(&mut self, query: String) -> Vec<Action> {
        self.workspace.picker_query = query;
        self.workspace.picker_selected = 0;
        self.touch();
        Vec::new()
    }

    pub(super) fn tabs_move_picker(&mut self, delta: i32) -> Vec<Action> {
        let current = i32::try_from(self.workspace.picker_selected).unwrap_or(0);
        let next = current.saturating_add(delta).max(0);
        let next_selected = usize::try_from(next).unwrap_or(0);
        if next_selected != self.workspace.picker_selected {
            self.workspace.picker_selected = next_selected;
            self.touch();
        }
        Vec::new()
    }
}
