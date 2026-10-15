use std::path::PathBuf;
use std::time::Instant;

use crate::discover::is_fabrik_repo;
use crate::model::Action;

use super::AppState;

impl AppState {
    pub(super) fn listen_connected(&mut self, live: bool) -> Vec<Action> {
        if live {
            self.toasts.clear_errors();
        }
        self.workspace.listen_live = live;
        Vec::new()
    }

    pub(super) fn listen_unavailable(&mut self, message: String) -> Vec<Action> {
        let was_live = self.workspace.listen_live;
        let changed = self.toasts.last_listen_error.as_deref() != Some(message.as_str());
        self.workspace.listen_error = Some(message.clone());
        self.workspace.listen_live = false;
        if was_live || changed {
            self.toasts.last_listen_error = Some(message.clone());
            self.toasts
                .push_error("fabrik listen unavailable".to_owned());
        }
        Vec::new()
    }

    pub(super) fn listen_live_event(
        &mut self,
        topic: String,
        id: Option<String>,
        op: Option<String>,
    ) -> Vec<Action> {
        let now = Instant::now();
        if let Some(msg) = crate::toast::live_event_message(&topic, id.as_deref(), op.as_deref()) {
            self.toasts.push_info(msg, now);
        }
        self.touch();
        match topic.as_str() {
            "issues" => vec![Action::Refresh],
            "memory" => vec![Action::RefreshMemory],
            "graph" => vec![Action::RefreshLookup],
            _ => Vec::new(),
        }
    }

    pub(super) fn listen_toggle_init_app(&mut self, app: String) -> Vec<Action> {
        if let Some(pos) = self.workspace.init_apps.iter().position(|a| a == &app) {
            self.workspace.init_apps.remove(pos);
        } else {
            self.workspace.init_apps.push(app);
        }
        if self.workspace.init_apps.is_empty() {
            self.workspace.init_apps.push("cursor".to_owned());
        }
        self.persist();
        self.touch();
        Vec::new()
    }

    pub(super) fn listen_run_project_init(&mut self) -> Vec<Action> {
        if self.workspace.repo.is_some() && !self.workspace.repo_initialized {
            self.workspace.init_in_progress = true;
            self.board.error = None;
            self.touch();
        }
        Vec::new()
    }

    pub(super) fn listen_project_init_done(&mut self, path: PathBuf) -> Vec<Action> {
        if self.workspace.repo.as_ref() == Some(&path) {
            self.workspace.repo_initialized = is_fabrik_repo(&path);
        }
        self.workspace.init_in_progress = false;
        self.board.error = None;
        self.touch();
        if self.workspace.repo_initialized {
            let mut actions = vec![Action::Refresh];
            if self.workspace.screen == crate::model::Screen::Lookup {
                actions.push(crate::model::Action::RefreshLookup);
            }
            actions
        } else {
            Vec::new()
        }
    }
}
