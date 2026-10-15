use std::path::PathBuf;

#[allow(clippy::struct_excessive_bools)]
pub struct MemoryStore {
    pub mem_context: Option<crate::fabrik::MemContext>,
    pub mem_search: String,
    pub mem_results: Vec<crate::fabrik::Observation>,
    pub mem_loading: bool,
    pub mem_filter_project: bool,
    pub mem_detail_id: Option<String>,
    pub mem_detail: Option<crate::fabrik::Observation>,
    pub mem_search_focused: bool,
    pub mem_check: Option<crate::fabrik::MemCheckResult>,
    pub mem_stats: Option<crate::fabrik::ContextStats>,
    pub mem_timeline: Option<crate::fabrik::MemTimelineResult>,
    pub mem_timeline_id: Option<String>,
    pub mem_current_project: Option<crate::fabrik::CurrentProjectResult>,
    pub mem_diagnostics_loading: bool,
    pub mem_form_open: bool,
    pub mem_form_title: String,
    pub mem_form_body: String,
    pub mem_form_kind: String,
    pub mem_form_topic: String,
    pub mem_form_symbol: String,
    pub mem_editing: Option<String>,
    pub mem_write_busy: bool,
    pub mem_write_error: Option<String>,
    pub mem_delete_confirm: Option<String>,
    pub mem_prompt_open: bool,
    pub mem_prompt_title: String,
    pub mem_prompt_capture: String,
    pub mem_kind_filter: Option<String>,
    pub mem_diagnostics_open: bool,
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
            mem_check: None,
            mem_stats: None,
            mem_timeline: None,
            mem_timeline_id: None,
            mem_current_project: None,
            mem_diagnostics_loading: false,
            mem_form_open: false,
            mem_form_title: String::new(),
            mem_form_body: String::new(),
            mem_form_kind: "discovery".to_owned(),
            mem_form_topic: String::new(),
            mem_form_symbol: String::new(),
            mem_editing: None,
            mem_write_busy: false,
            mem_write_error: None,
            mem_delete_confirm: None,
            mem_prompt_open: false,
            mem_prompt_title: String::new(),
            mem_prompt_capture: String::new(),
            mem_kind_filter: None,
            mem_diagnostics_open: false,
        }
    }

    pub fn project_filter(&self) -> Option<PathBuf> {
        // Resolve-then-filter: the `this project` label comes from the
        // current-project result (the tab's repo root), so the filter and
        // `mem save` (which labels with `RepoRoot` of the tab cwd) agree
        // even when the tab path is a subdir. Unresolved yet → unfiltered;
        // non-Fabrik repos (empty project) → global, never an empty set.
        if !self.mem_filter_project {
            return None;
        }
        self.resolved_project().map(PathBuf::from)
    }

    /// The tab's repo root as resolved by `mem current-project --json`
    /// under `-C <tab>`, or `None` while unresolved / for non-Fabrik tabs.
    #[must_use]
    pub fn resolved_project(&self) -> Option<String> {
        self.mem_current_project.as_ref().and_then(|current| {
            (!current.project.trim().is_empty()).then(|| current.project.clone())
        })
    }

    /// The tab resolved to a loaded-but-empty project (`source "none"`):
    /// not a Fabrik repo, so memories save globally.
    #[must_use]
    pub fn is_non_fabrik_repo(&self) -> bool {
        self.mem_current_project
            .as_ref()
            .is_some_and(|current| current.project.trim().is_empty())
    }

    /// Explanatory note for the `this project` toggle when it cannot
    /// filter: resolving, non-Fabrik, or no tab at all.
    #[must_use]
    pub fn project_filter_note(&self, repo: Option<&PathBuf>) -> Option<String> {
        if repo.is_none() {
            return Some("pick a repo tab first".to_owned());
        }
        if self.is_non_fabrik_repo() {
            return Some("not a Fabrik repo — memories save globally".to_owned());
        }
        if self.mem_current_project.is_none() {
            return Some("resolving project…".to_owned());
        }
        None
    }

    pub fn reset_for_repo(&mut self) {
        self.mem_detail_id = None;
        self.mem_detail = None;
        self.mem_kind_filter = None;
        self.mem_diagnostics_open = false;
        // Drop per-tab diagnostics so a repo switch never shows the
        // previous tab's numbers as steady state; the switch dispatches
        // `RefreshMemory` + `RefreshDiagnostics` (pending spinners) right
        // after this reset.
        self.mem_check = None;
        self.mem_stats = None;
        self.mem_timeline = None;
        self.mem_timeline_id = None;
        self.mem_current_project = None;
    }

    pub fn clear_all(&mut self) {
        self.mem_context = None;
        self.mem_search.clear();
        self.mem_results.clear();
        self.mem_loading = false;
        self.mem_detail_id = None;
        self.mem_detail = None;
        self.mem_search_focused = false;
        self.mem_check = None;
        self.mem_stats = None;
        self.mem_timeline = None;
        self.mem_timeline_id = None;
        self.mem_current_project = None;
        self.mem_diagnostics_loading = false;
        self.mem_form_open = false;
        self.mem_form_title.clear();
        self.mem_form_body.clear();
        self.mem_editing = None;
        self.mem_write_busy = false;
        self.mem_write_error = None;
        self.mem_delete_confirm = None;
        self.mem_prompt_open = false;
        self.mem_prompt_title.clear();
        self.mem_prompt_capture.clear();
        self.mem_kind_filter = None;
        self.mem_diagnostics_open = false;
    }

    #[must_use]
    pub fn diagnostics_summary(&self) -> String {
        // Nones are pending (loading spinners), never steady state: repo
        // switches auto-refresh, so loaded screens always carry real
        // check numbers and a resolved project.
        let check = self.mem_check.as_ref().map_or_else(
            || {
                if self.mem_diagnostics_loading {
                    "memory: loading…".to_owned()
                } else {
                    "memory: unknown".to_owned()
                }
            },
            |c| format!("memory: {} ({} obs)", c.memory, c.observations),
        );
        let project = self.mem_current_project.as_ref().map_or_else(
            || {
                if self.mem_diagnostics_loading {
                    "project: resolving…".to_owned()
                } else {
                    "project: none".to_owned()
                }
            },
            |c| {
                if c.project.trim().is_empty() {
                    "project: not a Fabrik repo".to_owned()
                } else {
                    format!("project: {} ({})", c.project, c.source)
                }
            },
        );
        format!("{check} · {project}")
    }
}

#[cfg(test)]
mod tests {
    use super::MemoryStore;

    fn current_project(project: &str, source: &str) -> crate::fabrik::CurrentProjectResult {
        crate::fabrik::CurrentProjectResult {
            project: project.to_owned(),
            source: source.to_owned(),
        }
    }

    #[test]
    fn filter_label_resolves_to_repo_root_not_tab_subdir() {
        let mut store = MemoryStore::new(true);
        // The tab path is not consulted: `current-project -C <tab>`
        // resolves the root, so filter and save always agree.
        store.mem_current_project = Some(current_project("/repo", "cwd"));
        assert_eq!(
            store.project_filter(),
            Some(std::path::PathBuf::from("/repo"))
        );
        assert_eq!(store.resolved_project().as_deref(), Some("/repo"));
        let tab = Some(std::path::PathBuf::from("/repo/sub/dir"));
        assert!(store.project_filter_note(tab.as_ref()).is_none());
    }

    #[test]
    fn non_fabrik_tab_filters_globally_with_note() {
        let mut store = MemoryStore::new(true);
        let tab = Some(std::path::PathBuf::from("/galaxy-game"));
        store.mem_current_project = Some(current_project("", "none"));
        assert!(store.is_non_fabrik_repo());
        assert_eq!(store.project_filter(), None);
        assert_eq!(
            store.project_filter_note(tab.as_ref()).as_deref(),
            Some("not a Fabrik repo — memories save globally")
        );
        assert_eq!(
            store.diagnostics_summary(),
            "memory: unknown · project: not a Fabrik repo"
        );
    }

    #[test]
    fn pending_resolve_is_unfiltered_with_loading_copy() {
        let mut store = MemoryStore::new(true);
        assert_eq!(store.project_filter(), None);
        store.mem_diagnostics_loading = true;
        assert_eq!(
            store.diagnostics_summary(),
            "memory: loading… · project: resolving…"
        );
        assert_eq!(
            store
                .project_filter_note(Some(&std::path::PathBuf::from("/repo")))
                .as_deref(),
            Some("resolving project…")
        );
    }

    #[test]
    fn reset_for_repo_drops_previous_tab_diagnostics() {
        let mut store = MemoryStore::new(true);
        store.mem_check = Some(crate::fabrik::MemCheckResult {
            memory: "ok".to_owned(),
            memory_path: "/tmp/memory.db".to_owned(),
            schema_memory: 2,
            observations: 12,
            sessions: 9,
            projects: vec!["fabrik".to_owned()],
        });
        store.mem_current_project = Some(current_project("/repo", "cwd"));
        store.reset_for_repo();
        assert!(store.mem_check.is_none());
        assert!(store.mem_current_project.is_none());
    }
}
