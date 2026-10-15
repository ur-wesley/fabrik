use std::time::{Duration, Instant};

use crate::model::{Action, Screen};
use crate::state::AppState;
use crate::subscribe::SubscribeService;
use crate::watch::RepoWorker;

const QUERY_DEBOUNCE_MS: u64 = 250;

fn opt_text(text: &str) -> Option<String> {
    let trimmed = text.trim();
    (!trimmed.is_empty()).then_some(trimmed.to_owned())
}

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
                // Resolve abbreviated `f-` fragments against the loaded
                // board so the worker fetches the full id.
                let resolved = self.state.board.resolve_issue_id(&id).unwrap_or(id);
                if self.state.workspace.repo.is_some() {
                    self.worker.request_detail(&resolved);
                }
                let followups = self.state.reduce(Action::OpenDetail(resolved));
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
            Action::SubmitCreate => {
                let followups = self.state.reduce(Action::SubmitCreate);
                if self.state.board.write_busy
                    && let Some(repo) = self.state.workspace.repo.clone()
                {
                    let params = crate::fabrik::IssueCreateParams {
                        title: self.state.board.create_title.clone(),
                        body: self.state.board.create_body.clone(),
                        acceptance: self.state.board.create_acceptance.clone(),
                        priority: self.state.board.create_priority,
                        kind: self.state.board.create_kind.clone(),
                    };
                    self.worker.create_issue(&repo, &params);
                }
                self.apply(followups);
            }
            Action::ClaimIssue(id) => {
                let followups = self.state.reduce(Action::ClaimIssue(id.clone()));
                if self.state.board.write_busy
                    && let Some(repo) = self.state.workspace.repo.clone()
                {
                    let by = self.state.board.op_by.clone();
                    self.worker.claim_issue(&repo, &id, &by);
                }
                self.apply(followups);
            }
            Action::UnclaimIssue(id) => {
                let followups = self.state.reduce(Action::UnclaimIssue(id.clone()));
                if self.state.board.write_busy
                    && let Some(repo) = self.state.workspace.repo.clone()
                {
                    let by = self.state.board.op_by.clone();
                    let force = self.state.board.write_force;
                    self.worker.unclaim_issue(&repo, &id, &by, force);
                }
                self.apply(followups);
            }
            Action::CloseIssue(id) => {
                let followups = self.state.reduce(Action::CloseIssue(id.clone()));
                if self.state.board.write_busy
                    && let Some(repo) = self.state.workspace.repo.clone()
                {
                    let by = self.state.board.op_by.clone();
                    let reason = self.state.board.close_reason.clone();
                    let force = self.state.board.write_force;
                    self.worker.close_issue(&repo, &id, &reason, &by, force);
                }
                self.apply(followups);
            }
            Action::SubmitDepAdd => {
                let followups = self.state.reduce(Action::SubmitDepAdd);
                if self.state.board.write_busy
                    && let (Some(repo), Some(blocked)) = (
                        self.state.workspace.repo.clone(),
                        self.state.board.detail_id.clone(),
                    )
                {
                    let blocker = self.state.board.dep_target.clone();
                    let kind = self.state.board.dep_kind.clone();
                    self.worker.add_dep(&repo, &blocked, &blocker, &kind);
                }
                self.apply(followups);
            }
            Action::WriteOk(_) | Action::MemWriteOk(_) | Action::MemDeleted(_) => {
                let followups = self.state.reduce(action);
                self.apply(followups);
            }
            Action::SubmitMemSave => {
                let followups = self.state.reduce(Action::SubmitMemSave);
                if self.state.memory.mem_write_busy {
                    let repo = self.state.workspace.repo.clone();
                    let params = crate::fabrik::MemSaveParams {
                        title: self.state.memory.mem_form_title.clone(),
                        body: self.state.memory.mem_form_body.clone(),
                        kind: self.state.memory.mem_form_kind.clone(),
                        topic_key: opt_text(&self.state.memory.mem_form_topic),
                        symbol_id: opt_text(&self.state.memory.mem_form_symbol),
                    };
                    self.worker.mem_save(repo.as_deref(), &params);
                }
                self.apply(followups);
            }
            Action::SubmitMemUpdate => {
                let followups = self.state.reduce(Action::SubmitMemUpdate);
                if self.state.memory.mem_write_busy
                    && let Some(id) = self.state.memory.mem_editing.clone()
                {
                    let params = crate::fabrik::MemUpdateParams {
                        id,
                        title: opt_text(&self.state.memory.mem_form_title),
                        body: opt_text(&self.state.memory.mem_form_body),
                        kind: opt_text(&self.state.memory.mem_form_kind),
                        topic_key: opt_text(&self.state.memory.mem_form_topic),
                        symbol_id: opt_text(&self.state.memory.mem_form_symbol),
                    };
                    self.worker.mem_update(&params);
                }
                self.apply(followups);
            }
            Action::ConfirmMemDelete => {
                let followups = self.state.reduce(Action::ConfirmMemDelete);
                if self.state.memory.mem_write_busy
                    && let Some(id) = self.state.memory.mem_delete_confirm.clone()
                {
                    self.worker.mem_delete(&id);
                }
                self.apply(followups);
            }
            Action::SubmitMemPrompt => {
                let followups = self.state.reduce(Action::SubmitMemPrompt);
                if self.state.memory.mem_write_busy {
                    let repo = self.state.workspace.repo.clone();
                    let title = self.state.memory.mem_prompt_title.clone();
                    let capture = self.state.memory.mem_prompt_capture.clone();
                    self.worker
                        .mem_save_prompt(repo.as_deref(), &title, &capture);
                }
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
                let repo = self.state.workspace.repo.clone();
                self.worker.refresh_memory(repo.as_deref());
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
            Action::FetchGraphOverview { repo, budget } => {
                self.worker.lookup_overview(&repo, budget);
                let followups = self
                    .state
                    .reduce(Action::FetchGraphOverview { repo, budget });
                self.apply(followups);
            }
            Action::LookupOverviewLoaded { .. } => {
                let followups = self.state.reduce(action);
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
            Action::FetchGraphContext(symbol) => {
                // Symbol-only gate (mirrors the `is_issue_id` write guards):
                // file anchors (`file:…`) fail live with
                // `node sym:file:… not found`, so never spawn — the header
                // shows path info + hint instead of firing.
                if crate::model::is_symbol_anchor(symbol.trim())
                    && let Some(repo) = self.state.workspace.repo.clone()
                {
                    self.worker.graph_context(&repo, &symbol);
                }
                let followups = self.state.reduce(Action::FetchGraphContext(symbol));
                self.apply(followups);
            }
            Action::GraphContextLoaded(_) => {
                let followups = self.state.reduce(action);
                self.apply(followups);
            }
            Action::FetchGraphImpact { symbol, depth } => {
                // Same symbol-only gate as context: file anchors never spawn.
                if crate::model::is_symbol_anchor(symbol.trim())
                    && let Some(repo) = self.state.workspace.repo.clone()
                {
                    self.worker.graph_impact(&repo, &symbol, depth);
                }
                let followups = self
                    .state
                    .reduce(Action::FetchGraphImpact { symbol, depth });
                self.apply(followups);
            }
            Action::GraphImpactLoaded(_) => {
                let followups = self.state.reduce(action);
                self.apply(followups);
            }
            Action::FetchDetectChanges(base_ref) => {
                if let Some(repo) = self.state.workspace.repo.clone() {
                    self.worker.detect_changes(&repo, &base_ref);
                }
                let followups = self.state.reduce(Action::FetchDetectChanges(base_ref));
                self.apply(followups);
            }
            Action::DetectChangesLoaded(_) => {
                let followups = self.state.reduce(action);
                self.apply(followups);
            }
            Action::ExploreIssueSeed { issue, query } => {
                if let Some(repo) = self.state.workspace.repo.clone() {
                    self.worker.explore_issue(&repo, &query, &issue);
                }
                let followups = self.state.reduce(Action::ExploreIssueSeed { issue, query });
                self.apply(followups);
            }
            Action::LookupClearInspect => {
                let followups = self.state.reduce(Action::LookupClearInspect);
                self.apply(followups);
            }
            Action::MemCheckLoaded(_) => {
                let followups = self.state.reduce(action);
                self.apply(followups);
            }
            Action::MemStatsLoaded(_) => {
                let followups = self.state.reduce(action);
                self.apply(followups);
            }
            Action::FetchMemTimeline { id, limit } => {
                let project = self.state.mem_project_filter();
                self.worker.mem_timeline(&id, limit, project.as_deref());
                let followups = self.state.reduce(Action::FetchMemTimeline { id, limit });
                self.apply(followups);
            }
            Action::MemTimelineLoaded(_) => {
                let followups = self.state.reduce(action);
                self.apply(followups);
            }
            Action::CurrentProjectLoaded(_) => {
                let followups = self.state.reduce(action);
                self.apply(followups);
            }
            Action::RefreshDiagnostics => {
                let repo = self.state.workspace.repo.clone();
                self.worker.mem_check();
                self.worker.mem_stats();
                self.worker.mem_current_project(repo.as_deref());
                if !self.state.memory.mem_search.trim().is_empty() {
                    self.schedule_mem_search(self.state.memory.mem_search.clone());
                } else {
                    self.worker.refresh_memory(repo.as_deref());
                }
                let followups = self.state.reduce(Action::RefreshDiagnostics);
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
            // Memory auto-refreshes on repo switch while visible so
            // per-tab diagnostics never sit on the previous tab's data
            // (the switch already reset them to pending above).
            if self.state.workspace.screen == Screen::Memory && self.state.workspace.repo.is_some()
            {
                self.apply(vec![Action::RefreshMemory, Action::RefreshDiagnostics]);
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
