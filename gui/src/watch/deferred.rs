use std::sync::mpsc::Sender;

use crate::model::Action;

use super::dispatch::{
    dispatch_current_project, dispatch_detect_changes, dispatch_explore_issue,
    dispatch_graph_build, dispatch_graph_context, dispatch_graph_impact, dispatch_graph_status,
    dispatch_init_repo, dispatch_lookup_explain, dispatch_lookup_explore, dispatch_lookup_path,
    dispatch_mem_check, dispatch_mem_context, dispatch_mem_get, dispatch_mem_search,
    dispatch_mem_stats, dispatch_mem_timeline,
};
use super::msg::WorkerMsg;
use super::overview::dispatch_lookup_overview;
use super::writes::{
    dispatch_issue_claim, dispatch_issue_close, dispatch_issue_create, dispatch_issue_dep_add,
    dispatch_issue_unclaim, dispatch_mem_delete, dispatch_mem_save, dispatch_mem_save_prompt,
    dispatch_mem_update,
};

/// Dispatch messages that were queued behind a board control message and
/// preserved by `drain_coalesced`. Runs after the board work so FIFO order
/// is roughly kept. Without this, a `RefreshMemory` dropped here after
/// `mem_loading` was set leaves the Memory pane spinning forever (same for
/// `MemSearch`, `MemGet`, and write messages stuck on busy).
pub(super) fn dispatch_deferred(action_tx: &Sender<Action>, deferred: &[WorkerMsg]) {
    for msg in deferred {
        match msg {
            // Board control messages never land here: `drain_coalesced`
            // routes them into `Coalesced`, not `deferred`.
            WorkerMsg::SetRepo(_)
            | WorkerMsg::Clear
            | WorkerMsg::Refresh
            | WorkerMsg::FetchDetail(_)
            | WorkerMsg::PostComment { .. } => {}
            WorkerMsg::RefreshMemory(repo) => dispatch_mem_context(action_tx, repo.as_deref()),
            WorkerMsg::MemSearch { query, project } => {
                dispatch_mem_search(action_tx, query, project.as_deref());
            }
            WorkerMsg::MemGet { id } => dispatch_mem_get(action_tx, id),
            WorkerMsg::InitRepo { path, apps } => dispatch_init_repo(action_tx, path, apps),
            WorkerMsg::LookupExplore { repo, query } => {
                dispatch_lookup_explore(action_tx, repo, query);
            }
            WorkerMsg::LookupExplain { repo, id } => {
                dispatch_lookup_explain(action_tx, repo, id);
            }
            WorkerMsg::LookupOverview { repo, budget } => {
                dispatch_lookup_overview(action_tx, repo, *budget);
            }
            WorkerMsg::LookupPath { repo, from, to } => {
                dispatch_lookup_path(action_tx, repo, from, to);
            }
            WorkerMsg::GraphStatus { repo } => dispatch_graph_status(action_tx, repo),
            WorkerMsg::GraphBuild { repo } => dispatch_graph_build(action_tx, repo),
            WorkerMsg::GraphContext { repo, symbol } => {
                dispatch_graph_context(action_tx, repo, symbol);
            }
            WorkerMsg::GraphImpact {
                repo,
                symbol,
                depth,
            } => dispatch_graph_impact(action_tx, repo, symbol, *depth),
            WorkerMsg::DetectChanges { repo, base_ref } => {
                dispatch_detect_changes(action_tx, repo, base_ref);
            }
            WorkerMsg::ExploreIssue { repo, query, issue } => {
                dispatch_explore_issue(action_tx, repo, query, issue);
            }
            WorkerMsg::MemCheck => dispatch_mem_check(action_tx),
            WorkerMsg::MemStats => dispatch_mem_stats(action_tx),
            WorkerMsg::MemTimeline { id, limit, project } => {
                dispatch_mem_timeline(action_tx, id, *limit, project.as_deref());
            }
            WorkerMsg::CurrentProject(repo) => dispatch_current_project(action_tx, repo.as_deref()),
            WorkerMsg::IssueCreate { repo, params } => {
                dispatch_issue_create(action_tx, repo, params);
            }
            WorkerMsg::IssueClaim { repo, id, by } => {
                dispatch_issue_claim(action_tx, repo, id, by);
            }
            WorkerMsg::IssueUnclaim {
                repo,
                id,
                by,
                force,
            } => dispatch_issue_unclaim(action_tx, repo, id, by, *force),
            WorkerMsg::IssueClose {
                repo,
                id,
                reason,
                by,
                force,
            } => dispatch_issue_close(action_tx, repo, id, reason, by, *force),
            WorkerMsg::IssueDepAdd {
                repo,
                blocked,
                blocker,
                kind,
            } => dispatch_issue_dep_add(action_tx, repo, blocked, blocker, kind),
            WorkerMsg::MemSave { repo, params } => {
                dispatch_mem_save(action_tx, repo.as_deref(), params);
            }
            WorkerMsg::MemUpdate { params } => dispatch_mem_update(action_tx, params),
            WorkerMsg::MemDelete { id } => dispatch_mem_delete(action_tx, id),
            WorkerMsg::MemSavePrompt(repo, title, capture) => {
                dispatch_mem_save_prompt(action_tx, repo.as_deref(), title, capture);
            }
        }
    }
}
