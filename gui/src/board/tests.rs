use std::collections::HashSet;
use std::path::{Path, PathBuf};

use chrono::Utc;

use crate::model::{ALL_LANES, Issue, IssueDetail, Snapshot};

use super::BoardStore;

fn lock_cache_env() -> std::sync::MutexGuard<'static, ()> {
    crate::cache::TEST_ENV_LOCK
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner)
}

fn snapshot_for(repo: &str, ids: &[&str]) -> Snapshot {
    Snapshot {
        repo: PathBuf::from(repo),
        issues: ids
            .iter()
            .map(|id| Issue {
                id: (*id).to_owned(),
                title: (*id).to_owned(),
                status: "open".to_owned(),
                priority: 2,
                kind: "task".to_owned(),
                assignee: None,
                updated_at: None,
                description: None,
                acceptance_criteria: None,
            })
            .collect(),
        ready_ids: HashSet::new(),
        fetched_at: Utc::now(),
    }
}

fn detail_for(id: &str) -> IssueDetail {
    IssueDetail {
        id: id.to_owned(),
        title: id.to_owned(),
        description: None,
        acceptance_criteria: None,
        status: "open".to_owned(),
        priority: 2,
        kind: "task".to_owned(),
        assignee: None,
        owner: None,
        parent: None,
        created_at: None,
        updated_at: None,
        started_at: None,
        closed_at: None,
        dependencies: Vec::new(),
        dependents: Vec::new(),
        dependency_count: 0,
        dependent_count: 0,
        comment_count: 0,
        comments: Vec::new(),
    }
}

#[test]
fn open_detail_seeds_preview() {
    let _guard = lock_cache_env();
    let mut board = BoardStore::new(HashSet::new());
    board.issues.push(Issue {
        id: "f-a1b2c3d4".to_owned(),
        title: "one".to_owned(),
        status: "open".to_owned(),
        priority: 1,
        kind: "task".to_owned(),
        assignee: None,
        updated_at: None,
        description: None,
        acceptance_criteria: None,
    });
    board.open_detail("f-a1b2c3d4", None);
    assert_eq!(board.detail.as_ref().map(|d| d.title.as_str()), Some("one"));
}

#[test]
fn lanes_cover_all_filters() {
    let board = BoardStore::new(HashSet::new());
    let lanes = board.lanes();
    assert_eq!(lanes.len(), ALL_LANES.len());
}

#[test]
fn open_detail_uses_list_markdown() {
    let _guard = lock_cache_env();
    let mut board = BoardStore::new(HashSet::new());
    board.issues.push(Issue {
        id: "f-a1b2c3d4".to_owned(),
        title: "one".to_owned(),
        status: "open".to_owned(),
        priority: 1,
        kind: "task".to_owned(),
        assignee: None,
        updated_at: None,
        description: Some("body markdown".to_owned()),
        acceptance_criteria: Some("criteria markdown".to_owned()),
    });
    board.open_detail("f-a1b2c3d4", None);
    let detail = board.detail.as_ref().unwrap();
    assert_eq!(detail.description.as_deref(), Some("body markdown"));
    assert_eq!(
        detail.acceptance_criteria.as_deref(),
        Some("criteria markdown")
    );
}

#[test]
fn rapid_switch_ignores_stale_snapshot() {
    let mut board = BoardStore::new(HashSet::new());
    let repo_b = PathBuf::from("b");
    board.apply_snapshot(snapshot_for("b", &["b-1"]), Some(&repo_b));
    assert_eq!(board.issues.len(), 1);
    assert_eq!(board.issues[0].id, "b-1");
    board.apply_snapshot(snapshot_for("a", &["a-1", "a-2"]), Some(&repo_b));
    assert_eq!(board.issues.len(), 1);
    assert_eq!(board.issues[0].id, "b-1");
}

#[test]
fn snapshot_for_current_repo_applies() {
    let mut board = BoardStore::new(HashSet::new());
    let repo_b = PathBuf::from("b");
    board.apply_snapshot(snapshot_for("b", &["b-1", "b-2"]), Some(&repo_b));
    assert_eq!(board.issues.len(), 2);
    assert!(!board.loading);
    assert!(board.error.is_none());
}

#[test]
fn stale_detail_loaded_is_ignored() {
    let _guard = lock_cache_env();
    let mut board = BoardStore::new(HashSet::new());
    let repo_b = PathBuf::from("b");
    board.open_detail("f-a1b2c3d4", Some(&repo_b));
    board.apply_detail_loaded(
        Path::new("a"),
        Box::new(detail_for("f-a1b2c3d4")),
        Some(&repo_b),
    );
    assert!(!board.detail_cache.contains_key("f-a1b2c3d4"));
    board.apply_detail_loaded(
        Path::new("b"),
        Box::new(detail_for("f-a1b2c3d4")),
        Some(&repo_b),
    );
    assert!(board.detail_cache.contains_key("f-a1b2c3d4"));
    assert_eq!(
        board.detail.as_ref().map(|d| d.id.as_str()),
        Some("f-a1b2c3d4")
    );
}

#[test]
fn stale_failed_does_not_clear_loading() {
    let mut board = BoardStore::new(HashSet::new());
    let repo_b = PathBuf::from("b");
    board.loading = true;
    board.apply_failed(Some(Path::new("a")), "old boom".to_owned(), Some(&repo_b));
    assert!(board.loading);
    assert!(board.error.is_none());
    board.apply_failed(Some(Path::new("b")), "new boom".to_owned(), Some(&repo_b));
    assert!(!board.loading);
    assert_eq!(board.error.as_deref(), Some("new boom"));
}

#[test]
fn comment_post_sets_spinner_and_success_clears_draft() {
    let _guard = lock_cache_env();
    let mut board = BoardStore::new(HashSet::new());
    let repo_b = PathBuf::from("b");
    board.open_detail("f-a1b2c3d4", Some(&repo_b));
    board.comment_draft = "hello".to_owned();
    board.comment_posting = true;
    board.apply_detail_loaded(
        Path::new("b"),
        Box::new(detail_for("f-a1b2c3d4")),
        Some(&repo_b),
    );
    assert!(!board.comment_posting);
    assert_eq!(board.comment_draft, "");
    assert!(board.comment_error.is_none());
}

#[test]
fn detail_refresh_while_typing_keeps_draft() {
    let _guard = lock_cache_env();
    let mut board = BoardStore::new(HashSet::new());
    let repo_b = PathBuf::from("b");
    board.open_detail("f-a1b2c3d4", Some(&repo_b));
    board.comment_draft = "typing…".to_owned();
    board.apply_detail_loaded(
        Path::new("b"),
        Box::new(detail_for("f-a1b2c3d4")),
        Some(&repo_b),
    );
    assert_eq!(board.comment_draft, "typing…");
}

#[test]
fn open_detail_resets_composer() {
    let _guard = lock_cache_env();
    let mut board = BoardStore::new(HashSet::new());
    let repo_b = PathBuf::from("b");
    board.open_detail("f-a1b2c3d4", Some(&repo_b));
    board.comment_draft = "hello".to_owned();
    board.comment_posting = true;
    board.open_detail("f-b2c3d4e5", Some(&repo_b));
    assert_eq!(board.comment_draft, "");
    assert!(!board.comment_posting);
    assert!(board.comment_error.is_none());
}

#[test]
fn resolve_issue_id_exact_prefix_substring() {
    let mut board = BoardStore::new(HashSet::new());
    for id in ["f-a1b2c3d4", "f-b2c3d4e5"] {
        board.issues.push(Issue {
            id: id.to_owned(),
            title: id.to_owned(),
            status: "open".to_owned(),
            priority: 2,
            kind: "task".to_owned(),
            assignee: None,
            updated_at: None,
            description: None,
            acceptance_criteria: None,
        });
    }
    // Full f-IDs round-trip exactly.
    assert_eq!(
        board.resolve_issue_id("f-a1b2c3d4").as_deref(),
        Some("f-a1b2c3d4")
    );
    // Short hex prefixes resolve (fuzzy fallback).
    assert_eq!(
        board.resolve_issue_id("f-a1b2").as_deref(),
        Some("f-a1b2c3d4")
    );
    assert_eq!(
        board.resolve_issue_id("F-B2C3").as_deref(),
        Some("f-b2c3d4e5")
    );
    assert_eq!(board.resolve_issue_id("  ").as_deref(), None);
    assert_eq!(board.resolve_issue_id("f-zzzz").as_deref(), None);
}
