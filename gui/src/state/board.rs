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
        // Board toasts clear on board success, but lookup-scoped errors
        // (the overview card) survive: the card is the source of truth for
        // overview failures, not the toast.
        self.toasts.clear_errors();
        self.touch();
        Vec::new()
    }

    pub(super) fn board_detail_loaded(
        &mut self,
        repo: PathBuf,
        detail: Box<crate::model::IssueDetail>,
    ) -> Vec<Action> {
        self.board
            .apply_detail_loaded(&repo, detail, self.workspace.repo.as_ref());
        self.touch();
        Vec::new()
    }

    pub(super) fn board_failed(&mut self, repo: Option<PathBuf>, message: String) -> Vec<Action> {
        let comment_failure = self.board.comment_posting;
        if self.board.write_busy {
            self.board.write_busy = false;
            self.board.write_error = Some(message.clone());
        }
        if self.memory.mem_write_busy {
            self.memory.mem_write_busy = false;
            self.memory.mem_write_error = Some(message.clone());
        }
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
        self.memory.mem_diagnostics_loading = false;
        self.lookup.lookup_status_loading = false;
        self.lookup.lookup_build_in_progress = false;
        self.lookup.lookup_loading = false;
        self.lookup.lookup_inspect_loading = false;
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
            // Memory auto-refreshes content + diagnostics on open so the
            // pane never sits on never-loaded Nones; repo switches while
            // on Memory refresh the same way via `Effects`. Pending flags
            // go up here so the spinners show before the worker answers
            // (mirrors the Lookup arm below).
            Screen::Memory => {
                self.memory.mem_loading = true;
                self.memory.mem_diagnostics_loading = true;
                vec![Action::RefreshMemory, Action::RefreshDiagnostics]
            }
            Screen::Lookup => {
                // Mirror `lookup_refresh` so the pane shows its spinner
                // before the worker answers. Route-enter also restores the
                // cached overview so the canvas shows without a Rebuild;
                // the RefreshLookup followup re-applies the same restore.
                self.lookup.lookup_status_loading = true;
                if !self.lookup.lookup_query.trim().is_empty() {
                    self.lookup.lookup_loading = true;
                } else if self.lookup.lookup_nodes.is_empty() {
                    if let Some(cached) = self.lookup.lookup_overview.clone() {
                        self.lookup.lookup_nodes = cached.nodes;
                        self.lookup.lookup_loading = false;
                    }
                }
                vec![Action::RefreshLookup]
            }
            _ => Vec::new(),
        }
    }

    pub(super) fn write_open_create(&mut self) -> Vec<Action> {
        self.board.create_open = true;
        self.board.write_error = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn write_close_create(&mut self) -> Vec<Action> {
        self.board.create_open = false;
        self.board.write_error = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn write_submit_create(&mut self) -> Vec<Action> {
        if self.board.write_busy {
            return Vec::new();
        }
        if self.workspace.repo.is_none() || !self.workspace.repo_initialized {
            self.board.write_error = Some("Pick a fabrik repo first".to_owned());
            self.touch();
            return Vec::new();
        }
        if self.board.create_title.trim().is_empty() {
            self.board.write_error = Some("Title is empty".to_owned());
            self.touch();
            return Vec::new();
        }
        self.board.write_busy = true;
        self.board.write_error = None;
        self.touch();
        Vec::new()
    }

    fn write_submit_op(&mut self, id: &str) -> bool {
        if self.board.write_busy {
            return false;
        }
        if self.workspace.repo.is_none() || !self.workspace.repo_initialized {
            self.board.write_error = Some("Pick a fabrik repo first".to_owned());
            self.touch();
            return false;
        }
        if !crate::model::is_issue_id(id.trim()) {
            self.board.write_error = Some(format!("Invalid issue id {id:?}"));
            self.touch();
            return false;
        }
        self.board.write_busy = true;
        self.board.write_error = None;
        self.touch();
        true
    }

    pub(super) fn write_claim(&mut self, id: String) -> Vec<Action> {
        self.write_submit_op(&id);
        Vec::new()
    }

    pub(super) fn write_unclaim(&mut self, id: String) -> Vec<Action> {
        self.write_submit_op(&id);
        Vec::new()
    }

    pub(super) fn write_close(&mut self, id: String) -> Vec<Action> {
        self.write_submit_op(&id);
        Vec::new()
    }

    pub(super) fn write_submit_dep(&mut self) -> Vec<Action> {
        if self.board.write_busy {
            return Vec::new();
        }
        if self.board.detail_id.is_none() {
            self.board.write_error = Some("Open an issue first".to_owned());
            self.touch();
            return Vec::new();
        }
        if self.workspace.repo.is_none() || !self.workspace.repo_initialized {
            self.board.write_error = Some("Pick a fabrik repo first".to_owned());
            self.touch();
            return Vec::new();
        }
        if !crate::model::is_issue_id(self.board.dep_target.trim()) {
            self.board.write_error = Some(format!(
                "Invalid blocker id {:?}",
                self.board.dep_target.trim()
            ));
            self.touch();
            return Vec::new();
        }
        self.board.dep_target = self.board.dep_target.trim().to_owned();
        self.board.write_busy = true;
        self.board.write_error = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn write_ok(&mut self, message: String) -> Vec<Action> {
        self.board.write_busy = false;
        self.board.write_error = None;
        self.board.create_open = false;
        self.board.create_title.clear();
        self.board.create_body.clear();
        self.board.create_acceptance.clear();
        self.board.close_reason.clear();
        self.board.dep_target.clear();
        self.toasts.push_info(message, std::time::Instant::now());
        self.touch();
        // The CLI publishes a listen event for every write, but the
        // listener may be down: refresh explicitly so the board always
        // converges after a write.
        vec![Action::Refresh]
    }
}
