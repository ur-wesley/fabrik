use std::collections::HashSet;
#[cfg(windows)]
use std::os::windows::process::CommandExt;
use std::path::Path;
use std::process::Command;

use chrono::Utc;
use rayon::prelude::*;
use serde::Deserialize;

use crate::model::{Issue, IssueDetail, ShowIssueJson, Snapshot};

#[derive(Debug)]
pub enum FabrikError {
    Spawn(String),
    Exit(i32, String),
    Parse(String),
}

impl std::fmt::Display for FabrikError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Spawn(msg) => write!(f, "failed to run fabrik: {msg}"),
            Self::Exit(code, msg) => write!(f, "fabrik exited {code}: {msg}"),
            Self::Parse(msg) => write!(f, "fabrik json parse failed: {msg}"),
        }
    }
}

pub fn fabrik_bin() -> String {
    std::env::var("FABRIK_BIN").unwrap_or_else(|_| "fabrik".to_owned())
}

pub fn fetch_snapshot(repo: &Path) -> Result<Snapshot, FabrikError> {
    let (issues, ready) = rayon::join(|| list_all(repo), || ready_ids(repo));
    match (issues, ready) {
        (Ok(issues), Ok(ready_ids)) => Ok(Snapshot {
            repo: repo.to_path_buf(),
            issues,
            ready_ids,
            fetched_at: Utc::now(),
        }),
        (Err(err), _) | (_, Err(err)) => Err(err),
    }
}

pub fn fetch_detail(repo: &Path, id: &str) -> Result<IssueDetail, FabrikError> {
    let output = run_fabrik(Some(repo), &["issue", "show", id, "--json"])?;
    let item: ShowIssueJson = parse_one(&output)?;
    Ok(item.into_detail())
}

pub fn post_note(repo: &Path, id: &str, body: &str) -> Result<IssueDetail, FabrikError> {
    if body.trim().is_empty() {
        return Err(FabrikError::Parse("note is empty".to_owned()));
    }
    run_fabrik(Some(repo), &["issue", "note", id, body.trim(), "--json"])?;
    fetch_detail(repo, id)
}

fn list_all(repo: &Path) -> Result<Vec<Issue>, FabrikError> {
    let output = run_fabrik(Some(repo), &["issue", "list", "--json"])?;
    let items = parse_array::<crate::model::IssueJson>(&output)?;
    Ok(items.into_par_iter().map(|j| j.into_issue()).collect())
}

fn ready_ids(repo: &Path) -> Result<HashSet<String>, FabrikError> {
    let output = run_fabrik(Some(repo), &["issue", "ready", "--json"])?;
    let items = parse_array::<crate::model::IssueJson>(&output)?;
    Ok(items.into_iter().map(|item| item.id).collect())
}

pub fn run_fabrik(repo: Option<&Path>, args: &[&str]) -> Result<String, FabrikError> {
    let mut command = Command::new(fabrik_bin());
    if let Some(r) = repo {
        command.arg("-C").arg(r);
    }
    command.args(args);
    #[cfg(windows)]
    command.creation_flags(0x0800_0000);
    let output = command
        .output()
        .map_err(|err| FabrikError::Spawn(err.to_string()))?;
    let stdout = String::from_utf8_lossy(&output.stdout).trim().to_owned();
    let stderr = String::from_utf8_lossy(&output.stderr).trim().to_owned();
    if output.status.success() {
        Ok(stdout)
    } else {
        let message = if stderr.is_empty() { stdout } else { stderr };
        let code = output.status.code().unwrap_or(-1);
        Err(FabrikError::Exit(code, message))
    }
}

fn parse_array<T>(raw: &str) -> Result<Vec<T>, FabrikError>
where
    T: for<'de> Deserialize<'de>,
{
    if raw.is_empty() {
        return Ok(Vec::new());
    }
    serde_json::from_str(raw).map_err(|err| FabrikError::Parse(err.to_string()))
}

fn parse_one<T>(raw: &str) -> Result<T, FabrikError>
where
    T: for<'de> Deserialize<'de>,
{
    if raw.is_empty() {
        return Err(FabrikError::Parse("empty response".to_owned()));
    }
    serde_json::from_str(raw).map_err(|err| FabrikError::Parse(err.to_string()))
}

#[derive(Debug, Clone, Deserialize)]
pub struct MemSession {
    pub id: String,
    #[serde(default)]
    pub started_at: String,
    #[serde(default)]
    pub ended_at: Option<String>,
    #[serde(default)]
    pub goal: String,
    #[serde(default)]
    pub discoveries: String,
    #[serde(default)]
    pub accomplished: String,
    #[serde(default)]
    pub next_steps: String,
    #[serde(default)]
    pub files: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct Observation {
    pub id: String,
    #[serde(default)]
    pub session_id: Option<String>,
    #[serde(default)]
    pub project: String,
    #[serde(default)]
    pub kind: String,
    #[serde(default)]
    pub title: String,
    #[serde(default)]
    pub body: String,
    #[serde(default)]
    pub created_at: String,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct ContextStats {
    #[serde(default)]
    pub observation_count: i64,
    #[serde(default)]
    pub session_count: i64,
    #[serde(default)]
    pub projects: Vec<String>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct MemContext {
    pub session: Option<MemSession>,
    #[serde(default)]
    pub observations: Vec<Observation>,
    #[serde(default)]
    pub stats: ContextStats,
}

pub fn run_project_init(repo: &Path, apps: &[String]) -> Result<(), FabrikError> {
    let mut args: Vec<String> = vec![
        "init".into(),
        "--yes".into(),
        "--json".into(),
        "--skip-checks".into(),
    ];
    for app in apps {
        args.push("--apps".into());
        args.push(app.clone());
    }
    let arg_refs: Vec<&str> = args.iter().map(String::as_str).collect();
    run_fabrik(Some(repo), &arg_refs)?;
    Ok(())
}

pub fn fetch_mem_context() -> Result<MemContext, FabrikError> {
    let output = run_fabrik(None, &["mem", "context", "--json"])?;
    parse_one(&output)
}

pub fn fetch_mem_search(query: &str, project: Option<&Path>) -> Result<Vec<Observation>, FabrikError> {
    let output = if let Some(p) = project {
        let path = p.to_string_lossy();
        run_fabrik(
            None,
            &["mem", "search", query, "--project", path.as_ref(), "--json"],
        )?
    } else {
        run_fabrik(None, &["mem", "search", query, "--json"])?
    };
    if output.is_empty() || output == "null" {
        return Ok(Vec::new());
    }
    parse_array(&output)
}

pub fn fetch_mem_get(id: &str) -> Result<Observation, FabrikError> {
    let output = run_fabrik(None, &["mem", "get", id, "--json"])?;
    parse_one(&output)
}

#[derive(Debug, Clone, Deserialize)]
pub struct GraphNode {
    pub id: String,
    #[serde(default)]
    pub kind: String,
    #[serde(default)]
    pub name: String,
    #[serde(default)]
    pub path: String,
    #[serde(default)]
    pub line: i32,
    #[serde(default)]
    pub lang: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct GraphEdge {
    pub src: String,
    #[serde(default)]
    pub dst: String,
    #[serde(default)]
    pub kind: String,
    #[serde(default)]
    pub label: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ExploreResult {
    #[serde(default)]
    pub indexed: bool,
    #[serde(default)]
    pub hint: String,
    #[serde(default)]
    pub files: Vec<ExploreFile>,
    #[serde(default)]
    pub paths: Vec<ExplorePath>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ExploreFile {
    pub path: String,
    #[serde(default)]
    pub snippets: Vec<ExploreSnippet>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ExploreSnippet {
    #[serde(default)]
    pub line_start: i32,
    #[serde(default)]
    pub line_end: i32,
    #[serde(default)]
    pub symbol: String,
    #[serde(default)]
    pub text: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ExplorePath {
    pub from: String,
    pub to: String,
    #[serde(default)]
    pub hops: Vec<String>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ExplainResult {
    pub node: GraphNode,
    #[serde(default)]
    pub edges: Vec<GraphEdge>,
    #[serde(default)]
    pub truncated: bool,
}

#[derive(Debug, Clone, Deserialize)]
pub struct PathResult {
    #[serde(default)]
    pub nodes: Vec<GraphNode>,
    #[serde(default)]
    pub edges: Vec<GraphEdge>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct QueryResult {
    #[serde(default)]
    pub nodes: Vec<GraphNode>,
    #[serde(default)]
    pub edges: Vec<GraphEdge>,
}

pub fn nodes_from_explore(explore: &ExploreResult) -> Vec<GraphNode> {
    let mut nodes = Vec::new();
    for file in &explore.files {
        for snip in &file.snippets {
            let id = if snip.symbol.is_empty() {
                format!("file:{}", file.path)
            } else {
                format!("sym:{}:{}", file.path, snip.symbol)
            };
            nodes.push(GraphNode {
                id,
                kind: if snip.symbol.is_empty() {
                    "file".to_owned()
                } else {
                    "symbol".to_owned()
                },
                name: if snip.symbol.is_empty() {
                    file.path.clone()
                } else {
                    snip.symbol.clone()
                },
                path: file.path.clone(),
                line: snip.line_start,
                lang: String::new(),
            });
        }
    }
    nodes
}

pub fn fetch_graph_explore(repo: &Path, query: &str) -> Result<ExploreResult, FabrikError> {
    let output = run_fabrik(
        Some(repo),
        &["graph", "explore", query, "--budget", "40", "--json"],
    )?;
    parse_one(&output)
}

pub fn fetch_graph_explain(repo: &Path, id: &str) -> Result<ExplainResult, FabrikError> {
    let output = run_fabrik(Some(repo), &["graph", "explain", id, "--json"])?;
    parse_one(&output)
}

pub fn fetch_graph_path(repo: &Path, from: &str, to: &str) -> Result<PathResult, FabrikError> {
    let output = run_fabrik(
        Some(repo),
        &["graph", "path", from, to, "--json"],
    )?;
    parse_one(&output)
}

pub fn fetch_graph_query(repo: &Path, query: &str) -> Result<QueryResult, FabrikError> {
    let output = run_fabrik(
        Some(repo),
        &["graph", "query", query, "--budget", "40", "--json"],
    )?;
    parse_one(&output)
}

#[derive(Debug, Clone, Deserialize)]
pub struct GraphStatus {
    #[serde(default)]
    pub indexed: bool,
    #[serde(default)]
    pub files: i32,
    #[serde(default)]
    pub symbols: i32,
    #[serde(default)]
    pub stale_files: i32,
    #[serde(default)]
    pub last_build_at: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct GraphBuildResult {
    #[serde(default)]
    pub files: i32,
    #[serde(default)]
    pub parsed: i32,
    #[serde(default)]
    pub skipped: i32,
    #[serde(default)]
    pub nodes: i32,
    #[serde(default)]
    pub edges: i32,
}

pub fn fetch_graph_status(repo: &Path) -> Result<GraphStatus, FabrikError> {
    let output = run_fabrik(Some(repo), &["graph", "status", "--json"])?;
    parse_one(&output)
}

pub fn run_graph_build(repo: &Path) -> Result<GraphBuildResult, FabrikError> {
    let output = run_fabrik(Some(repo), &["graph", "build", "--json"])?;
    parse_one(&output)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn mem_search_null_is_empty_vec() {
        let raw = "null";
        let parsed: Vec<Observation> = if raw.is_empty() || raw == "null" {
            Vec::new()
        } else {
            serde_json::from_str(raw).unwrap()
        };
        assert!(parsed.is_empty());
    }

    #[test]
    fn explore_empty_files_yields_no_nodes() {
        let explore = ExploreResult {
            indexed: true,
            hint: String::new(),
            files: Vec::new(),
            paths: Vec::new(),
        };
        assert!(nodes_from_explore(&explore).is_empty());
    }
}
