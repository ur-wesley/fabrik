use std::path::PathBuf;
use std::sync::mpsc::Receiver;

#[derive(Debug)]
pub enum WorkerMsg {
    SetRepo(PathBuf),
    Clear,
    Refresh,
    FetchDetail(String),
    PostComment {
        id: String,
        body: String,
    },
    RefreshMemory(Option<PathBuf>),
    MemSearch {
        query: String,
        project: Option<PathBuf>,
    },
    MemGet {
        id: String,
    },
    InitRepo {
        path: PathBuf,
        apps: Vec<String>,
    },
    LookupExplore {
        repo: PathBuf,
        query: String,
    },
    LookupOverview {
        repo: PathBuf,
        budget: u32,
    },
    LookupExplain {
        repo: PathBuf,
        id: String,
    },
    LookupPath {
        repo: PathBuf,
        from: String,
        to: String,
    },
    GraphStatus {
        repo: PathBuf,
    },
    GraphBuild {
        repo: PathBuf,
    },
    GraphContext {
        repo: PathBuf,
        symbol: String,
    },
    GraphImpact {
        repo: PathBuf,
        symbol: String,
        depth: u32,
    },
    DetectChanges {
        repo: PathBuf,
        base_ref: String,
    },
    ExploreIssue {
        repo: PathBuf,
        query: String,
        issue: String,
    },
    MemCheck,
    MemStats,
    MemTimeline {
        id: String,
        limit: u32,
        project: Option<PathBuf>,
    },
    CurrentProject(Option<PathBuf>),
    IssueCreate {
        repo: PathBuf,
        params: crate::fabrik::IssueCreateParams,
    },
    IssueClaim {
        repo: PathBuf,
        id: String,
        by: String,
    },
    IssueUnclaim {
        repo: PathBuf,
        id: String,
        by: String,
        force: bool,
    },
    IssueClose {
        repo: PathBuf,
        id: String,
        reason: String,
        by: String,
        force: bool,
    },
    IssueDepAdd {
        repo: PathBuf,
        blocked: String,
        blocker: String,
        kind: String,
    },
    MemSave {
        repo: Option<PathBuf>,
        params: crate::fabrik::MemSaveParams,
    },
    MemUpdate {
        params: crate::fabrik::MemUpdateParams,
    },
    MemDelete {
        id: String,
    },
    MemSavePrompt(Option<PathBuf>, String, String),
}
#[derive(Debug)]
pub(crate) struct QueuedComment {
    pub(crate) id: String,
    pub(crate) body: String,
}

#[derive(Debug, Default)]
pub(crate) struct Coalesced {
    pub(crate) latest_repo: Option<PathBuf>,
    pub(crate) refresh_seen: bool,
    pub(crate) clear_seen: bool,
    pub(crate) details: Vec<String>,
    pub(crate) comments: Vec<QueuedComment>,
    /// Non-board messages drained while a board message waited. These must
    /// be dispatched after the board work: dropping a queued
    /// `RefreshMemory` after `mem_loading` was already set leaves the
    /// Memory pane on its spinner forever.
    pub(crate) deferred: Vec<WorkerMsg>,
}

/// Drain the queue without loss: `Refresh` messages coalesce, only the latest
/// `SetRepo` is kept, and `FetchDetail`/`PostComment` queued before a
/// `SetRepo` are stale and dropped. `Clear` acts as a barrier: everything
/// queued before it (pending refresh/details/comments and any earlier
/// `SetRepo`) is dropped; only messages queued after it survive. Board
/// control messages coalesce as described; every other message (memory,
/// lookup, graph, writes, init) is preserved in `deferred` FIFO order so a
/// queued `RefreshMemory`/`MemSearch`/`MemGet`/write is never silently lost.
/// Never blocks; returns what was pending at call time.
pub(crate) fn drain_coalesced(rx: &Receiver<WorkerMsg>) -> Coalesced {
    let mut out = Coalesced::default();
    while let Ok(pending) = rx.try_recv() {
        match pending {
            WorkerMsg::SetRepo(repo) => {
                out.latest_repo = Some(repo);
                out.details.clear();
                out.comments.clear();
                out.refresh_seen = false;
            }
            WorkerMsg::Clear => {
                out.clear_seen = true;
                out.latest_repo = None;
                out.details.clear();
                out.comments.clear();
                out.refresh_seen = false;
            }
            WorkerMsg::Refresh => {
                out.refresh_seen = true;
            }
            WorkerMsg::FetchDetail(id) => {
                out.details.push(id);
            }
            WorkerMsg::PostComment { id, body } => {
                out.comments.push(QueuedComment { id, body });
            }
            // Memory/lookup/graph/write/init messages: never coalesce away,
            // defer them for dispatch after the board work.
            other => out.deferred.push(other),
        }
    }
    out
}
