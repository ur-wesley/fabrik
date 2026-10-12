use std::path::PathBuf;

pub struct MemoryStore {
    pub mem_context: Option<crate::fabrik::MemContext>,
    pub mem_search: String,
    pub mem_results: Vec<crate::fabrik::Observation>,
    pub mem_loading: bool,
    pub mem_filter_project: bool,
    pub mem_detail_id: Option<String>,
    pub mem_detail: Option<crate::fabrik::Observation>,
    pub mem_search_focused: bool,
}

impl MemoryStore {
    pub fn new(mem_filter_project: bool) -> Self {
        Self {
            mem_context: None,
            mem_search: String::new(),
            mem_results: Vec::new(),
            mem_loading: false,
            mem_filter_project,
            mem_detail_id: None,
            mem_detail: None,
            mem_search_focused: false,
        }
    }

    pub fn project_filter(&self, repo: &Option<PathBuf>) -> Option<PathBuf> {
        if self.mem_filter_project {
            repo.clone()
        } else {
            None
        }
    }

    pub fn reset_for_repo(&mut self) {
        self.mem_detail_id = None;
        self.mem_detail = None;
    }

    pub fn clear_all(&mut self) {
        self.mem_context = None;
        self.mem_search.clear();
        self.mem_results.clear();
        self.mem_loading = false;
        self.mem_detail_id = None;
        self.mem_detail = None;
        self.mem_search_focused = false;
    }

}
