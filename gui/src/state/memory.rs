use chrono::Utc;

use crate::fabrik::Observation;
use crate::model::Action;

use super::AppState;

impl AppState {
    pub(super) fn memory_focus_search(&mut self) -> Vec<Action> {
        self.memory.mem_search_focused = true;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_set_search_focus(&mut self, focused: bool) -> Vec<Action> {
        self.memory.mem_search_focused = focused;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_clear_pane_search(&mut self) -> Vec<Action> {
        self.memory.mem_search.clear();
        self.memory.mem_detail_id = None;
        self.memory.mem_detail = None;
        self.memory.mem_results.clear();
        self.touch();
        vec![Action::RefreshMemory]
    }

    pub(super) fn memory_refresh(&mut self) -> Vec<Action> {
        self.memory.mem_loading = true;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_context_loaded(&mut self, ctx: crate::fabrik::MemContext) -> Vec<Action> {
        self.memory.mem_context = Some(ctx);
        self.memory.mem_loading = false;
        self.board.last_ok = Some(Utc::now());
        self.board.error = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_search_loaded(&mut self, rows: Vec<Observation>) -> Vec<Action> {
        self.memory.mem_results = rows;
        self.memory.mem_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_toggle_project_filter(&mut self) -> Vec<Action> {
        // A resolved-but-empty project is not a Fabrik repo: the toggle
        // is disabled in the UI, and a stray toggle must not filter to an
        // empty set — stay global instead.
        if self.memory.is_non_fabrik_repo() {
            return Vec::new();
        }
        self.memory.mem_filter_project = !self.memory.mem_filter_project;
        self.persist();
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_set_kind_filter(&mut self, kind: Option<String>) -> Vec<Action> {
        self.memory.mem_kind_filter = kind.map(|k| k.trim().to_owned()).filter(|k| !k.is_empty());
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_toggle_diagnostics(&mut self) -> Vec<Action> {
        self.memory.mem_diagnostics_open = !self.memory.mem_diagnostics_open;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_select(&mut self, id: Option<String>) -> Vec<Action> {
        let loading = id.is_some();
        self.memory.mem_detail_id = id;
        self.memory.mem_detail = None;
        if loading {
            self.memory.mem_loading = true;
        }
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_get_loaded(&mut self, obs: Observation) -> Vec<Action> {
        self.memory.mem_detail = Some(obs);
        self.memory.mem_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_clear_detail(&mut self) -> Vec<Action> {
        self.memory.mem_detail_id = None;
        self.memory.mem_detail = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_set_search(&mut self, query: String) -> Vec<Action> {
        let empty = query.trim().is_empty();
        self.memory.mem_search = query;
        self.memory.mem_detail_id = None;
        self.memory.mem_detail = None;
        if empty {
            self.memory.mem_results.clear();
            self.memory.mem_loading = true;
            self.touch();
            vec![Action::RefreshMemory]
        } else {
            self.memory.mem_loading = true;
            self.touch();
            Vec::new()
        }
    }

    pub(super) fn memory_refresh_diagnostics(&mut self) -> Vec<Action> {
        self.memory.mem_diagnostics_loading = true;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_check_loaded(
        &mut self,
        result: crate::fabrik::MemCheckResult,
    ) -> Vec<Action> {
        self.memory.mem_check = Some(result);
        self.memory.mem_diagnostics_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_stats_loaded(
        &mut self,
        stats: crate::fabrik::ContextStats,
    ) -> Vec<Action> {
        self.memory.mem_stats = Some(stats);
        self.memory.mem_diagnostics_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_fetch_timeline(&mut self, id: String) -> Vec<Action> {
        let trimmed = id.trim().to_owned();
        if trimmed.is_empty() {
            return Vec::new();
        }
        self.memory.mem_timeline_id = Some(trimmed);
        self.memory.mem_diagnostics_loading = true;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_timeline_loaded(
        &mut self,
        result: crate::fabrik::MemTimelineResult,
    ) -> Vec<Action> {
        self.memory.mem_timeline = Some(result);
        self.memory.mem_diagnostics_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn memory_current_project_loaded(
        &mut self,
        result: crate::fabrik::CurrentProjectResult,
    ) -> Vec<Action> {
        self.memory.mem_current_project = Some(result);
        self.memory.mem_diagnostics_loading = false;
        self.touch();
        Vec::new()
    }

    pub(super) fn mem_open_form(&mut self) -> Vec<Action> {
        self.memory.mem_form_open = true;
        self.memory.mem_editing = None;
        self.memory.mem_form_title.clear();
        self.memory.mem_form_body.clear();
        self.memory.mem_form_kind = "discovery".to_owned();
        self.memory.mem_form_topic.clear();
        self.memory.mem_form_symbol.clear();
        self.memory.mem_write_error = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn mem_close_form(&mut self) -> Vec<Action> {
        self.memory.mem_form_open = false;
        self.memory.mem_editing = None;
        self.memory.mem_write_error = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn mem_open_edit(&mut self, id: String) -> Vec<Action> {
        let trimmed = id.trim().to_owned();
        if trimmed.is_empty() {
            return Vec::new();
        }
        let found = self
            .memory
            .mem_results
            .iter()
            .chain(
                self.memory
                    .mem_context
                    .iter()
                    .flat_map(|ctx| &ctx.observations),
            )
            .find(|obs| obs.id == trimmed)
            .cloned()
            .or_else(|| {
                if self
                    .memory
                    .mem_detail
                    .as_ref()
                    .is_some_and(|d| d.id == trimmed)
                {
                    self.memory.mem_detail.clone()
                } else {
                    None
                }
            });
        if let Some(obs) = found {
            self.memory.mem_form_title = obs.title;
            self.memory.mem_form_body = obs.body;
            self.memory.mem_form_kind = if obs.kind.trim().is_empty() {
                "discovery".to_owned()
            } else {
                obs.kind
            };
            self.memory.mem_form_topic = obs.topic_key.unwrap_or_default();
            self.memory.mem_form_symbol = obs.symbol_id.unwrap_or_default();
            self.memory.mem_editing = Some(trimmed);
            self.memory.mem_form_open = true;
            self.memory.mem_write_error = None;
            self.touch();
        }
        Vec::new()
    }

    fn mem_submit_guard(&mut self) -> bool {
        if self.memory.mem_write_busy {
            return false;
        }
        if self.memory.mem_form_title.trim().is_empty() {
            self.memory.mem_write_error = Some("Title is empty".to_owned());
            self.touch();
            return false;
        }
        if self.memory.mem_form_body.trim().is_empty() {
            self.memory.mem_write_error = Some("Body is empty".to_owned());
            self.touch();
            return false;
        }
        self.memory.mem_write_busy = true;
        self.memory.mem_write_error = None;
        self.touch();
        true
    }

    pub(super) fn mem_submit_save(&mut self) -> Vec<Action> {
        self.mem_submit_guard();
        Vec::new()
    }

    pub(super) fn mem_submit_update(&mut self) -> Vec<Action> {
        if self.memory.mem_editing.is_none() {
            self.memory.mem_write_error = Some("Nothing selected to update".to_owned());
            self.touch();
            return Vec::new();
        }
        self.mem_submit_guard();
        Vec::new()
    }

    pub(super) fn mem_request_delete(&mut self, id: String) -> Vec<Action> {
        let trimmed = id.trim().to_owned();
        if trimmed.is_empty() || !crate::model::is_observation_id(&trimmed) {
            return Vec::new();
        }
        self.memory.mem_delete_confirm = Some(trimmed);
        self.touch();
        Vec::new()
    }

    pub(super) fn mem_cancel_delete(&mut self) -> Vec<Action> {
        self.memory.mem_delete_confirm = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn mem_confirm_delete(&mut self) -> Vec<Action> {
        if self.memory.mem_write_busy {
            return Vec::new();
        }
        if self.memory.mem_delete_confirm.is_none() {
            return Vec::new();
        }
        self.memory.mem_write_busy = true;
        self.memory.mem_write_error = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn mem_deleted(&mut self, id: String) -> Vec<Action> {
        self.memory.mem_write_busy = false;
        self.memory.mem_write_error = None;
        self.memory.mem_delete_confirm = None;
        if self.memory.mem_detail_id.as_deref() == Some(id.as_str()) {
            self.memory.mem_detail_id = None;
            self.memory.mem_detail = None;
        }
        if self.memory.mem_editing.as_deref() == Some(id.as_str()) {
            self.memory.mem_editing = None;
            self.memory.mem_form_open = false;
        }
        self.toasts.push_info(
            format!("Observation {id} deleted"),
            std::time::Instant::now(),
        );
        self.touch();
        vec![Action::RefreshMemory]
    }

    pub(super) fn mem_write_ok(&mut self, message: String) -> Vec<Action> {
        self.memory.mem_write_busy = false;
        self.memory.mem_write_error = None;
        self.memory.mem_form_open = false;
        self.memory.mem_editing = None;
        self.memory.mem_prompt_open = false;
        self.memory.mem_prompt_title.clear();
        self.memory.mem_prompt_capture.clear();
        self.toasts.push_info(message, std::time::Instant::now());
        self.touch();
        vec![Action::RefreshMemory]
    }

    pub(super) fn mem_open_prompt(&mut self) -> Vec<Action> {
        self.memory.mem_prompt_open = true;
        self.memory.mem_write_error = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn mem_close_prompt(&mut self) -> Vec<Action> {
        self.memory.mem_prompt_open = false;
        self.memory.mem_write_error = None;
        self.touch();
        Vec::new()
    }

    pub(super) fn mem_submit_prompt(&mut self) -> Vec<Action> {
        if self.memory.mem_write_busy {
            return Vec::new();
        }
        if self.memory.mem_prompt_capture.trim().is_empty() {
            self.memory.mem_write_error = Some("Prompt text is empty".to_owned());
            self.touch();
            return Vec::new();
        }
        self.memory.mem_write_busy = true;
        self.memory.mem_write_error = None;
        self.touch();
        Vec::new()
    }
}

#[cfg(test)]
mod tests {
    use std::path::PathBuf;

    use crate::model::Action;

    use super::AppState;

    fn current_project(project: &str, source: &str) -> crate::fabrik::CurrentProjectResult {
        crate::fabrik::CurrentProjectResult {
            project: project.to_owned(),
            source: source.to_owned(),
        }
    }

    #[test]
    fn failed_clears_memory_and_diagnostics_loading() {
        let mut state = AppState::new();
        state.memory.mem_loading = true;
        state.memory.mem_diagnostics_loading = true;
        state.reduce(Action::Failed(None, "boom".to_owned()));
        assert!(!state.memory.mem_loading);
        assert!(!state.memory.mem_diagnostics_loading);
        assert_eq!(state.toasts.items().len(), 1);
    }

    #[test]
    fn memory_screen_open_refreshes_memory_and_diagnostics() {
        let mut state = AppState::new();
        state.workspace.repo = Some(PathBuf::from("r"));
        let followups = state.reduce(Action::SetScreen(crate::model::Screen::Memory));
        assert!(followups.iter().any(|a| matches!(a, Action::RefreshMemory)));
        assert!(
            followups
                .iter()
                .any(|a| matches!(a, Action::RefreshDiagnostics))
        );
        assert!(state.memory.mem_loading);
        assert!(state.memory.mem_diagnostics_loading);
    }

    #[test]
    fn project_toggle_ignored_for_non_fabrik_repo() {
        let mut state = AppState::new();
        state.workspace.repo = Some(PathBuf::from("galaxy"));
        state.memory.mem_filter_project = true;
        state.memory.mem_current_project = Some(current_project("", "none"));
        assert_eq!(state.mem_project_filter(), None);
        state.reduce(Action::ToggleMemProjectFilter);
        assert!(state.memory.mem_filter_project);
    }

    #[test]
    fn project_filter_uses_resolved_root() {
        let mut state = AppState::new();
        state.workspace.repo = Some(PathBuf::from("/repo/sub"));
        state.memory.mem_filter_project = true;
        state.memory.mem_current_project = Some(current_project("/repo", "cwd"));
        assert_eq!(state.mem_project_filter(), Some(PathBuf::from("/repo")));
    }
}
