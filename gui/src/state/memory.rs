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

    pub(super) fn memory_context_loaded(
        &mut self,
        ctx: crate::fabrik::MemContext,
    ) -> Vec<Action> {
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
        self.memory.mem_filter_project = !self.memory.mem_filter_project;
        self.persist();
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
}
