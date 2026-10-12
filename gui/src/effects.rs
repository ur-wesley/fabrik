use std::time::{Duration, Instant};

use crate::model::{Action, Screen};
use crate::state::AppState;
use crate::subscribe::SubscribeService;
use crate::watch::RepoWorker;

const QUERY_DEBOUNCE_MS: u64 = 250;

pub struct Effects<'a> {
    pub state: &'a mut AppState,
    pub worker: &'a mut RepoWorker,
    pub subscribe: &'a mut SubscribeService,
    pub mem_query_pending: &'a mut Option<(String, Instant)>,
    pub lookup_query_pending: &'a mut Option<(String, Instant)>,
}

impl Effects<'_> {
    pub fn apply(&mut self, actions: Vec<Action>) {
        for action in actions {
            self.apply_one(action);
        }
    }

    fn apply_one(&mut self, action: Action) {
        match action {
            Action::OpenDetail(id) => {
                if self.state.workspace.repo.is_some() {
                    self.worker.request_detail(&id);
                }
                let followups = self.state.reduce(Action::OpenDetail(id));
                self.apply(followups);
            }
            Action::PostComment => {
                if let (Some(_), Some(id)) = (
                    self.state.workspace.repo.clone(),
                    self.state.board.detail_id.clone(),
                ) {
                    let body = self.state.board.comment_draft.clone();
                    if !body.trim().is_empty() && !self.state.board.comment_posting {
                        self.worker.post_comment(&id, &body);
                    }
                }
                let followups = self.state.reduce(Action::PostComment);
                self.apply(followups);
            }
            Action::Refresh => {
                if self.state.workspace.repo_initialized {
                    self.worker.schedule_refresh();
                }
                let followups = self.state.reduce(Action::Refresh);
                self.apply(followups);
            }
            Action::RunProjectInit => {
                let followups = self.state.reduce(Action::RunProjectInit);
                if self.state.workspace.init_in_progress {
                    if let Some(repo) = self.state.workspace.repo.clone() {
                        let apps = self.state.workspace.init_apps.clone();
                        self.worker.init_repo(&repo, &apps);
                    }
                }
                self.apply(followups);
            }
            Action::ProjectInitDone(path) => {
                let followups = self.state.reduce(Action::ProjectInitDone(path));
                if let Some(repo) = self.state.workspace.repo.clone() {
                    self.worker.set_repo(&repo);
                    self.subscribe.set_repo(Some(&repo));
                }
                self.apply(followups);
            }
            Action::RefreshMemory => {
                self.worker.refresh_memory();
                let followups = self.state.reduce(Action::RefreshMemory);
                if !self.state.memory.mem_search.trim().is_empty() {
                    self.schedule_mem_search(self.state.memory.mem_search.clone());
                }
                self.apply(followups);
            }
            Action::SetMemSearch(query) => {
                let empty = query.trim().is_empty();
                let followups = self.state.reduce(Action::SetMemSearch(query));
                if empty {
                    *self.mem_query_pending = None;
                } else {
                    self.schedule_mem_search(self.state.memory.mem_search.clone());
                }
                self.apply(followups);
            }
            Action::ToggleMemProjectFilter => {
                let followups = self.state.reduce(Action::ToggleMemProjectFilter);
                if !self.state.memory.mem_search.trim().is_empty() {
                    self.schedule_mem_search(self.state.memory.mem_search.clone());
                }
                self.apply(followups);
            }
            Action::MemSelect(id) => {
                if let Some(ref obs_id) = id {
                    self.worker.mem_get(obs_id);
                }
                let followups = self.state.reduce(Action::MemSelect(id));
                self.apply(followups);
            }
            Action::SetLookupQuery(query) => {
                let empty = query.trim().is_empty();
                let followups = self.state.reduce(Action::SetLookupQuery(query));
                if empty {
                    *self.lookup_query_pending = None;
                } else if self.state.workspace.repo.is_some() {
                    self.schedule_lookup_search(self.state.lookup.lookup_query.clone());
                }
                self.apply(followups);
            }
            Action::RefreshLookup => {
                if self.state.workspace.repo_initialized {
                    if let Some(repo) = self.state.workspace.repo.clone() {
                        self.worker.graph_status(&repo);
                        let q = self.state.lookup.lookup_query.clone();
                        if !q.trim().is_empty() {
                            self.worker.lookup_explore(&repo, &q);
                        }
                    }
                }
                let followups = self.state.reduce(Action::RefreshLookup);
                self.apply(followups);
            }
            Action::RunGraphBuild => {
                let followups = self.state.reduce(Action::RunGraphBuild);
                if self.state.lookup.lookup_build_in_progress {
                    if let Some(repo) = self.state.workspace.repo.clone() {
                        self.worker.graph_build(&repo);
                    }
                }
                self.apply(followups);
            }
            Action::GraphBuildDone(path) => {
                let followups = self.state.reduce(Action::GraphBuildDone(path));
                self.apply(followups);
            }
            Action::LookupSelectNode(id) => {
                let repo = self.state.workspace.repo.clone();
                let prev = self.state.lookup.lookup_anchor.clone();
                let followups = self.state.reduce(Action::LookupSelectNode(id.clone()));
                if let (Some(repo), Some(id)) = (repo, id) {
                    if let Some(from) = prev {
                        if from != id {
                            self.worker.lookup_path(&repo, &from, &id);
                        } else {
                            self.worker.lookup_explain(&repo, &id);
                        }
                    } else {
                        self.worker.lookup_explain(&repo, &id);
                    }
                }
                self.apply(followups);
            }
            Action::RepoPicked(path) => {
                self.apply_repo_switch(Action::RepoPicked(path));
            }
            Action::PickRecent(path) => {
                self.apply_repo_switch(Action::PickRecent(path));
            }
            Action::CloseTab(path) => {
                self.apply_repo_switch(Action::CloseTab(path));
            }
            Action::NextTab | Action::PrevTab | Action::SelectTab(_) | Action::ReopenTab => {
                self.apply_repo_switch(action);
            }
            Action::Tick => {
                self.flush_query_debounce();
                let followups = self.state.reduce(Action::Tick);
                self.apply(followups);
            }
            Action::GraphStatusLoaded(status) => {
                let followups = self.state.reduce(Action::GraphStatusLoaded(status));
                self.apply(followups);
            }
            other => {
                if let Action::SetScreen(screen) = &other {
                    self.subscribe.set_lookup_active(*screen == Screen::Lookup);
                }
                let followups = self.state.reduce(other);
                self.apply(followups);
            }
        }
    }

    fn schedule_mem_search(&mut self, query: String) {
        *self.mem_query_pending = Some((query, Instant::now()));
    }

    fn schedule_lookup_search(&mut self, query: String) {
        *self.lookup_query_pending = Some((query, Instant::now()));
    }

    pub fn flush_query_debounce(&mut self) {
        let wait = Duration::from_millis(QUERY_DEBOUNCE_MS);
        let now = Instant::now();
        if let Some((query, started)) = self.mem_query_pending.clone() {
            if now.duration_since(started) >= wait {
                *self.mem_query_pending = None;
                let project = self.state.mem_project_filter();
                self.worker.mem_search(&query, project.as_deref());
            }
        }
        if let Some((query, started)) = self.lookup_query_pending.clone() {
            if now.duration_since(started) >= wait {
                *self.lookup_query_pending = None;
                if let Some(repo) = self.state.workspace.repo.clone() {
                    self.worker.lookup_explore(&repo, &query);
                }
            }
        }
    }

    fn apply_repo_switch(&mut self, action: Action) {
        let before = self.state.workspace.repo.clone();
        let followups = self.state.reduce(action);
        if self.state.workspace.repo != before {
            if let Some(repo) = self.state.workspace.repo.clone() {
                self.worker.set_repo(&repo);
                self.subscribe.set_repo(Some(&repo));
            } else {
                self.worker.clear();
                self.subscribe.set_repo(None);
            }
            self.subscribe
                .set_lookup_active(self.state.workspace.screen == Screen::Lookup);
            if self.state.workspace.screen == Screen::Lookup
                && self.state.workspace.repo_initialized
            {
                self.apply(vec![Action::RefreshLookup]);
            }
        }
        for followup in followups {
            match followup {
                Action::Refresh => {
                    let nested = self.state.reduce(Action::Refresh);
                    self.apply(nested);
                }
                other => self.apply(vec![other]),
            }
        }
    }
}
