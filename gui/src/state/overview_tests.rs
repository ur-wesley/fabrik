//! Instant graph overview (`graph overview --json`) auto-load and cache.
//! Split from `tests` so both files stay under the 500-line cap.
use std::collections::HashSet;
use std::path::PathBuf;

use chrono::Utc;

use crate::fabrik::{GraphNode, GraphStatus, QueryResult};
use crate::lookup::is_unknown_command;
use crate::model::{Action, Snapshot};

use super::AppState;

fn lookup_ready_store() -> AppState {
    let mut state = AppState::new();
    state.workspace.open_tabs = vec![PathBuf::from("r")];
    state.workspace.repo = Some(PathBuf::from("r"));
    state.workspace.closed_tab_stack = Vec::new();
    state.workspace.repo_initialized = true;
    state
}

fn fresh_indexed_status() -> GraphStatus {
    GraphStatus {
        indexed: true,
        files: 10,
        symbols: 100,
        stale_files: 0,
        last_build_at: String::new(),
    }
}

fn overview_result(ids: &[&str]) -> QueryResult {
    QueryResult {
        nodes: ids
            .iter()
            .map(|id| GraphNode {
                id: (*id).to_owned(),
                kind: "symbol".to_owned(),
                name: (*id).to_owned(),
                path: "a.go".to_owned(),
                line: 1,
                lang: "go".to_owned(),
            })
            .collect(),
        edges: Vec::new(),
    }
}

#[test]
fn overview_auto_loads_once_on_fresh_status_with_empty_query() {
    let mut state = lookup_ready_store();
    let followups = state.reduce(Action::GraphStatusLoaded(fresh_indexed_status()));
    assert_eq!(followups.len(), 1);
    assert!(matches!(
        followups[0],
        Action::FetchGraphOverview { budget: 60, .. }
    ));
    let repo = PathBuf::from("r");
    state.reduce(Action::FetchGraphOverview {
        repo: repo.clone(),
        budget: 60,
    });
    assert!(state.lookup.lookup_loading);
    state.reduce(Action::LookupOverviewLoaded {
        repo: repo.clone(),
        result: overview_result(&["sym:a", "sym:b"]),
    });
    assert_eq!(state.lookup.lookup_nodes.len(), 2);
    assert!(!state.lookup.lookup_loading);
    assert_eq!(state.lookup.overview_loaded_for, Some(repo));
    // Second status event must not refetch.
    let followups = state.reduce(Action::GraphStatusLoaded(fresh_indexed_status()));
    assert!(followups.is_empty());
}

#[test]
fn overview_skipped_when_query_anchor_or_stale() {
    let mut state = lookup_ready_store();
    state.reduce(Action::SetLookupQuery("main".to_owned()));
    let followups = state.reduce(Action::GraphStatusLoaded(fresh_indexed_status()));
    assert!(followups.is_empty());

    let mut state = lookup_ready_store();
    state.reduce(Action::LookupSelectNode(Some("sym:a".to_owned())));
    let followups = state.reduce(Action::GraphStatusLoaded(fresh_indexed_status()));
    assert!(followups.is_empty());

    let mut state = lookup_ready_store();
    let mut stale_status = fresh_indexed_status();
    stale_status.stale_files = 2;
    let followups = state.reduce(Action::GraphStatusLoaded(stale_status));
    assert!(followups.is_empty());

    let mut state = lookup_ready_store();
    state.reduce(Action::GraphStatusLoaded(GraphStatus {
        indexed: false,
        files: 0,
        symbols: 0,
        stale_files: 0,
        last_build_at: String::new(),
    }));
    let followups = state.reduce(Action::GraphStatusLoaded(fresh_indexed_status()));
    // No nodes yet and query empty: this one SHOULD fetch (sanity).
    assert_eq!(followups.len(), 1);
}

#[test]
fn overview_typed_query_replaces_and_clear_restores_cache() {
    let mut state = lookup_ready_store();
    state.reduce(Action::GraphStatusLoaded(fresh_indexed_status()));
    state.reduce(Action::LookupOverviewLoaded {
        repo: PathBuf::from("r"),
        result: overview_result(&["sym:a", "sym:b"]),
    });
    assert_eq!(state.lookup.lookup_nodes.len(), 2);

    state.reduce(Action::SetLookupQuery("main".to_owned()));
    assert!(state.lookup.lookup_nodes.is_empty());
    assert!(state.lookup.lookup_loading);
    // Cache survives the typed query.
    assert!(state.lookup.lookup_overview.is_some());

    state.reduce(Action::SetLookupQuery(String::new()));
    assert_eq!(state.lookup.lookup_nodes.len(), 2);
    assert!(!state.lookup.lookup_loading);
}

#[test]
fn overview_guard_resets_on_repo_switch() {
    let mut state = lookup_ready_store();
    state.reduce(Action::GraphStatusLoaded(fresh_indexed_status()));
    state.reduce(Action::LookupOverviewLoaded {
        repo: PathBuf::from("r"),
        result: overview_result(&["sym:a"]),
    });
    assert!(state.lookup.overview_loaded_for.is_some());
    state.lookup.reset_for_repo();
    assert!(state.lookup.overview_loaded_for.is_none());
    assert!(state.lookup.lookup_overview.is_none());
    assert!(state.lookup.lookup_nodes.is_empty());
}

#[test]
fn overview_failure_sets_persistent_card_with_binary_and_toast() {
    let mut state = lookup_ready_store();
    state.reduce(Action::FetchGraphOverview {
        repo: PathBuf::from("r"),
        budget: 60,
    });
    assert!(state.lookup.lookup_loading);
    state.reduce(Action::LookupOverviewFailed {
        repo: PathBuf::from("r"),
        message: "fabrik exited 1: boom".to_owned(),
    });
    // Loader resets so the pane never sticks on its spinner.
    assert!(!state.lookup.lookup_loading);
    // Card keeps the full message plus the failing binary path.
    let card = state
        .lookup
        .lookup_overview_error
        .as_deref()
        .expect("overview card");
    assert!(card.contains("fabrik exited 1: boom"));
    assert!(card.contains(crate::fabrik::fabrik_bin().as_str()));
    // Toast path unchanged: the failure still toasts, and board state
    // stays untouched (lookup-scoped, not a board error).
    assert_eq!(state.toasts.items().len(), 1);
    assert!(state.board.error.is_none());
}

#[test]
fn overview_failure_card_survives_board_snapshot() {
    let mut state = lookup_ready_store();
    state.reduce(Action::LookupOverviewFailed {
        repo: PathBuf::from("r"),
        message: "fabrik exited 1: boom".to_owned(),
    });
    assert!(state.lookup.lookup_overview_error.is_some());
    state.reduce(Action::Snapshot(Snapshot {
        repo: PathBuf::from("r"),
        issues: vec![],
        ready_ids: HashSet::new(),
        fetched_at: Utc::now(),
    }));
    // Board toasts clear on board success, but the card persists until an
    // overview succeeds: the card is the source of truth.
    assert!(state.toasts.items().is_empty());
    assert!(state.lookup.lookup_overview_error.is_some());
}

#[test]
fn overview_retry_refetches_and_keeps_card_until_success() {
    let mut state = lookup_ready_store();
    state.reduce(Action::LookupOverviewFailed {
        repo: PathBuf::from("r"),
        message: "fabrik exited 1: boom".to_owned(),
    });
    // Retry refetches: the pane spins again while the old card stays up.
    state.reduce(Action::FetchGraphOverview {
        repo: PathBuf::from("r"),
        budget: 60,
    });
    assert!(state.lookup.lookup_loading);
    assert!(state.lookup.lookup_overview_error.is_some());
    // Success clears the card and fills the canvas.
    state.reduce(Action::LookupOverviewLoaded {
        repo: PathBuf::from("r"),
        result: overview_result(&["sym:a"]),
    });
    assert!(state.lookup.lookup_overview_error.is_none());
    assert!(!state.lookup.lookup_loading);
    assert_eq!(state.lookup.lookup_nodes.len(), 1);
}

#[test]
fn overview_failure_stale_repo_is_dropped() {
    let mut state = lookup_ready_store();
    state.reduce(Action::LookupOverviewFailed {
        repo: PathBuf::from("other"),
        message: "fabrik exited 1: boom".to_owned(),
    });
    assert!(state.lookup.lookup_overview_error.is_none());
    assert!(state.toasts.items().is_empty());
}

#[test]
fn board_failed_resets_lookup_loaders() {
    let mut state = lookup_ready_store();
    state.lookup.lookup_loading = true;
    state.lookup.lookup_inspect_loading = true;
    state.reduce(Action::Failed(Some(PathBuf::from("r")), "boom".to_owned()));
    assert!(!state.lookup.lookup_loading);
    assert!(!state.lookup.lookup_inspect_loading);
}

#[test]
fn is_unknown_command_classifies_old_binary() {
    assert!(is_unknown_command(
        "fabrik exited 1: Error: unknown command \"overview\" for \"fabrik graph\""
    ));
    assert!(is_unknown_command("Error: unknown subcommand \"overview\""));
    assert!(is_unknown_command(
        "fabrik exited 1: Error: unknown flag: --budget"
    ));
    // Case-insensitive: wrapped messages still classify.
    assert!(is_unknown_command(
        "FABRIK EXITED 1: UNKNOWN COMMAND \"overview\""
    ));
    assert!(!is_unknown_command("fabrik exited 1: boom"));
    assert!(!is_unknown_command(
        "failed to run fabrik: No such file or directory"
    ));
    assert!(!is_unknown_command(""));
}

#[test]
fn file_anchor_context_impact_never_load_without_busy() {
    // Regression test (mirrors `claim_invalid_id_sets_error_without_busy`):
    // file anchors fail live with `node sym:file:… not found`, so the
    // reducer must stay quiet — no loading flag, no stored symbol, no
    // followups — and the worker (gated in `Effects`) never spawns.
    let mut state = lookup_ready_store();
    let followups = state.reduce(Action::FetchGraphContext("file:cli/go.mod".to_owned()));
    assert!(followups.is_empty());
    assert!(!state.lookup.lookup_inspect_loading);
    assert!(state.lookup.lookup_context_symbol.is_none());
    let followups = state.reduce(Action::FetchGraphImpact {
        symbol: "file:cli/go.mod".to_owned(),
        depth: 3,
    });
    assert!(followups.is_empty());
    assert!(!state.lookup.lookup_inspect_loading);
    assert!(state.lookup.lookup_impact_symbol.is_none());
    // Symbol anchors still arm the inspect fetch as before.
    let followups = state.reduce(Action::FetchGraphContext("sym:go.mod:main".to_owned()));
    assert!(followups.is_empty());
    assert!(state.lookup.lookup_inspect_loading);
    assert_eq!(
        state.lookup.lookup_context_symbol.as_deref(),
        Some("sym:go.mod:main")
    );
}
