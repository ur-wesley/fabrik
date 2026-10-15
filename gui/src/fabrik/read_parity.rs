use std::collections::BTreeMap;
use std::path::Path;

use serde::Deserialize;

use super::{
    ContextStats, ExploreResult, FabrikError, GraphEdge, GraphNode, Observation, QueryResult,
};

#[derive(Debug, Clone, Deserialize)]
pub struct ContextObservation {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub title: String,
    #[serde(default)]
    pub kind: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ContextTest {
    #[serde(default)]
    pub path: String,
    #[serde(default)]
    pub symbol: String,
}

#[derive(Debug, Clone, Deserialize)]
pub struct GraphContextResult {
    pub symbol: GraphNode,
    #[serde(default)]
    pub callers: Vec<GraphNode>,
    #[serde(default)]
    pub callees: Vec<GraphEdge>,
    #[serde(default)]
    pub observations: Vec<ContextObservation>,
    #[serde(default)]
    pub tests: Vec<ContextTest>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct GraphImpactResult {
    #[serde(default)]
    pub root: String,
    #[serde(default)]
    pub by_depth: BTreeMap<String, Vec<GraphNode>>,
    #[serde(default)]
    pub total: i32,
}

impl GraphImpactResult {
    #[must_use]
    pub fn flattened(&self) -> Vec<&GraphNode> {
        let mut out = Vec::new();
        for nodes in self.by_depth.values() {
            for node in nodes {
                out.push(node);
            }
        }
        out
    }
}

#[derive(Debug, Clone, Deserialize)]
pub struct DetectChangesResult {
    #[serde(default)]
    pub changed_files: Vec<String>,
    #[serde(default)]
    pub touched_symbols: Vec<String>,
    #[serde(default)]
    pub impacted_symbols: Vec<String>,
    #[serde(default)]
    pub suggested_checks: Vec<String>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct MemCheckResult {
    #[serde(default)]
    pub memory: String,
    #[serde(default)]
    pub memory_path: String,
    #[serde(default)]
    pub schema_memory: i32,
    #[serde(default)]
    pub observations: i32,
    #[serde(default)]
    pub sessions: i32,
    #[serde(default)]
    pub projects: Vec<String>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct MemTimelineResult {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub before: Vec<Observation>,
    #[serde(default)]
    pub after: Vec<Observation>,
    pub observation: Observation,
}

#[derive(Debug, Clone, Deserialize)]
pub struct CurrentProjectResult {
    #[serde(default)]
    pub project: String,
    #[serde(default)]
    pub source: String,
}

pub fn fetch_graph_context(repo: &Path, symbol: &str) -> Result<GraphContextResult, FabrikError> {
    let query = symbol.trim();
    if query.is_empty() {
        return Err(FabrikError::Parse(
            "graph context needs a symbol".to_owned(),
        ));
    }
    // Symbol-only gate (mirrors the `is_issue_id` write guards): file
    // anchors resolve to `node sym:file:… not found` live, so reject
    // client-side without spawning.
    if !crate::model::is_symbol_anchor(query) {
        return Err(FabrikError::Parse(format!(
            "graph context needs a symbol anchor, got {query:?}"
        )));
    }
    let output = super::run_fabrik(Some(repo), &["graph", "context", query, "--json"])?;
    super::parse_one(&output)
}

pub fn fetch_graph_impact(
    repo: &Path,
    symbol: &str,
    depth: u32,
) -> Result<GraphImpactResult, FabrikError> {
    let query = symbol.trim();
    if query.is_empty() {
        return Err(FabrikError::Parse("graph impact needs a symbol".to_owned()));
    }
    // Same symbol-only gate as context: file anchors never spawn.
    if !crate::model::is_symbol_anchor(query) {
        return Err(FabrikError::Parse(format!(
            "graph impact needs a symbol anchor, got {query:?}"
        )));
    }
    let clamped = depth.clamp(1, 10).to_string();
    let output = super::run_fabrik(
        Some(repo),
        &[
            "graph",
            "impact",
            query,
            "--depth",
            clamped.as_str(),
            "--json",
        ],
    )?;
    super::parse_one(&output)
}

pub fn fetch_graph_detect_changes(
    repo: &Path,
    base_ref: &str,
) -> Result<DetectChangesResult, FabrikError> {
    let output = if base_ref.trim().is_empty() {
        super::run_fabrik(Some(repo), &["graph", "detect-changes", "--json"])?
    } else {
        super::run_fabrik(
            Some(repo),
            &[
                "graph",
                "detect-changes",
                "--ref",
                base_ref.trim(),
                "--json",
            ],
        )?
    };
    super::parse_one(&output)
}

pub fn fetch_graph_explore_issue(
    repo: &Path,
    query: &str,
    issue_id: &str,
) -> Result<ExploreResult, FabrikError> {
    let issue = issue_id.trim();
    if issue.is_empty() || !crate::model::is_issue_id(issue) {
        return Err(FabrikError::Parse(format!(
            "invalid issue id {issue:?}: must match f-<hex>"
        )));
    }
    let effective = if query.trim().is_empty() {
        issue
    } else {
        query.trim()
    };
    let output = super::run_fabrik(
        Some(repo),
        &[
            "graph", "explore", effective, "--issue", issue, "--budget", "40", "--json",
        ],
    )?;
    super::parse_one(&output)
}

pub fn fetch_graph_overview(repo: &Path, budget: u32) -> Result<QueryResult, FabrikError> {
    let clamped = budget.clamp(1, 200).to_string();
    let output = super::run_fabrik(
        Some(repo),
        &["graph", "overview", "--budget", clamped.as_str(), "--json"],
    )?;
    super::parse_one(&output)
}

pub fn fetch_mem_check() -> Result<MemCheckResult, FabrikError> {
    let output = super::run_fabrik(None, &["mem", "check", "--json"])?;
    super::parse_one(&output)
}

pub fn fetch_mem_stats() -> Result<ContextStats, FabrikError> {
    let output = super::run_fabrik(None, &["mem", "stats", "--json"])?;
    super::parse_one(&output)
}

pub fn fetch_mem_timeline(
    id: &str,
    limit: u32,
    project: Option<&Path>,
) -> Result<MemTimelineResult, FabrikError> {
    // Legacy ID split: context/search accept legacy numeric ids, but
    // timeline is strict like get — ids must match `o-<hex>` (CLI
    // `observationIDRe`). Empty is rejected client-side without spawning;
    // callers gate non-hex ids via `is_observation_id` before dispatch.
    let trimmed = id.trim();
    if trimmed.is_empty() {
        return Err(FabrikError::Parse(
            "mem timeline needs an observation id".to_owned(),
        ));
    }
    let clamped = limit.clamp(1, 50).to_string();
    let output = if let Some(p) = project {
        let label = crate::model::project_label(p);
        super::run_fabrik(
            Some(p),
            &[
                "mem",
                "timeline",
                "--id",
                trimmed,
                "--limit",
                clamped.as_str(),
                "--project",
                label.as_str(),
                "--json",
            ],
        )?
    } else {
        super::run_fabrik(
            None,
            &[
                "mem",
                "timeline",
                "--id",
                trimmed,
                "--limit",
                clamped.as_str(),
                "--json",
            ],
        )?
    };
    super::parse_one(&output)
}

pub fn fetch_mem_current_project(repo: Option<&Path>) -> Result<CurrentProjectResult, FabrikError> {
    let output = super::run_fabrik(repo, &["mem", "current-project", "--json"])?;
    super::parse_one(&output)
}
