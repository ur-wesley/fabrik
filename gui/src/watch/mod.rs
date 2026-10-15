use std::path::{Path, PathBuf};
use std::sync::mpsc::{Receiver, Sender, channel};
use std::thread;

use notify::{Config, Event, RecommendedWatcher, Watcher};

use crate::model::Action;

mod deferred;
mod dispatch;
mod msg;
mod overview;
mod writes;

use dispatch::worker_loop;
use msg::WorkerMsg;

pub struct WatchService {
    worker_tx: Sender<WorkerMsg>,
    action_rx: Receiver<Action>,
    watcher: Option<RecommendedWatcher>,
    watched_repo: Option<PathBuf>,
}

pub type RepoWorker = WatchService;

impl WatchService {
    pub fn new() -> Self {
        let (worker_tx, worker_rx) = channel::<WorkerMsg>();
        let (action_tx, action_rx) = channel::<Action>();

        thread::spawn(move || {
            worker_loop(&worker_rx, &action_tx);
        });

        Self {
            worker_tx,
            action_rx,
            watcher: None,
            watched_repo: None,
        }
    }

    pub fn set_repo(&mut self, repo: &Path) {
        // Always forward SetRepo: the worker thread's `current_repo` lags
        // `watched_repo`, so an early-return on equality would permanently
        // desync the worker after a dropped/queued switch.
        self.watched_repo = Some(repo.to_path_buf());
        if let Err(err) = self.worker_tx.send(WorkerMsg::SetRepo(repo.to_path_buf())) {
            eprintln!("[worker] SetRepo send failed: {err:?}");
        }
        self.watcher = None;
        let dolt = repo.join(".fabrik/dolt");
        if dolt.is_dir() {
            let worker_tx = self.worker_tx.clone();
            let mut watcher = RecommendedWatcher::new(
                move |res: Result<Event, notify::Error>| {
                    let changed = matches!(
                        res,
                        Ok(ref event) if matches!(
                            event.kind,
                            notify::EventKind::Create(_)
                                | notify::EventKind::Modify(_)
                                | notify::EventKind::Remove(_)
                        )
                    );
                    if changed && let Err(err) = worker_tx.send(WorkerMsg::Refresh) {
                        eprintln!("[worker] Refresh send failed: {err:?}");
                    }
                },
                Config::default(),
            )
            .ok();
            if let Some(ref mut active) = watcher
                && let Err(err) = active.watch(&dolt, notify::RecursiveMode::Recursive)
            {
                eprintln!("[worker] watch failed for {}: {err:?}", dolt.display());
            }
            self.watcher = watcher;
        }
    }

    pub fn schedule_refresh(&self) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::Refresh) {
            eprintln!("[worker] Refresh send failed: {err:?}");
        }
    }

    pub fn request_detail(&self, id: &str) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::FetchDetail(id.to_owned())) {
            eprintln!("[worker] FetchDetail send failed: {err:?}");
        }
    }

    pub fn post_comment(&self, id: &str, body: &str) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::PostComment {
            id: id.to_owned(),
            body: body.to_owned(),
        }) {
            eprintln!("[worker] PostComment send failed: {err:?}");
        }
    }

    pub fn refresh_memory(&self, repo: Option<&Path>) {
        let msg = WorkerMsg::RefreshMemory(repo.map(Path::to_path_buf));
        if let Err(err) = self.worker_tx.send(msg) {
            eprintln!("[worker] RefreshMemory send failed: {err:?}");
        }
    }

    pub fn mem_search(&self, query: &str, project: Option<&Path>) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::MemSearch {
            query: query.to_owned(),
            project: project.map(Path::to_path_buf),
        }) {
            eprintln!("[worker] MemSearch send failed: {err:?}");
        }
    }

    pub fn init_repo(&self, repo: &Path, apps: &[String]) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::InitRepo {
            path: repo.to_path_buf(),
            apps: apps.to_vec(),
        }) {
            eprintln!("[worker] InitRepo send failed: {err:?}");
        }
    }

    pub fn mem_get(&self, id: &str) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::MemGet { id: id.to_owned() }) {
            eprintln!("[worker] MemGet send failed: {err:?}");
        }
    }

    pub fn lookup_explore(&self, repo: &Path, query: &str) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::LookupExplore {
            repo: repo.to_path_buf(),
            query: query.to_owned(),
        }) {
            eprintln!("[worker] LookupExplore send failed: {err:?}");
        }
    }

    pub fn lookup_overview(&self, repo: &Path, budget: u32) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::LookupOverview {
            repo: repo.to_path_buf(),
            budget,
        }) {
            eprintln!("[worker] LookupOverview send failed: {err:?}");
        }
    }

    pub fn lookup_explain(&self, repo: &Path, id: &str) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::LookupExplain {
            repo: repo.to_path_buf(),
            id: id.to_owned(),
        }) {
            eprintln!("[worker] PreviewExplain send failed: {err:?}");
        }
    }

    pub fn lookup_path(&self, repo: &Path, from: &str, to: &str) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::LookupPath {
            repo: repo.to_path_buf(),
            from: from.to_owned(),
            to: to.to_owned(),
        }) {
            eprintln!("[worker] LookupPath send failed: {err:?}");
        }
    }

    pub fn graph_status(&self, repo: &Path) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::GraphStatus {
            repo: repo.to_path_buf(),
        }) {
            eprintln!("[worker] GraphStatus send failed: {err:?}");
        }
    }

    pub fn graph_build(&self, repo: &Path) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::GraphBuild {
            repo: repo.to_path_buf(),
        }) {
            eprintln!("[worker] GraphBuild send failed: {err:?}");
        }
    }

    pub fn graph_context(&self, repo: &Path, symbol: &str) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::GraphContext {
            repo: repo.to_path_buf(),
            symbol: symbol.to_owned(),
        }) {
            eprintln!("[worker] GraphContext send failed: {err:?}");
        }
    }

    pub fn graph_impact(&self, repo: &Path, symbol: &str, depth: u32) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::GraphImpact {
            repo: repo.to_path_buf(),
            symbol: symbol.to_owned(),
            depth,
        }) {
            eprintln!("[worker] GraphImpact send failed: {err:?}");
        }
    }

    pub fn detect_changes(&self, repo: &Path, base_ref: &str) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::DetectChanges {
            repo: repo.to_path_buf(),
            base_ref: base_ref.to_owned(),
        }) {
            eprintln!("[worker] DetectChanges send failed: {err:?}");
        }
    }

    pub fn explore_issue(&self, repo: &Path, query: &str, issue: &str) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::ExploreIssue {
            repo: repo.to_path_buf(),
            query: query.to_owned(),
            issue: issue.to_owned(),
        }) {
            eprintln!("[worker] ExploreIssue send failed: {err:?}");
        }
    }

    pub fn mem_check(&self) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::MemCheck) {
            eprintln!("[worker] MemCheck send failed: {err:?}");
        }
    }

    pub fn mem_stats(&self) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::MemStats) {
            eprintln!("[worker] MemStats send failed: {err:?}");
        }
    }

    pub fn mem_timeline(&self, id: &str, limit: u32, project: Option<&Path>) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::MemTimeline {
            id: id.to_owned(),
            limit,
            project: project.map(Path::to_path_buf),
        }) {
            eprintln!("[worker] MemTimeline send failed: {err:?}");
        }
    }

    pub fn mem_current_project(&self, repo: Option<&Path>) {
        let msg = WorkerMsg::CurrentProject(repo.map(Path::to_path_buf));
        if let Err(err) = self.worker_tx.send(msg) {
            eprintln!("[worker] CurrentProject send failed: {err:?}");
        }
    }

    pub fn create_issue(&self, repo: &Path, params: &crate::fabrik::IssueCreateParams) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::IssueCreate {
            repo: repo.to_path_buf(),
            params: params.clone(),
        }) {
            eprintln!("[worker] IssueCreate send failed: {err:?}");
        }
    }

    pub fn claim_issue(&self, repo: &Path, id: &str, by: &str) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::IssueClaim {
            repo: repo.to_path_buf(),
            id: id.to_owned(),
            by: by.to_owned(),
        }) {
            eprintln!("[worker] IssueClaim send failed: {err:?}");
        }
    }

    pub fn unclaim_issue(&self, repo: &Path, id: &str, by: &str, force: bool) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::IssueUnclaim {
            repo: repo.to_path_buf(),
            id: id.to_owned(),
            by: by.to_owned(),
            force,
        }) {
            eprintln!("[worker] IssueUnclaim send failed: {err:?}");
        }
    }

    pub fn close_issue(&self, repo: &Path, id: &str, reason: &str, by: &str, force: bool) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::IssueClose {
            repo: repo.to_path_buf(),
            id: id.to_owned(),
            reason: reason.to_owned(),
            by: by.to_owned(),
            force,
        }) {
            eprintln!("[worker] IssueClose send failed: {err:?}");
        }
    }

    pub fn add_dep(&self, repo: &Path, blocked: &str, blocker: &str, kind: &str) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::IssueDepAdd {
            repo: repo.to_path_buf(),
            blocked: blocked.to_owned(),
            blocker: blocker.to_owned(),
            kind: kind.to_owned(),
        }) {
            eprintln!("[worker] IssueDepAdd send failed: {err:?}");
        }
    }

    pub fn mem_save(&self, repo: Option<&Path>, params: &crate::fabrik::MemSaveParams) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::MemSave {
            repo: repo.map(Path::to_path_buf),
            params: params.clone(),
        }) {
            eprintln!("[worker] MemSave send failed: {err:?}");
        }
    }

    pub fn mem_update(&self, params: &crate::fabrik::MemUpdateParams) {
        if let Err(err) = self.worker_tx.send(WorkerMsg::MemUpdate {
            params: params.clone(),
        }) {
            eprintln!("[worker] MemUpdate send failed: {err:?}");
        }
    }

    pub fn mem_delete(&self, id: &str) {
        if let Err(err) = self
            .worker_tx
            .send(WorkerMsg::MemDelete { id: id.to_owned() })
        {
            eprintln!("[worker] MemDelete send failed: {err:?}");
        }
    }

    pub fn mem_save_prompt(&self, repo: Option<&Path>, title: &str, capture: &str) {
        let msg = WorkerMsg::MemSavePrompt(
            repo.map(Path::to_path_buf),
            title.to_owned(),
            capture.to_owned(),
        );
        if let Err(err) = self.worker_tx.send(msg) {
            eprintln!("[worker] MemSavePrompt send failed: {err:?}");
        }
    }

    pub fn clear(&mut self) {
        self.watched_repo = None;
        self.watcher = None;
        // Tell the worker to drop its repo too; otherwise the next queued
        // Refresh would waste a fetch for a repo with no tabs open.
        if let Err(err) = self.worker_tx.send(WorkerMsg::Clear) {
            eprintln!("[worker] Clear send failed: {err:?}");
        }
    }

    pub fn drain_actions(&self) -> Vec<Action> {
        let mut actions = Vec::new();
        while let Ok(action) = self.action_rx.try_recv() {
            actions.push(action);
        }
        actions
    }

    pub fn poll(&self, ctx: &eframe::egui::Context) -> Vec<Action> {
        let actions = self.drain_actions();
        if !actions.is_empty() {
            ctx.request_repaint();
        }
        actions
    }
}

impl Default for WatchService {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use std::sync::mpsc::channel;

    use super::msg::{WorkerMsg, drain_coalesced};

    #[test]
    fn rapid_switch_keeps_latest_setrepo() {
        let (tx, rx) = channel::<WorkerMsg>();
        tx.send(WorkerMsg::SetRepo("a".into())).unwrap();
        tx.send(WorkerMsg::SetRepo("b".into())).unwrap();
        let pending = drain_coalesced(&rx);
        assert_eq!(pending.latest_repo, Some("b".into()));
    }

    #[test]
    fn refresh_burst_does_not_drop_setrepo() {
        let (tx, rx) = channel::<WorkerMsg>();
        tx.send(WorkerMsg::Refresh).unwrap();
        tx.send(WorkerMsg::Refresh).unwrap();
        tx.send(WorkerMsg::SetRepo("b".into())).unwrap();
        tx.send(WorkerMsg::Refresh).unwrap();
        let pending = drain_coalesced(&rx);
        // The queued SetRepo must survive the Refresh burst.
        assert_eq!(pending.latest_repo, Some("b".into()));
        assert!(pending.refresh_seen);
    }

    #[test]
    fn refresh_burst_without_setrepo_coalesces() {
        let (tx, rx) = channel::<WorkerMsg>();
        tx.send(WorkerMsg::Refresh).unwrap();
        tx.send(WorkerMsg::Refresh).unwrap();
        tx.send(WorkerMsg::FetchDetail("f-a1b2c3d4".to_owned()))
            .unwrap();
        let pending = drain_coalesced(&rx);
        assert_eq!(pending.latest_repo, None);
        assert!(pending.refresh_seen);
        assert_eq!(pending.details, vec!["f-a1b2c3d4".to_owned()]);
    }

    #[test]
    fn stale_details_before_setrepo_are_dropped() {
        let (tx, rx) = channel::<WorkerMsg>();
        tx.send(WorkerMsg::FetchDetail("old".to_owned())).unwrap();
        tx.send(WorkerMsg::SetRepo("b".into())).unwrap();
        tx.send(WorkerMsg::FetchDetail("new".to_owned())).unwrap();
        let pending = drain_coalesced(&rx);
        assert_eq!(pending.latest_repo, Some("b".into()));
        assert_eq!(pending.details, vec!["new".to_owned()]);
    }

    #[test]
    fn clear_drops_queued_refresh() {
        let (tx, rx) = channel::<WorkerMsg>();
        tx.send(WorkerMsg::Refresh).unwrap();
        tx.send(WorkerMsg::FetchDetail("old".to_owned())).unwrap();
        tx.send(WorkerMsg::Clear).unwrap();
        let pending = drain_coalesced(&rx);
        // Everything queued before the Clear is dropped: no wasted fetch.
        assert!(pending.clear_seen);
        assert!(!pending.refresh_seen);
        assert_eq!(pending.latest_repo, None);
        assert_eq!(pending.details, Vec::<String>::new());
        assert!(pending.comments.is_empty());
    }

    #[test]
    fn memory_messages_survive_refresh_burst() {
        let (tx, rx) = channel::<WorkerMsg>();
        tx.send(WorkerMsg::Refresh).unwrap();
        tx.send(WorkerMsg::RefreshMemory(None)).unwrap();
        tx.send(WorkerMsg::MemCheck).unwrap();
        tx.send(WorkerMsg::MemGet {
            id: "o-abc123".to_owned(),
        })
        .unwrap();
        let pending = drain_coalesced(&rx);
        assert!(pending.refresh_seen);
        assert_eq!(pending.deferred.len(), 3);
        assert!(matches!(pending.deferred[0], WorkerMsg::RefreshMemory(_)));
        assert!(matches!(pending.deferred[1], WorkerMsg::MemCheck));
        assert!(matches!(pending.deferred[2], WorkerMsg::MemGet { .. }));
    }

    #[test]
    fn clear_keeps_memory_queued_after_it() {
        let (tx, rx) = channel::<WorkerMsg>();
        tx.send(WorkerMsg::Refresh).unwrap();
        tx.send(WorkerMsg::Clear).unwrap();
        tx.send(WorkerMsg::RefreshMemory(None)).unwrap();
        let pending = drain_coalesced(&rx);
        // Everything board-scoped queued before the Clear is dropped…
        assert!(pending.clear_seen);
        assert!(!pending.refresh_seen);
        // …but the memory refresh queued after it survives in `deferred`.
        assert_eq!(pending.deferred.len(), 1);
        assert!(matches!(pending.deferred[0], WorkerMsg::RefreshMemory(_)));
    }

    #[test]
    fn stale_comments_before_setrepo_are_dropped() {
        let (tx, rx) = channel::<WorkerMsg>();
        tx.send(WorkerMsg::PostComment {
            id: "old".to_owned(),
            body: "stale".to_owned(),
        })
        .unwrap();
        tx.send(WorkerMsg::SetRepo("b".into())).unwrap();
        tx.send(WorkerMsg::PostComment {
            id: "new".to_owned(),
            body: "fresh".to_owned(),
        })
        .unwrap();
        let pending = drain_coalesced(&rx);
        assert_eq!(pending.latest_repo, Some("b".into()));
        assert_eq!(pending.details, Vec::<String>::new());
        assert_eq!(pending.comments.len(), 1);
        assert_eq!(pending.comments[0].id, "new");
        assert_eq!(pending.comments[0].body, "fresh");
    }
}
