use std::path::PathBuf;
use std::sync::mpsc::Receiver;

pub enum WorkerMsg {
    SetRepo(PathBuf),
    Clear,
    Refresh,
    FetchDetail(String),
    PostComment { id: String, body: String },
    RefreshMemory,
    MemSearch { query: String, project: Option<PathBuf> },
    MemGet { id: String },
    InitRepo { path: PathBuf, apps: Vec<String> },
    LookupExplore { repo: PathBuf, query: String },
    LookupExplain { repo: PathBuf, id: String },
    LookupPath { repo: PathBuf, from: String, to: String },
    GraphStatus { repo: PathBuf },
    GraphBuild { repo: PathBuf },
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
}

/// Drain the queue without loss: `Refresh` messages coalesce, only the latest
/// `SetRepo` is kept, and `FetchDetail`/`PostComment` queued before a
/// `SetRepo` are stale and dropped. `Clear` acts as a barrier: everything
/// queued before it (pending refresh/details/comments and any earlier
/// `SetRepo`) is dropped; only messages queued after it survive. Never
/// blocks; returns what was pending at call time.
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
            WorkerMsg::RefreshMemory
            | WorkerMsg::MemSearch { .. }
            | WorkerMsg::MemGet { .. }
            | WorkerMsg::InitRepo { .. }
            | WorkerMsg::LookupExplore { .. }
            | WorkerMsg::LookupExplain { .. }
            | WorkerMsg::LookupPath { .. }
            | WorkerMsg::GraphStatus { .. }
            | WorkerMsg::GraphBuild { .. } => {}
        }
    }
    out
}
