use std::path::{Path, PathBuf};
use std::sync::mpsc::{Receiver, Sender};

use crate::fabrik::{
    fetch_detail, fetch_graph_explore, fetch_graph_explain, fetch_graph_path, fetch_graph_query,
    fetch_graph_status, fetch_mem_context, fetch_mem_get, fetch_mem_search, nodes_from_explore,
    fetch_snapshot, post_note, run_graph_build, run_project_init,
};
use crate::model::Action;

use super::msg::{Coalesced, WorkerMsg, drain_coalesced};

fn dispatch_pending(action_tx: &Sender<Action>, repo: &Path, pending: &Coalesced) {
    dispatch_refresh(action_tx, repo);
    for id in &pending.details {
        dispatch_detail(action_tx, repo, id);
    }
    for queued in &pending.comments {
        dispatch_post(action_tx, repo, &queued.id, &queued.body);
    }
}

/// Serve details/comments queued behind an already-handled message, then
/// refresh only if a `Refresh` arrived mid-flight (same coalescing the loop
/// always applied; factored out so `worker_loop` stays compact).
fn dispatch_queued(action_tx: &Sender<Action>, repo: &Path, pending: &Coalesced) {
    for id in &pending.details {
        dispatch_detail(action_tx, repo, id);
    }
    for queued in &pending.comments {
        dispatch_post(action_tx, repo, &queued.id, &queued.body);
    }
    if pending.refresh_seen {
        dispatch_refresh(action_tx, repo);
    }
}

pub(crate) fn worker_loop(rx: &Receiver<WorkerMsg>, action_tx: &Sender<Action>) {
    let mut current_repo: Option<PathBuf> = None;
    while let Ok(msg) = rx.recv() {
        match msg {
            WorkerMsg::SetRepo(repo) => {
                current_repo = Some(repo.clone());
                // Coalesce everything queued while this message waited:
                // Refresh bursts collapse into the SetRepo fetch, the latest
                // SetRepo wins, stale FetchDetails for older repos are dropped.
                let pending = drain_coalesced(rx);
                if pending.clear_seen && pending.latest_repo.is_none() {
                    // A Clear arrived after this SetRepo: the repo is gone,
                    // so drop the fetch instead of wasting it.
                    current_repo = None;
                } else if let Some(ref last) = pending.latest_repo {
                    current_repo = Some(last.clone());
                    dispatch_pending(action_tx, last, &pending);
                } else {
                    dispatch_pending(action_tx, &repo, &pending);
                }
            }
            WorkerMsg::Clear => {
                // Last-tab-close: forget the repo so queued Refresh bursts
                // are dropped instead of fetched.
                let pending = drain_coalesced(rx);
                if let Some(ref last) = pending.latest_repo {
                    // A SetRepo queued after the Clear still wins.
                    current_repo = Some(last.clone());
                    dispatch_pending(action_tx, last, &pending);
                } else {
                    current_repo = None;
                }
            }
            WorkerMsg::Refresh => {
                // Coalesce Refresh bursts but never drop SetRepo/FetchDetail.
                let pending = drain_coalesced(rx);
                if pending.clear_seen {
                    // Cleared while this Refresh waited: only a SetRepo
                    // queued after the Clear justifies a fetch.
                    current_repo.clone_from(&pending.latest_repo);
                    if let Some(ref last) = current_repo {
                        dispatch_pending(action_tx, last, &pending);
                    }
                } else if let Some(ref last) = pending.latest_repo {
                    current_repo = Some(last.clone());
                    dispatch_pending(action_tx, last, &pending);
                } else if let Some(ref repo) = current_repo.clone() {
                    dispatch_pending(action_tx, repo, &pending);
                }
            }
            WorkerMsg::FetchDetail(id) => {
                let pending = drain_coalesced(rx);
                if pending.clear_seen && pending.latest_repo.is_none() {
                    // Cleared while this request waited: drop it.
                    current_repo = None;
                } else if let Some(ref last) = pending.latest_repo {
                    // The requested id belongs to the previous repo; drop it
                    // and serve only details queued after the latest SetRepo.
                    current_repo = Some(last.clone());
                    dispatch_pending(action_tx, last, &pending);
                } else if let Some(ref repo) = current_repo.clone() {
                    dispatch_detail(action_tx, repo, &id);
                    dispatch_queued(action_tx, repo, &pending);
                }
            }
            WorkerMsg::PostComment { id, body } => {
                let pending = drain_coalesced(rx);
                if pending.clear_seen && pending.latest_repo.is_none() {
                    current_repo = None;
                } else if let Some(ref last) = pending.latest_repo {
                    current_repo = Some(last.clone());
                    dispatch_pending(action_tx, last, &pending);
                } else if let Some(ref repo) = current_repo.clone() {
                    dispatch_post(action_tx, repo, &id, &body);
                    dispatch_queued(action_tx, repo, &pending);
                }
            }
            WorkerMsg::RefreshMemory => dispatch_mem_context(action_tx),
            WorkerMsg::MemSearch { query, project } => {
                dispatch_mem_search(action_tx, &query, project.as_deref());
            }
            WorkerMsg::MemGet { id } => dispatch_mem_get(action_tx, &id),
            WorkerMsg::InitRepo { path, apps } => dispatch_init_repo(action_tx, &path, &apps),
            WorkerMsg::LookupExplore { repo, query } => {
                dispatch_lookup_explore(action_tx, &repo, &query);
            }
            WorkerMsg::LookupExplain { repo, id } => {
                dispatch_lookup_explain(action_tx, &repo, &id);
            }
            WorkerMsg::LookupPath { repo, from, to } => {
                dispatch_lookup_path(action_tx, &repo, &from, &to);
            }
            WorkerMsg::GraphStatus { repo } => dispatch_graph_status(action_tx, &repo),
            WorkerMsg::GraphBuild { repo } => dispatch_graph_build(action_tx, &repo),
        }
    }
}

fn dispatch_refresh(action_tx: &Sender<Action>, path: &Path) {
    match fetch_snapshot(path) {
        Ok(snapshot) => {
            if let Err(err) = action_tx.send(Action::Snapshot(snapshot)) {
                eprintln!("[worker] Snapshot send failed: {err:?}");
            }
        }
        Err(err) => {
            if let Err(send_err) =
                action_tx.send(Action::Failed(Some(path.to_path_buf()), err.to_string()))
            {
                eprintln!("[worker] Failed send failed: {send_err:?}");
            }
        }
    }
}

fn dispatch_detail(action_tx: &Sender<Action>, path: &Path, id: &str) {
    match fetch_detail(path, id) {
        Ok(detail) => {
            if let Err(err) =
                action_tx.send(Action::DetailLoaded(path.to_path_buf(), Box::new(detail)))
            {
                eprintln!("[worker] DetailLoaded send failed: {err:?}");
            }
        }
        Err(err) => {
            if let Err(send_err) =
                action_tx.send(Action::Failed(Some(path.to_path_buf()), err.to_string()))
            {
                eprintln!("[worker] Failed send failed: {send_err:?}");
            }
        }
    }
}

fn dispatch_mem_context(action_tx: &Sender<Action>) {
    match fetch_mem_context() {
        Ok(ctx) => {
            let _ = action_tx.send(Action::MemContextLoaded(ctx));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
        }
    }
}

fn dispatch_mem_search(action_tx: &Sender<Action>, query: &str, project: Option<&Path>) {
    match fetch_mem_search(query, project) {
        Ok(rows) => {
            let _ = action_tx.send(Action::MemSearchLoaded(rows));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
        }
    }
}

fn dispatch_init_repo(action_tx: &Sender<Action>, path: &Path, apps: &[String]) {
    match run_project_init(path, apps) {
        Ok(()) => {
            let _ = action_tx.send(Action::ProjectInitDone(path.to_path_buf()));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(path.to_path_buf()), err.to_string()));
        }
    }
}

fn dispatch_mem_get(action_tx: &Sender<Action>, id: &str) {
    match fetch_mem_get(id) {
        Ok(obs) => {
            let _ = action_tx.send(Action::MemGetLoaded(obs));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
        }
    }
}

fn dispatch_lookup_explore(action_tx: &Sender<Action>, repo: &Path, query: &str) {
    match fetch_graph_explore(repo, query) {
        Ok(result) => {
            if !result.indexed {
                let _ = action_tx.send(Action::LookupExploreLoaded(result));
                return;
            }
            let nodes = nodes_from_explore(&result);
            if !nodes.is_empty() {
                let _ = action_tx.send(Action::LookupExploreLoaded(result));
                return;
            }
            match fetch_graph_query(repo, query) {
                Ok(q) if !q.nodes.is_empty() => {
                    let _ = action_tx.send(Action::LookupQueryLoaded(q));
                }
                _ => {
                    let _ = action_tx.send(Action::LookupExploreLoaded(result));
                }
            }
        }
        Err(_) => match fetch_graph_query(repo, query) {
            Ok(q) => {
                let _ = action_tx.send(Action::LookupQueryLoaded(q));
            }
            Err(err) => {
                let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
            }
        },
    }
}

fn dispatch_lookup_explain(action_tx: &Sender<Action>, repo: &Path, id: &str) {
    match fetch_graph_explain(repo, id) {
        Ok(result) => {
            let _ = action_tx.send(Action::LookupExplainLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

fn dispatch_graph_status(action_tx: &Sender<Action>, repo: &Path) {
    match fetch_graph_status(repo) {
        Ok(status) => {
            let _ = action_tx.send(Action::GraphStatusLoaded(status));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

fn dispatch_graph_build(action_tx: &Sender<Action>, repo: &Path) {
    match run_graph_build(repo) {
        Ok(_) => {
            let _ = action_tx.send(Action::GraphBuildDone(repo.to_path_buf()));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

fn dispatch_lookup_path(action_tx: &Sender<Action>, repo: &Path, from: &str, to: &str) {
    match fetch_graph_path(repo, from, to) {
        Ok(result) => {
            let _ = action_tx.send(Action::LookupPathLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

fn dispatch_post(action_tx: &Sender<Action>, path: &Path, id: &str, body: &str) {
    match post_note(path, id, body) {
        Ok(detail) => {
            if let Err(err) =
                action_tx.send(Action::DetailLoaded(path.to_path_buf(), Box::new(detail)))
            {
                eprintln!("[worker] DetailLoaded send failed: {err:?}");
            }
        }
        Err(err) => {
            if let Err(send_err) =
                action_tx.send(Action::Failed(Some(path.to_path_buf()), err.to_string()))
            {
                eprintln!("[worker] Failed send failed: {send_err:?}");
            }
        }
    }
}
