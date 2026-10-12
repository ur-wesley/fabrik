use std::collections::HashSet;
use std::path::PathBuf;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

use super::screen::Filter;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Issue {
    pub id: String,
    pub title: String,
    pub status: String,
    pub priority: u8,
    pub kind: String,
    pub assignee: Option<String>,
    pub updated_at: Option<DateTime<Utc>>,
    pub description: Option<String>,
    pub acceptance_criteria: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct DepNode {
    pub id: String,
    pub title: String,
    pub status: Option<String>,
    pub priority: Option<u8>,
    pub kind: Option<String>,
    pub dep_type: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct Comment {
    pub author: Option<String>,
    pub body: String,
    pub created_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct IssueDetail {
    pub id: String,
    pub title: String,
    pub description: Option<String>,
    pub acceptance_criteria: Option<String>,
    pub status: String,
    pub priority: u8,
    pub kind: String,
    pub assignee: Option<String>,
    pub owner: Option<String>,
    pub parent: Option<String>,
    pub created_at: Option<DateTime<Utc>>,
    pub updated_at: Option<DateTime<Utc>>,
    pub started_at: Option<DateTime<Utc>>,
    pub closed_at: Option<DateTime<Utc>>,
    pub dependencies: Vec<DepNode>,
    pub dependents: Vec<DepNode>,
    pub dependency_count: usize,
    pub dependent_count: usize,
    pub comment_count: usize,
    #[serde(default)]
    pub comments: Vec<Comment>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct IssueRow {
    pub id: String,
    pub title: String,
    pub status: String,
    pub priority: u8,
    pub kind: String,
    pub flashed: bool,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct KanbanLane {
    pub filter: Filter,
    pub label: &'static str,
    pub cards: Vec<IssueRow>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RecentRepo {
    pub path: PathBuf,
    pub name: String,
    pub parent: String,
}

#[derive(Debug, Clone)]
pub struct Snapshot {
    pub repo: PathBuf,
    pub issues: Vec<Issue>,
    pub ready_ids: HashSet<String>,
    pub fetched_at: DateTime<Utc>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ClosedTab {
    pub path: PathBuf,
    pub index: usize,
}
