use std::path::{Path, PathBuf};
use std::sync::mpsc::{Receiver, Sender};

use crate::fabrik::{
    fetch_detail, fetch_graph_context, fetch_graph_detect_changes, fetch_graph_explain,
    fetch_graph_explore, fetch_graph_explore_issue, fetch_graph_impact, fetch_graph_path,
    fetch_graph_query, fetch_graph_status, fetch_mem_check, fetch_mem_context,
    fetch_mem_current_project, fetch_mem_get, fetch_mem_search, fetch_mem_stats,
    fetch_mem_timeline, fetch_snapshot, nodes_from_explore, post_note, run_graph_build,
    run_project_init,
};
use crate::model::Action;

use super::deferred::dispatch_deferred;
use super::msg::{Coalesced, WorkerMsg, drain_coalesced};
use super::overview::dispatch_lookup_overview;
use super::writes::{
    dispatch_issue_claim, dispatch_issue_close, dispatch_issue_create, dispatch_issue_dep_add,
    dispatch_issue_unclaim, dispatch_mem_delete, dispatch_mem_save, dispatch_mem_save_prompt,
    dispatch_mem_update,
};

fn dispatch_pending(action_tx: &Sender<Action>, repo: &Path, pending: &Coalesced) {
    dispatch_refresh(action_tx, repo);
    for id in &pending.details {
        dispatch_detail(action_tx, repo, id);
    }
    for queued in &pending.comments {
        dispatch_post(action_tx, repo, &queued.id, &queued.body);
    }
    dispatch_deferred(action_tx, &pending.deferred);
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
    dispatch_deferred(action_tx, &pending.deferred);
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
                    // so drop the fetch instead of wasting it. Deferred
                    // memory/lookup/write messages are repo-independent, so
                    // they still run.
                    current_repo = None;
                    dispatch_deferred(action_tx, &pending.deferred);
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
                    // Deferred messages carry their own repo (or none for
                    // memory), so they still run instead of sticking their
                    // busy flags forever.
                    dispatch_deferred(action_tx, &pending.deferred);
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
                    } else {
                        dispatch_deferred(action_tx, &pending.deferred);
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
                    dispatch_deferred(action_tx, &pending.deferred);
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
                    dispatch_deferred(action_tx, &pending.deferred);
                } else if let Some(ref last) = pending.latest_repo {
                    current_repo = Some(last.clone());
                    dispatch_pending(action_tx, last, &pending);
                } else if let Some(ref repo) = current_repo.clone() {
                    dispatch_post(action_tx, repo, &id, &body);
                    dispatch_queued(action_tx, repo, &pending);
                }
            }
            WorkerMsg::RefreshMemory(repo) => dispatch_mem_context(action_tx, repo.as_deref()),
            WorkerMsg::MemSearch { query, project } => {
                dispatch_mem_search(action_tx, &query, project.as_deref());
            }
            WorkerMsg::MemGet { id } => dispatch_mem_get(action_tx, &id),
            WorkerMsg::InitRepo { path, apps } => dispatch_init_repo(action_tx, &path, &apps),
            WorkerMsg::LookupExplore { repo, query } => {
                dispatch_lookup_explore(action_tx, &repo, &query);
            }
            WorkerMsg::LookupOverview { repo, budget } => {
                dispatch_lookup_overview(action_tx, &repo, budget);
            }
            WorkerMsg::LookupExplain { repo, id } => {
                dispatch_lookup_explain(action_tx, &repo, &id);
            }
            WorkerMsg::LookupPath { repo, from, to } => {
                dispatch_lookup_path(action_tx, &repo, &from, &to);
            }
            WorkerMsg::GraphStatus { repo } => dispatch_graph_status(action_tx, &repo),
            WorkerMsg::GraphBuild { repo } => dispatch_graph_build(action_tx, &repo),
            WorkerMsg::GraphContext { repo, symbol } => {
                dispatch_graph_context(action_tx, &repo, &symbol);
            }
            WorkerMsg::GraphImpact {
                repo,
                symbol,
                depth,
            } => {
                dispatch_graph_impact(action_tx, &repo, &symbol, depth);
            }
            WorkerMsg::DetectChanges { repo, base_ref } => {
                dispatch_detect_changes(action_tx, &repo, &base_ref);
            }
            WorkerMsg::ExploreIssue { repo, query, issue } => {
                dispatch_explore_issue(action_tx, &repo, &query, &issue);
            }
            WorkerMsg::MemCheck => dispatch_mem_check(action_tx),
            WorkerMsg::MemStats => dispatch_mem_stats(action_tx),
            WorkerMsg::MemTimeline { id, limit, project } => {
                dispatch_mem_timeline(action_tx, &id, limit, project.as_deref());
            }
            WorkerMsg::CurrentProject(repo) => dispatch_current_project(action_tx, repo.as_deref()),
            WorkerMsg::IssueCreate { repo, params } => {
                dispatch_issue_create(action_tx, &repo, &params);
            }
            WorkerMsg::IssueClaim { repo, id, by } => {
                dispatch_issue_claim(action_tx, &repo, &id, &by);
            }
            WorkerMsg::IssueUnclaim {
                repo,
                id,
                by,
                force,
            } => {
                dispatch_issue_unclaim(action_tx, &repo, &id, &by, force);
            }
            WorkerMsg::IssueClose {
                repo,
                id,
                reason,
                by,
                force,
            } => {
                dispatch_issue_close(action_tx, &repo, &id, &reason, &by, force);
            }
            WorkerMsg::IssueDepAdd {
                repo,
                blocked,
                blocker,
                kind,
            } => {
                dispatch_issue_dep_add(action_tx, &repo, &blocked, &blocker, &kind);
            }
            WorkerMsg::MemSave { repo, params } => {
                dispatch_mem_save(action_tx, repo.as_deref(), &params);
            }
            WorkerMsg::MemUpdate { params } => dispatch_mem_update(action_tx, &params),
            WorkerMsg::MemDelete { id } => dispatch_mem_delete(action_tx, &id),
            WorkerMsg::MemSavePrompt(repo, title, capture) => {
                dispatch_mem_save_prompt(action_tx, repo.as_deref(), &title, &capture);
            }
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

pub(super) fn dispatch_mem_context(action_tx: &Sender<Action>, repo: Option<&Path>) {
    match fetch_mem_context(repo) {
        Ok(ctx) => {
            let _ = action_tx.send(Action::MemContextLoaded(ctx));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
        }
    }
}

pub(super) fn dispatch_mem_search(action_tx: &Sender<Action>, query: &str, project: Option<&Path>) {
    match fetch_mem_search(query, project) {
        Ok(rows) => {
            let _ = action_tx.send(Action::MemSearchLoaded(rows));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
        }
    }
}

pub(super) fn dispatch_init_repo(action_tx: &Sender<Action>, path: &Path, apps: &[String]) {
    match run_project_init(path, apps) {
        Ok(()) => {
            let _ = action_tx.send(Action::ProjectInitDone(path.to_path_buf()));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(path.to_path_buf()), err.to_string()));
        }
    }
}

pub(super) fn dispatch_mem_get(action_tx: &Sender<Action>, id: &str) {
    match fetch_mem_get(id) {
        Ok(obs) => {
            let _ = action_tx.send(Action::MemGetLoaded(obs));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
        }
    }
}

pub(super) fn dispatch_lookup_explore(action_tx: &Sender<Action>, repo: &Path, query: &str) {
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

pub(super) fn dispatch_lookup_explain(action_tx: &Sender<Action>, repo: &Path, id: &str) {
    match fetch_graph_explain(repo, id) {
        Ok(result) => {
            let _ = action_tx.send(Action::LookupExplainLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

pub(super) fn dispatch_graph_status(action_tx: &Sender<Action>, repo: &Path) {
    match fetch_graph_status(repo) {
        Ok(status) => {
            let _ = action_tx.send(Action::GraphStatusLoaded(status));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

pub(super) fn dispatch_graph_build(action_tx: &Sender<Action>, repo: &Path) {
    match run_graph_build(repo) {
        Ok(_) => {
            let _ = action_tx.send(Action::GraphBuildDone(repo.to_path_buf()));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

pub(super) fn dispatch_lookup_path(action_tx: &Sender<Action>, repo: &Path, from: &str, to: &str) {
    match fetch_graph_path(repo, from, to) {
        Ok(result) => {
            let _ = action_tx.send(Action::LookupPathLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

pub(super) fn dispatch_graph_context(action_tx: &Sender<Action>, repo: &Path, symbol: &str) {
    match fetch_graph_context(repo, symbol) {
        Ok(result) => {
            let _ = action_tx.send(Action::GraphContextLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

pub(super) fn dispatch_graph_impact(
    action_tx: &Sender<Action>,
    repo: &Path,
    symbol: &str,
    depth: u32,
) {
    match fetch_graph_impact(repo, symbol, depth) {
        Ok(result) => {
            let _ = action_tx.send(Action::GraphImpactLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

pub(super) fn dispatch_detect_changes(action_tx: &Sender<Action>, repo: &Path, base_ref: &str) {
    match fetch_graph_detect_changes(repo, base_ref) {
        Ok(result) => {
            let _ = action_tx.send(Action::DetectChangesLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

pub(super) fn dispatch_explore_issue(
    action_tx: &Sender<Action>,
    repo: &Path,
    query: &str,
    issue: &str,
) {
    match fetch_graph_explore_issue(repo, query, issue) {
        Ok(result) => {
            let _ = action_tx.send(Action::LookupExploreLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

pub(super) fn dispatch_mem_check(action_tx: &Sender<Action>) {
    match fetch_mem_check() {
        Ok(result) => {
            let _ = action_tx.send(Action::MemCheckLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
        }
    }
}

pub(super) fn dispatch_mem_stats(action_tx: &Sender<Action>) {
    match fetch_mem_stats() {
        Ok(result) => {
            let _ = action_tx.send(Action::MemStatsLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
        }
    }
}

pub(super) fn dispatch_mem_timeline(
    action_tx: &Sender<Action>,
    id: &str,
    limit: u32,
    project: Option<&Path>,
) {
    match fetch_mem_timeline(id, limit, project) {
        Ok(result) => {
            let _ = action_tx.send(Action::MemTimelineLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
        }
    }
}

pub(super) fn dispatch_current_project(action_tx: &Sender<Action>, repo: Option<&Path>) {
    match fetch_mem_current_project(repo) {
        Ok(result) => {
            let _ = action_tx.send(Action::CurrentProjectLoaded(result));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
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
