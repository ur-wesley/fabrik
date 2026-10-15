use std::path::Path;
use std::sync::mpsc::Sender;

use crate::fabrik::{
    FabrikError, IssueCreateParams, MemSaveParams, MemUpdateParams, Observation, add_dep,
    claim_issue, close_issue, create_issue, delete_observation, save_observation, save_prompt,
    unclaim_issue, update_observation,
};
use crate::model::{Action, IssueDetail};

/// Issue writes share one completion path: the `--json` output already
/// carries the full detail, so the popup updates without a second fetch
/// and `WriteOk` drives the toast + board refresh in `Effects`.
fn send_issue_write(
    action_tx: &Sender<Action>,
    repo: &Path,
    result: Result<IssueDetail, FabrikError>,
    message: String,
) {
    match result {
        Ok(detail) => {
            let _ = action_tx.send(Action::DetailLoaded(repo.to_path_buf(), Box::new(detail)));
            let _ = action_tx.send(Action::WriteOk(message));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(Some(repo.to_path_buf()), err.to_string()));
        }
    }
}

pub(super) fn dispatch_issue_create(
    action_tx: &Sender<Action>,
    repo: &Path,
    params: &IssueCreateParams,
) {
    let result = create_issue(repo, params);
    let message = match &result {
        Ok(detail) => format!("Issue {} created", detail.id),
        Err(_) => "Create failed".to_owned(),
    };
    send_issue_write(action_tx, repo, result, message);
}

pub(super) fn dispatch_issue_claim(action_tx: &Sender<Action>, repo: &Path, id: &str, by: &str) {
    let result = claim_issue(repo, id, by);
    send_issue_write(action_tx, repo, result, format!("Issue {id} claimed"));
}

pub(super) fn dispatch_issue_unclaim(
    action_tx: &Sender<Action>,
    repo: &Path,
    id: &str,
    by: &str,
    force: bool,
) {
    let result = unclaim_issue(repo, id, by, force);
    send_issue_write(action_tx, repo, result, format!("Issue {id} unclaimed"));
}

pub(super) fn dispatch_issue_close(
    action_tx: &Sender<Action>,
    repo: &Path,
    id: &str,
    reason: &str,
    by: &str,
    force: bool,
) {
    let result = close_issue(repo, id, reason, by, force);
    send_issue_write(action_tx, repo, result, format!("Issue {id} closed"));
}

pub(super) fn dispatch_issue_dep_add(
    action_tx: &Sender<Action>,
    repo: &Path,
    blocked: &str,
    blocker: &str,
    kind: &str,
) {
    let result = add_dep(repo, blocked, blocker, kind);
    send_issue_write(
        action_tx,
        repo,
        result,
        format!("{blocker} now blocks {blocked}"),
    );
}

/// Memory writes show the saved observation inline (`MemGetLoaded`) and
/// toast + refresh via `MemWriteOk` in `Effects`.
fn send_mem_saved(
    action_tx: &Sender<Action>,
    result: Result<Observation, FabrikError>,
    verb: &str,
) {
    match result {
        Ok(obs) => {
            let message = format!("Observation {} {verb}", obs.id);
            let _ = action_tx.send(Action::MemGetLoaded(obs));
            let _ = action_tx.send(Action::MemWriteOk(message));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
        }
    }
}

pub(super) fn dispatch_mem_save(
    action_tx: &Sender<Action>,
    repo: Option<&Path>,
    params: &MemSaveParams,
) {
    send_mem_saved(action_tx, save_observation(repo, params), "saved");
}

pub(super) fn dispatch_mem_update(action_tx: &Sender<Action>, params: &MemUpdateParams) {
    send_mem_saved(action_tx, update_observation(params), "updated");
}

pub(super) fn dispatch_mem_delete(action_tx: &Sender<Action>, id: &str) {
    match delete_observation(id) {
        Ok(()) => {
            let _ = action_tx.send(Action::MemDeleted(id.to_owned()));
        }
        Err(err) => {
            let _ = action_tx.send(Action::Failed(None, err.to_string()));
        }
    }
}

pub(super) fn dispatch_mem_save_prompt(
    action_tx: &Sender<Action>,
    repo: Option<&Path>,
    title: &str,
    capture: &str,
) {
    send_mem_saved(action_tx, save_prompt(repo, title, capture), "saved");
}
