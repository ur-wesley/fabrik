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

/// Port of the CLI `issueIDRe` (`^f-[0-9a-f]{8,32}$`): live issue IDs are
/// `f-` plus 8-32 lowercase hex chars. Anything else is rejected by
/// `fabrik issue show <id>` before it touches the store.
pub fn is_issue_id(id: &str) -> bool {
    let Some(hex) = id.strip_prefix("f-") else {
        return false;
    };
    (8..=32).contains(&hex.len())
        && hex
            .bytes()
            .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase())
}

/// Port of the CLI `observationIDRe` (`^o-[0-9a-f]+$`). Legacy numeric
/// memory IDs (e.g. `"358"`) fail this check and are rejected by
/// `fabrik mem get <id>` as well.
pub fn is_observation_id(id: &str) -> bool {
    let Some(hex) = id.strip_prefix("o-") else {
        return false;
    };
    !hex.is_empty()
        && hex
            .bytes()
            .all(|b| b.is_ascii_hexdigit() && !b.is_ascii_uppercase())
}

/// Graph anchor classifier (mirrors the `is_issue_id` / `is_observation_id`
/// validator pattern): `graph context` / `graph impact` resolve symbols
/// only, so only `sym:` anchors may spawn a fetch. File anchors
/// (`file:…`) show path info + hint in the header instead of firing —
/// live-confirmed: `graph context 'file:…'` fails with
/// `node sym:file:… not found`.
#[must_use]
pub fn is_symbol_anchor(id: &str) -> bool {
    id.trim().starts_with("sym:")
}

/// File-node anchors (`file:…`) carry a path, not a symbol: context and
/// impact buttons stay disabled while one is selected.
#[must_use]
pub fn is_file_anchor(id: &str) -> bool {
    id.trim().starts_with("file:")
}

/// CLI-expected `--project` label for `mem search`/`mem context`: the label
/// the CLI stores is the absolute repo root (`store.RepoRoot`), so a
/// relative GUI path must be made absolute first. Absolute paths pass
/// through untouched (no symlink resolution: `canonicalize` would prepend
/// `\\?\` on Windows and break the label match).
pub fn project_label(repo: &std::path::Path) -> String {
    if repo.is_absolute() {
        return repo.to_string_lossy().into_owned();
    }
    std::path::absolute(repo).map_or_else(
        |_| repo.to_string_lossy().into_owned(),
        |abs| abs.to_string_lossy().into_owned(),
    )
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
