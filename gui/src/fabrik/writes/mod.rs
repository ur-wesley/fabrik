use std::path::Path;

use super::{FabrikError, Observation};
use crate::model::ShowIssueJson;

pub const ISSUE_KINDS: [&str; 5] = ["task", "bug", "feature", "epic", "chore"];
pub const MEM_KINDS: [&str; 6] = [
    "discovery",
    "decision",
    "bugfix",
    "pattern",
    "handoff",
    "prompt",
];
pub const DEP_KINDS: [&str; 2] = ["blocks", "relates"];
pub const DEFAULT_ASSIGNEE: &str = "fabrik";
pub const DEFAULT_ISSUE_KIND: &str = "task";
pub const DEFAULT_MEM_KIND: &str = "discovery";
pub const DEFAULT_DEP_KIND: &str = "blocks";

#[derive(Debug, Clone)]
pub struct IssueCreateParams {
    pub title: String,
    pub body: String,
    pub acceptance: String,
    pub priority: u8,
    pub kind: String,
}

#[derive(Debug, Clone)]
pub struct MemSaveParams {
    pub title: String,
    pub body: String,
    pub kind: String,
    pub topic_key: Option<String>,
    pub symbol_id: Option<String>,
}

#[derive(Debug, Clone)]
pub struct MemUpdateParams {
    pub id: String,
    pub title: Option<String>,
    pub body: Option<String>,
    pub kind: Option<String>,
    pub topic_key: Option<String>,
    pub symbol_id: Option<String>,
}

/// Positional + flag argv for `issue create` (pure: unit-tested, never spawns).
#[must_use]
pub fn create_args(params: &IssueCreateParams) -> Vec<String> {
    let mut args = vec![
        "issue".to_owned(),
        "create".to_owned(),
        params.title.trim().to_owned(),
    ];
    if !params.body.trim().is_empty() {
        args.push("--body".to_owned());
        args.push(params.body.clone());
    }
    if !params.acceptance.trim().is_empty() {
        args.push("--acceptance".to_owned());
        args.push(params.acceptance.clone());
    }
    args.push("--priority".to_owned());
    args.push(params.priority.clamp(0, 4).to_string());
    args.push("--kind".to_owned());
    args.push(normalize_issue_kind(&params.kind).to_owned());
    args.push("--json".to_owned());
    args
}

/// Argv for `issue claim|unclaim|close` (pure: unit-tested, never spawns).
#[must_use]
pub fn claim_args(op: &str, id: &str, by: &str, reason: &str, force: bool) -> Vec<String> {
    let mut args = vec!["issue".to_owned(), op.to_owned(), id.trim().to_owned()];
    let by = by.trim();
    if !by.is_empty() {
        args.push("--by".to_owned());
        args.push(by.to_owned());
    }
    if op == "close" && !reason.trim().is_empty() {
        args.push("--reason".to_owned());
        args.push(reason.to_owned());
    }
    if force && (op == "unclaim" || op == "close") {
        args.push("--force".to_owned());
    }
    args.push("--json".to_owned());
    args
}

/// Argv for `issue dep` (pure: unit-tested, never spawns).
#[must_use]
pub fn dep_args(blocked: &str, blocker: &str, kind: &str) -> Vec<String> {
    vec![
        "issue".to_owned(),
        "dep".to_owned(),
        blocked.trim().to_owned(),
        blocker.trim().to_owned(),
        "--kind".to_owned(),
        normalize_dep_kind(kind).to_owned(),
        "--json".to_owned(),
    ]
}

/// Argv for `mem save` (pure: unit-tested, never spawns).
#[must_use]
pub fn mem_save_args(params: &MemSaveParams) -> Vec<String> {
    let mut args = vec![
        "mem".to_owned(),
        "save".to_owned(),
        "--title".to_owned(),
        params.title.trim().to_owned(),
        "--body".to_owned(),
        params.body.clone(),
        "--kind".to_owned(),
        normalize_mem_kind(&params.kind).to_owned(),
    ];
    if let Some(topic) = params.topic_key.as_deref().map(str::trim)
        && !topic.is_empty()
    {
        args.push("--topic-key".to_owned());
        args.push(topic.to_owned());
    }
    if let Some(symbol) = params.symbol_id.as_deref().map(str::trim)
        && !symbol.is_empty()
    {
        args.push("--symbol-id".to_owned());
        args.push(symbol.to_owned());
    }
    args.push("--json".to_owned());
    args
}

/// Argv for `mem update` (pure: unit-tested, never spawns).
#[must_use]
pub fn mem_update_args(params: &MemUpdateParams) -> Vec<String> {
    let mut args = vec![
        "mem".to_owned(),
        "update".to_owned(),
        "--id".to_owned(),
        params.id.trim().to_owned(),
    ];
    for (flag, value) in [
        ("--title", params.title.as_deref()),
        ("--body", params.body.as_deref()),
        ("--kind", params.kind.as_deref()),
        ("--topic-key", params.topic_key.as_deref()),
        ("--symbol-id", params.symbol_id.as_deref()),
    ] {
        if let Some(text) = value.map(str::trim)
            && !text.is_empty()
        {
            args.push(flag.to_owned());
            args.push(text.to_owned());
        }
    }
    args.push("--json".to_owned());
    args
}

#[must_use]
pub fn normalize_issue_kind(kind: &str) -> &str {
    let trimmed = kind.trim();
    if ISSUE_KINDS.contains(&trimmed) {
        trimmed
    } else {
        DEFAULT_ISSUE_KIND
    }
}

#[must_use]
pub fn normalize_mem_kind(kind: &str) -> &str {
    let trimmed = kind.trim();
    if MEM_KINDS.contains(&trimmed) {
        trimmed
    } else {
        DEFAULT_MEM_KIND
    }
}

#[must_use]
pub fn normalize_dep_kind(kind: &str) -> &str {
    let trimmed = kind.trim();
    if DEP_KINDS.contains(&trimmed) {
        trimmed
    } else {
        DEFAULT_DEP_KIND
    }
}

#[must_use]
pub fn normalize_assignee(by: &str) -> &str {
    let trimmed = by.trim();
    if trimmed.is_empty() {
        DEFAULT_ASSIGNEE
    } else {
        trimmed
    }
}

fn run_owned(repo: Option<&Path>, args: &[String]) -> Result<String, FabrikError> {
    let refs: Vec<&str> = args.iter().map(String::as_str).collect();
    super::run_fabrik(repo, &refs)
}

fn parse_write_detail(raw: &str) -> Result<crate::model::IssueDetail, FabrikError> {
    if raw.trim().is_empty() {
        return Err(FabrikError::Parse("empty response".to_owned()));
    }
    // Every `issue … --json` write encodes the same `doltstore.Issue`
    // shape as `issue show --json`, so one parse covers create, claim,
    // unclaim, close, and dep.
    let item: ShowIssueJson =
        serde_json::from_str(raw).map_err(|err| FabrikError::Parse(err.to_string()))?;
    Ok(item.into_detail())
}

fn parse_observation(raw: &str) -> Result<Observation, FabrikError> {
    if raw.trim().is_empty() {
        return Err(FabrikError::Parse("empty response".to_owned()));
    }
    serde_json::from_str(raw).map_err(|err| FabrikError::Parse(err.to_string()))
}

/// Append a note to an issue, then return the refreshed issue detail.
///
/// `issue note <id> <body> --json` encodes a note object (`id`, `issue_id`,
/// `body`, `created_at`), not the `ShowIssueJson` issue shape that
/// `parse_write_detail` handles for create/claim/close/dep. The note payload
/// is therefore intentionally discarded (`run_owned(...)?`) and the return
/// value comes from `fetch_detail` (`issue show <id> --json`), which also
/// picks up the publish side effects (`publishIssueNote`) and any concurrent
/// updates. The extra roundtrip is required, not wasteful: the `--json` flag
/// only selects machine-readable output, it does not change the shape.
pub fn post_note(
    repo: &Path,
    id: &str,
    body: &str,
) -> Result<crate::model::IssueDetail, FabrikError> {
    if body.trim().is_empty() {
        return Err(FabrikError::Parse("note is empty".to_owned()));
    }
    run_owned(
        Some(repo),
        &[
            "issue".to_owned(),
            "note".to_owned(),
            id.trim().to_owned(),
            body.trim().to_owned(),
            "--json".to_owned(),
        ],
    )?;
    super::fetch_detail(repo, id.trim())
}

pub fn create_issue(
    repo: &Path,
    params: &IssueCreateParams,
) -> Result<crate::model::IssueDetail, FabrikError> {
    if params.title.trim().is_empty() {
        return Err(FabrikError::Parse("issue title is empty".to_owned()));
    }
    let output = run_owned(Some(repo), &create_args(params))?;
    parse_write_detail(&output)
}

pub fn claim_issue(
    repo: &Path,
    id: &str,
    by: &str,
) -> Result<crate::model::IssueDetail, FabrikError> {
    if !crate::model::is_issue_id(id.trim()) {
        return Err(FabrikError::Parse(format!(
            "invalid issue id {id:?}: must match f-<hex>"
        )));
    }
    let output = run_owned(
        Some(repo),
        &claim_args("claim", id, normalize_assignee(by), "", false),
    )?;
    parse_write_detail(&output)
}

pub fn unclaim_issue(
    repo: &Path,
    id: &str,
    by: &str,
    force: bool,
) -> Result<crate::model::IssueDetail, FabrikError> {
    if !crate::model::is_issue_id(id.trim()) {
        return Err(FabrikError::Parse(format!(
            "invalid issue id {id:?}: must match f-<hex>"
        )));
    }
    let output = run_owned(
        Some(repo),
        &claim_args("unclaim", id, normalize_assignee(by), "", force),
    )?;
    parse_write_detail(&output)
}

pub fn close_issue(
    repo: &Path,
    id: &str,
    reason: &str,
    by: &str,
    force: bool,
) -> Result<crate::model::IssueDetail, FabrikError> {
    if !crate::model::is_issue_id(id.trim()) {
        return Err(FabrikError::Parse(format!(
            "invalid issue id {id:?}: must match f-<hex>"
        )));
    }
    let output = run_owned(
        Some(repo),
        &claim_args("close", id, normalize_assignee(by), reason, force),
    )?;
    parse_write_detail(&output)
}

pub fn add_dep(
    repo: &Path,
    blocked: &str,
    blocker: &str,
    kind: &str,
) -> Result<crate::model::IssueDetail, FabrikError> {
    if !crate::model::is_issue_id(blocked.trim()) {
        return Err(FabrikError::Parse(format!(
            "invalid issue id {blocked:?}: must match f-<hex>"
        )));
    }
    if !crate::model::is_issue_id(blocker.trim()) {
        return Err(FabrikError::Parse(format!(
            "invalid issue id {blocker:?}: must match f-<hex>"
        )));
    }
    let output = run_owned(Some(repo), &dep_args(blocked, blocker, kind))?;
    parse_write_detail(&output)
}

pub fn save_observation(
    repo: Option<&Path>,
    params: &MemSaveParams,
) -> Result<Observation, FabrikError> {
    if params.title.trim().is_empty() {
        return Err(FabrikError::Parse("observation title is empty".to_owned()));
    }
    if params.body.trim().is_empty() {
        return Err(FabrikError::Parse("observation body is empty".to_owned()));
    }
    // `-C` scopes the CLI's cwd-derived project label (`RepoRoot`) to the
    // selected tab, so saves and the `this project` filter agree.
    let output = run_owned(repo, &mem_save_args(params))?;
    parse_observation(&output)
}

pub fn update_observation(params: &MemUpdateParams) -> Result<Observation, FabrikError> {
    if !crate::model::is_observation_id(params.id.trim()) {
        return Err(FabrikError::Parse(format!(
            "invalid observation id {:?}: must match o-<hex>",
            params.id
        )));
    }
    let output = run_owned(None, &mem_update_args(params))?;
    parse_observation(&output)
}

pub fn delete_observation(id: &str) -> Result<(), FabrikError> {
    if !crate::model::is_observation_id(id.trim()) {
        return Err(FabrikError::Parse(format!(
            "invalid observation id {id:?}: must match o-<hex>"
        )));
    }
    // `mem delete` has no `--json`: success is exit 0 with empty stdout.
    run_owned(
        None,
        &[
            "mem".to_owned(),
            "delete".to_owned(),
            "--id".to_owned(),
            id.trim().to_owned(),
        ],
    )?;
    Ok(())
}

pub fn save_prompt(
    repo: Option<&Path>,
    title: &str,
    capture: &str,
) -> Result<Observation, FabrikError> {
    if capture.trim().is_empty() {
        return Err(FabrikError::Parse("prompt text is empty".to_owned()));
    }
    let output = run_owned(
        repo,
        &[
            "mem".to_owned(),
            "save-prompt".to_owned(),
            "--title".to_owned(),
            title.trim().to_owned(),
            "--capture-prompt".to_owned(),
            capture.to_owned(),
            "--json".to_owned(),
        ],
    )?;
    parse_observation(&output)
}

#[cfg(test)]
mod tests;
