use std::collections::HashSet;
use std::path::PathBuf;

use chrono::{DateTime, Utc};

use super::issue::{Issue, RecentRepo};
use super::screen::Filter;

pub const DEFAULT_PRIORITY: u8 = 2;
pub const DEFAULT_KIND: &str = "task";

pub(crate) fn priority_u8(v: i32) -> u8 {
    u8::try_from(v.clamp(0, 4)).unwrap_or(DEFAULT_PRIORITY)
}

pub(crate) fn default_kind(v: String) -> String {
    if v.trim().is_empty() {
        DEFAULT_KIND.to_owned()
    } else {
        v
    }
}

pub(crate) fn parse_time(raw: Option<&str>) -> Option<DateTime<Utc>> {
    raw.and_then(|s| {
        DateTime::parse_from_rfc3339(s)
            .ok()
            .map(|d| d.with_timezone(&Utc))
    })
}

pub(crate) fn nonempty(val: Option<String>) -> Option<String> {
    val.and_then(|s| {
        let trimmed = s.trim();
        (!trimmed.is_empty()).then_some(trimmed.to_owned())
    })
}

pub(crate) fn nonempty_opt(val: String) -> Option<String> {
    let trimmed = val.trim();
    (!trimmed.is_empty()).then_some(trimmed.to_owned())
}

fn to_local(value: DateTime<Utc>) -> DateTime<chrono::Local> {
    value.with_timezone(&chrono::Local)
}

pub fn format_local(value: DateTime<Utc>) -> String {
    to_local(value).format("%d.%m.%Y %H:%M").to_string()
}

pub fn format_clock(value: DateTime<Utc>) -> String {
    to_local(value).format("%H:%M:%S").to_string()
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct IssueIdParts {
    pub prefix: String,
    pub number: String,
}

pub fn split_issue_id(id: &str) -> IssueIdParts {
    if let Some((prefix, num)) = id.rsplit_once('-') {
        IssueIdParts {
            prefix: prefix.to_owned(),
            number: num.to_owned(),
        }
    } else {
        IssueIdParts {
            prefix: id.to_owned(),
            number: String::new(),
        }
    }
}

pub fn matches_filter(issue: &Issue, filter: Filter, ready_ids: &HashSet<String>) -> bool {
    match filter {
        Filter::Ready => ready_ids.contains(&issue.id),
        Filter::Open => issue.status == "open",
        Filter::Wip => issue.status == "active",
        Filter::Blocked => issue.status == "blocked",
        Filter::Done => issue.status == "done",
    }
}

pub fn matches_search(issue: &Issue, query: &str) -> bool {
    let trimmed = query.trim();
    if trimmed.is_empty() {
        return true;
    }
    let q = trimmed.to_ascii_lowercase();
    issue.title.to_ascii_lowercase().contains(&q) || issue.id.to_ascii_lowercase().contains(&q)
}

pub fn recent_from_path(path: PathBuf) -> RecentRepo {
    let name = path.file_name().map_or_else(
        || path.display().to_string(),
        |n| n.to_string_lossy().into_owned(),
    );
    let parent = path
        .parent()
        .map(|p| p.display().to_string())
        .unwrap_or_default();
    RecentRepo { path, name, parent }
}
