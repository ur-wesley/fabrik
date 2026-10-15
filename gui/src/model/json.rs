use serde::Deserialize;

use super::issue::{Comment, DepNode, Issue, IssueDetail};
use super::query::{
    DEFAULT_PRIORITY, default_kind, nonempty, nonempty_opt, parse_time, priority_u8,
};

#[derive(Debug, Clone, Deserialize)]
pub struct IssueJson {
    pub id: String,
    #[serde(default)]
    pub title: String,
    #[serde(default)]
    pub body: String,
    #[serde(default)]
    pub acceptance: String,
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub priority: i32,
    #[serde(default)]
    pub kind: String,
    #[serde(default)]
    pub assignee: Option<String>,
    #[serde(default)]
    pub updated_at: Option<String>,
    // Live `issue list --json` also sends these; the board only needs the
    // fields above, but parsing must tolerate the full shape.
    #[serde(default)]
    #[allow(dead_code)]
    pub ready: bool,
    #[serde(default)]
    #[allow(dead_code)]
    pub created_at: Option<String>,
    #[serde(default)]
    #[allow(dead_code)]
    pub started_at: Option<String>,
    #[serde(default)]
    #[allow(dead_code)]
    pub closed_at: Option<String>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct IssueRefJson {
    pub id: String,
    #[serde(default)]
    pub title: String,
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub priority: i32,
    #[serde(default)]
    pub kind: String,
}

impl IssueRefJson {
    pub fn into_dep(self) -> DepNode {
        let pri = u8::try_from(self.priority.clamp(0, 4)).unwrap_or(DEFAULT_PRIORITY);
        DepNode {
            id: self.id,
            title: self.title,
            status: Some(self.status),
            priority: Some(pri),
            kind: Some(self.kind),
            dep_type: None,
        }
    }
}

#[derive(Debug, Clone, Deserialize)]
pub struct ShowIssueJson {
    pub id: String,
    #[serde(default)]
    pub title: String,
    #[serde(default)]
    pub body: String,
    #[serde(default)]
    pub acceptance: String,
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub priority: i32,
    #[serde(default)]
    pub kind: String,
    #[serde(default)]
    pub assignee: Option<String>,
    #[serde(default)]
    pub created_at: Option<String>,
    #[serde(default)]
    pub updated_at: Option<String>,
    #[serde(default)]
    pub started_at: Option<String>,
    #[serde(default)]
    pub closed_at: Option<String>,
    #[serde(default)]
    pub blockers: Vec<IssueRefJson>,
    #[serde(default)]
    pub blocking: Vec<IssueRefJson>,
    #[serde(default)]
    pub notes: Vec<NoteJson>,
    #[serde(default)]
    #[allow(dead_code)]
    pub ready: bool,
}

#[derive(Debug, Clone, Deserialize)]
pub struct NoteJson {
    #[allow(dead_code)]
    pub id: String,
    #[serde(default)]
    #[allow(dead_code)]
    pub issue_id: Option<String>,
    #[serde(default)]
    pub body: String,
    #[serde(default)]
    pub created_at: Option<String>,
}

impl NoteJson {
    pub fn into_comment(self) -> Comment {
        Comment {
            author: None,
            body: self.body,
            created_at: parse_time(self.created_at.as_deref()),
        }
    }
}

impl ShowIssueJson {
    pub fn into_detail(self) -> IssueDetail {
        let dependencies: Vec<DepNode> = self
            .blockers
            .into_iter()
            .map(IssueRefJson::into_dep)
            .collect();
        let dependents: Vec<DepNode> = self
            .blocking
            .into_iter()
            .map(IssueRefJson::into_dep)
            .collect();
        let comments: Vec<Comment> = self.notes.into_iter().map(NoteJson::into_comment).collect();
        let dependency_count = dependencies.len();
        let dependent_count = dependents.len();
        let comment_count = comments.len();
        IssueDetail {
            id: self.id,
            title: self.title,
            description: nonempty_opt(self.body),
            acceptance_criteria: nonempty_opt(self.acceptance),
            status: self.status,
            priority: priority_u8(self.priority),
            kind: default_kind(self.kind),
            assignee: nonempty(self.assignee),
            owner: None,
            parent: None,
            created_at: parse_time(self.created_at.as_deref()),
            updated_at: parse_time(self.updated_at.as_deref()),
            started_at: parse_time(self.started_at.as_deref()),
            closed_at: parse_time(self.closed_at.as_deref()),
            dependencies,
            dependents,
            dependency_count,
            dependent_count,
            comment_count,
            comments,
        }
    }
}

#[derive(Debug, Clone, Deserialize)]
#[serde(untagged)]
pub enum DepJson {
    Id(String),
    Object(DepFields),
}

#[derive(Debug, Clone, Deserialize)]
pub struct DepFields {
    pub id: String,
    #[serde(default)]
    pub title: Option<String>,
    #[serde(default)]
    pub status: Option<String>,
    #[serde(default)]
    pub priority: Option<u8>,
    #[serde(rename = "type", default)]
    pub kind: Option<String>,
    #[serde(default)]
    pub dep_type: Option<String>,
}

impl DepJson {
    pub fn into_node(self) -> DepNode {
        match self {
            Self::Id(id) => DepNode {
                id,
                title: String::new(),
                status: None,
                priority: None,
                kind: None,
                dep_type: None,
            },
            Self::Object(fields) => DepNode {
                id: fields.id,
                title: fields.title.unwrap_or_default(),
                status: fields.status,
                priority: fields.priority,
                kind: fields.kind,
                dep_type: fields.dep_type,
            },
        }
    }
}

#[allow(dead_code)]
pub fn deserialize_dep_list<'de, D>(deserializer: D) -> Result<Vec<DepNode>, D::Error>
where
    D: serde::Deserializer<'de>,
{
    let raw = Option::<Vec<DepJson>>::deserialize(deserializer)?;
    Ok(raw
        .unwrap_or_default()
        .into_iter()
        .map(DepJson::into_node)
        .collect())
}

impl IssueJson {
    pub fn into_issue(self) -> Issue {
        Issue {
            id: self.id,
            title: self.title,
            status: self.status,
            priority: priority_u8(self.priority),
            kind: default_kind(self.kind),
            assignee: nonempty(self.assignee),
            updated_at: parse_time(self.updated_at.as_deref()),
            description: nonempty_opt(self.body),
            acceptance_criteria: nonempty_opt(self.acceptance),
        }
    }
}
