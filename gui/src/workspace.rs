use std::fs;
use std::path::{Path, PathBuf};

use crate::discover::is_fabrik_repo;
use crate::model::{ClosedTab, RecentRepo, Screen, recent_from_path};

const MAX_RECENTS: usize = 12;
const MAX_TABS: usize = 8;
/// 1-based tab slot max for `SelectTab` (Ctrl+1..Ctrl+8, Chrome clamp).
pub const MAX_TAB_SLOT: u8 = 8;

#[derive(Debug, Clone, Default, serde::Serialize, serde::Deserialize)]
pub struct PersistedState {
    pub recents: Vec<PathBuf>,
    pub open_tabs: Vec<PathBuf>,
    #[serde(default)]
    pub selected_tab: Option<PathBuf>,
    #[serde(default)]
    pub collapsed_lanes: Vec<String>,
    #[serde(default)]
    pub mem_filter_project: bool,
    #[serde(default = "default_init_apps")]
    pub init_apps: Vec<String>,
}

pub(crate) fn default_init_apps() -> Vec<String> {
    vec!["cursor".to_owned()]
}

#[allow(clippy::struct_excessive_bools)]
pub struct WorkspaceStore {
    pub repo: Option<PathBuf>,
    pub screen: Screen,
    pub recents: Vec<PathBuf>,
    pub open_tabs: Vec<PathBuf>,
    pub closed_tab_stack: Vec<ClosedTab>,
    pub picker_open: bool,
    pub picker_query: String,
    pub picker_selected: usize,
    pub listen_live: bool,
    pub listen_error: Option<String>,
    pub repo_initialized: bool,
    pub init_in_progress: bool,
    pub init_apps: Vec<String>,
}

impl WorkspaceStore {
    pub fn from_persisted(persisted: &PersistedState) -> Self {
        let recents = sanitize_paths(&persisted.recents);
        let mut open_tabs = sanitize_paths(&persisted.open_tabs);
        let selected_restored = persisted
            .selected_tab
            .as_ref()
            .filter(|p| open_tabs.iter().any(|t| t == *p))
            .cloned();
        let repo = selected_restored.or_else(|| {
            recents
                .iter()
                .find(|p| p.is_dir())
                .or_else(|| recents.first())
                .cloned()
        });
        if let Some(ref r) = repo
            && !open_tabs.iter().any(|p| p == r)
        {
            open_tabs.insert(0, r.clone());
        }
        let repo_initialized = repo.as_ref().is_some_and(|p| is_fabrik_repo(p));
        let open_tabs: Vec<PathBuf> = open_tabs.into_iter().filter(|p| p.is_dir()).collect();

        Self {
            repo,
            screen: Screen::Board,
            recents,
            open_tabs,
            closed_tab_stack: Vec::new(),
            picker_open: false,
            picker_query: String::new(),
            picker_selected: 0,
            listen_live: false,
            listen_error: None,
            repo_initialized,
            init_in_progress: false,
            init_apps: persisted.init_apps.clone(),
        }
    }

    pub fn needs_project_init(&self) -> bool {
        self.repo.is_some() && !self.repo_initialized
    }

    pub fn pane_search_hint(&self) -> &'static str {
        match self.screen {
            Screen::Memory => "search memories",
            Screen::Lookup => "graph explore",
            Screen::Settings => "",
            _ => "search title or id",
        }
    }

    pub fn tab_entries(&self) -> Vec<RecentRepo> {
        self.open_tabs
            .iter()
            .map(|path| recent_from_path(path.clone()))
            .collect()
    }

    pub fn recent_entries(&self) -> Vec<RecentRepo> {
        self.recents
            .iter()
            .map(|path| recent_from_path(path.clone()))
            .collect()
    }

    pub fn repo_label(&self) -> String {
        self.repo.as_ref().map_or_else(
            || "pick a fabrik repo".to_owned(),
            |p| {
                p.file_name()
                    .and_then(|n| n.to_str())
                    .unwrap_or("repo")
                    .to_owned()
            },
        )
    }

    pub fn repo_path_label(&self) -> String {
        self.repo
            .as_ref()
            .and_then(|p| p.to_str())
            .unwrap_or("")
            .to_owned()
    }

    pub fn active_tab_index(&self) -> Option<usize> {
        self.repo
            .as_ref()
            .and_then(|repo| self.open_tabs.iter().position(|p| p == repo))
    }

    pub fn push_recent(&mut self, path: PathBuf) {
        self.recents.retain(|p| p != &path);
        self.recents.insert(0, path);
        if self.recents.len() > MAX_RECENTS {
            self.recents.truncate(MAX_RECENTS);
        }
    }

    pub fn push_tab(&mut self, path: PathBuf) {
        if !self.open_tabs.contains(&path) {
            self.open_tabs.push(path);
            if self.open_tabs.len() > MAX_TABS {
                self.open_tabs.remove(0);
            }
        }
    }

    pub fn set_repo_meta(&mut self, path: PathBuf) {
        self.repo_initialized = is_fabrik_repo(&path);
        self.repo = Some(path.clone());
        self.push_recent(path.clone());
        self.push_tab(path);
        self.picker_open = false;
        self.screen = Screen::Board;
    }

    pub fn clear_repo_meta(&mut self) {
        self.repo = None;
        self.screen = Screen::Board;
    }

    pub fn close_tab(&mut self, path: &Path) -> Option<PathBuf> {
        let Some(index) = self.open_tabs.iter().position(|p| p == path) else {
            return None;
        };
        let removed = self.open_tabs.remove(index);
        self.closed_tab_stack.push(ClosedTab {
            path: removed,
            index,
        });
        if self.open_tabs.is_empty() {
            self.clear_repo_meta();
            return None;
        }
        let was_active = self.repo.as_deref() == Some(path);
        if was_active {
            return self.open_tabs.last().cloned();
        }
        None
    }

    pub fn step_tab(&mut self, delta: i32) -> Option<PathBuf> {
        let len = self.open_tabs.len();
        if len < 2 {
            return None;
        }
        let Some(current) = self.active_tab_index() else {
            return None;
        };
        let next = if delta >= 0 {
            current
                .saturating_add(1)
                .checked_rem(len)
                .unwrap_or(current)
        } else {
            current
                .checked_sub(1)
                .unwrap_or_else(|| len.saturating_sub(1))
        };
        if next == current {
            return None;
        }
        self.open_tabs.get(next).cloned()
    }

    pub fn select_tab(&mut self, slot: u8) -> Option<PathBuf> {
        debug_assert!(MAX_TABS >= usize::from(MAX_TAB_SLOT));
        if !(1..=MAX_TAB_SLOT).contains(&slot) || self.open_tabs.is_empty() {
            return None;
        }
        let len = self.open_tabs.len();
        let target = if slot == MAX_TAB_SLOT && len > MAX_TABS {
            len.saturating_sub(1)
        } else {
            usize::from(slot).saturating_sub(1)
        };
        if target >= len {
            return None;
        }
        if self.active_tab_index() == Some(target) {
            return None;
        }
        self.open_tabs.get(target).cloned()
    }

    pub fn reopen_tab(&mut self) -> Option<PathBuf> {
        let Some(closed) = self.closed_tab_stack.pop() else {
            return None;
        };
        if self.open_tabs.contains(&closed.path) {
            return None;
        }
        let mut index = closed.index.min(self.open_tabs.len());
        if self.open_tabs.len() >= MAX_TABS {
            self.open_tabs.remove(0);
            index = index.saturating_sub(1).min(self.open_tabs.len());
        }
        self.open_tabs.insert(index, closed.path.clone());
        Some(closed.path)
    }

    pub fn set_picker(&mut self, open: bool) {
        if self.picker_open != open {
            self.picker_open = open;
        }
        if open {
            self.picker_query.clear();
            self.picker_selected = 0;
        }
    }

    pub fn persist_snapshot(
        &self,
        collapsed_lanes: &[String],
        mem_filter_project: bool,
    ) {
        let state = PersistedState {
            recents: self.recents.clone(),
            open_tabs: self.open_tabs.clone(),
            selected_tab: self.repo.clone(),
            collapsed_lanes: collapsed_lanes.to_vec(),
            mem_filter_project,
            init_apps: self.init_apps.clone(),
        };
        std::thread::spawn(move || {
            save_state(&state);
        });
    }
}

pub fn load_persisted() -> PersistedState {
    let Some(path) = state_path() else {
        return PersistedState::default();
    };
    let Ok(bytes) = fs::read(path) else {
        return PersistedState::default();
    };
    serde_json::from_slice(&bytes).unwrap_or_default()
}

fn state_path() -> Option<PathBuf> {
    #[cfg(windows)]
    let base = std::env::var_os("APPDATA");
    #[cfg(not(windows))]
    let base = std::env::var_os("XDG_CONFIG_HOME").or_else(|| {
        std::env::var_os("HOME").map(|h| PathBuf::from(h).join(".config").into_os_string())
    });
    base.map(|b| PathBuf::from(b).join("fabrik").join("neon").join("state.json"))
}

fn sanitize_paths(paths: &[PathBuf]) -> Vec<PathBuf> {
    paths.iter().filter(|p| p.is_dir()).cloned().collect()
}

#[cfg(test)]
const fn save_state(_state: &PersistedState) {}

#[cfg(not(test))]
fn save_state(state: &PersistedState) {
    let Some(path) = state_path() else {
        return;
    };
    if let Some(parent) = path.parent() {
        let _ = fs::create_dir_all(parent);
    }
    if let Ok(bytes) = serde_json::to_vec_pretty(state) {
        let _ = fs::write(path, bytes);
    }
}

#[cfg(test)]
mod tests {
    use std::path::PathBuf;

    use super::{PersistedState, WorkspaceStore, default_init_apps};

    #[test]
    fn recents_cap_and_dedupe() {
        let mut ws = WorkspaceStore::from_persisted(&PersistedState::default());
        ws.recents = Vec::new();
        for i in 0..13 {
            ws.push_recent(PathBuf::from(format!("r{i}")));
        }
        assert_eq!(ws.recents.len(), 12);
        assert_eq!(ws.recents[0], PathBuf::from("r12"));
        assert_eq!(ws.recents[11], PathBuf::from("r1"));
        assert!(!ws.recents.contains(&PathBuf::from("r0")));
        ws.push_recent(PathBuf::from("r5"));
        assert_eq!(ws.recents.len(), 12);
        assert_eq!(ws.recents[0], PathBuf::from("r5"));
        assert_eq!(
            ws.recents
                .iter()
                .filter(|p| *p == &PathBuf::from("r5"))
                .count(),
            1
        );
    }

    #[test]
    fn open_tabs_add_keep_order_and_close() {
        let mut ws = WorkspaceStore::from_persisted(&PersistedState::default());
        ws.open_tabs = Vec::new();
        for i in 0..9 {
            ws.push_tab(PathBuf::from(format!("tab{i}")));
        }
        assert_eq!(ws.open_tabs.len(), 8);
        assert!(!ws.open_tabs.contains(&PathBuf::from("tab0")));
        assert_eq!(ws.open_tabs[0], PathBuf::from("tab1"));
        assert_eq!(ws.open_tabs[7], PathBuf::from("tab8"));
        assert_eq!(ws.tab_entries().len(), 8);
    }

    #[test]
    fn persisted_state_roundtrips_tabs_and_selection() {
        let state = PersistedState {
            recents: vec![PathBuf::from("a")],
            open_tabs: vec![PathBuf::from("a"), PathBuf::from("b")],
            selected_tab: Some(PathBuf::from("b")),
            collapsed_lanes: Vec::new(),
            mem_filter_project: false,
            init_apps: default_init_apps(),
        };
        let bytes = serde_json::to_vec(&state).unwrap();
        let back: PersistedState = serde_json::from_slice(&bytes).unwrap();
        assert_eq!(back.open_tabs.len(), 2);
        assert_eq!(back.selected_tab, Some(PathBuf::from("b")));
    }
}
