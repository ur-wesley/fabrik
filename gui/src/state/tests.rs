use std::collections::HashSet;
use std::path::PathBuf;
use std::time::{Duration, Instant};

use chrono::Utc;

use crate::fabrik::GraphStatus;
use crate::model::{Action, Screen, Snapshot};

use super::AppState;

fn tab_store(paths: &[&str], active: &str) -> AppState {
    let mut state = AppState::new();
    state.workspace.open_tabs = paths.iter().map(PathBuf::from).collect();
    state.workspace.repo = Some(PathBuf::from(active));
    state.workspace.closed_tab_stack = Vec::new();
    state
}

#[test]
fn revision_bumps_on_visible_mutations() {
    let mut state = AppState::new();
    let rev0 = state.revision;
    state.reduce(Action::SetSearch("foo".to_owned()));
    assert!(state.revision > rev0);
}

#[test]
fn next_tab_wraps_sequentially() {
    let mut state = tab_store(&["a", "b", "c"], "c");
    state.reduce(Action::NextTab);
    assert_eq!(state.workspace.repo, Some(PathBuf::from("a")));
    state.reduce(Action::NextTab);
    assert_eq!(state.workspace.repo, Some(PathBuf::from("b")));
}

#[test]
fn prev_tab_wraps_left() {
    let mut state = tab_store(&["a", "b", "c"], "a");
    state.reduce(Action::PrevTab);
    assert_eq!(state.workspace.repo, Some(PathBuf::from("c")));
    state.reduce(Action::PrevTab);
    assert_eq!(state.workspace.repo, Some(PathBuf::from("b")));
}

#[test]
fn select_tab_picks_nth_and_clamps_slot_8_to_last() {
    let mut state = tab_store(
        &["t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8", "t9"],
        "t1",
    );
    state.reduce(Action::SelectTab(3));
    assert_eq!(state.workspace.repo, Some(PathBuf::from("t3")));
    state.reduce(Action::SelectTab(8));
    assert_eq!(state.workspace.repo, Some(PathBuf::from("t9")));
    state.reduce(Action::SelectTab(7));
    assert_eq!(state.workspace.repo, Some(PathBuf::from("t7")));
}

#[test]
fn select_tab_out_of_range_is_noop() {
    let mut state = tab_store(&["a", "b"], "a");
    state.reduce(Action::SelectTab(5));
    assert_eq!(state.workspace.repo, Some(PathBuf::from("a")));
    state.reduce(Action::SelectTab(0));
    assert_eq!(state.workspace.repo, Some(PathBuf::from("a")));
    state.reduce(Action::SelectTab(9));
    assert_eq!(state.workspace.repo, Some(PathBuf::from("a")));
}

#[test]
fn close_pushes_stack_and_reopen_restores_position() {
    let mut state = tab_store(&["a", "b", "c"], "b");
    state.reduce(Action::CloseTab(PathBuf::from("b")));
    assert_eq!(state.workspace.closed_tab_stack.len(), 1);
    assert_eq!(
        state.workspace.closed_tab_stack.first().map(|c| c.index),
        Some(1)
    );
    assert_eq!(state.workspace.repo, Some(PathBuf::from("c")));
    state.reduce(Action::ReopenTab);
    assert_eq!(
        state.workspace.open_tabs,
        vec![PathBuf::from("a"), PathBuf::from("b"), PathBuf::from("c"),]
    );
    assert_eq!(state.workspace.repo, Some(PathBuf::from("b")));
    assert_eq!(
        state.workspace.closed_tab_stack,
        Vec::<crate::model::ClosedTab>::new()
    );
}

#[test]
fn reopen_empty_stack_is_noop() {
    let mut state = tab_store(&["a"], "a");
    let rev = state.revision;
    state.reduce(Action::ReopenTab);
    assert_eq!(state.workspace.open_tabs, vec![PathBuf::from("a")]);
    assert_eq!(state.workspace.repo, Some(PathBuf::from("a")));
    assert_eq!(state.revision, rev);
}

#[test]
fn picker_toggle_open_close() {
    let mut state = tab_store(&["a"], "a");
    assert!(!state.workspace.picker_open);
    state.reduce(Action::TogglePicker);
    assert!(state.workspace.picker_open);
    state.reduce(Action::TogglePicker);
    assert!(!state.workspace.picker_open);
    state.reduce(Action::TogglePicker);
    assert!(state.workspace.picker_open);
    state.reduce(Action::ClosePicker);
    assert!(!state.workspace.picker_open);
}

#[test]
fn picker_query_resets_selection_and_move_saturates() {
    let mut state = tab_store(&["a"], "a");
    state.reduce(Action::SetPickerQuery("neon".to_owned()));
    assert_eq!(state.workspace.picker_query, "neon");
    assert_eq!(state.workspace.picker_selected, 0);
    state.reduce(Action::MovePicker(3));
    assert_eq!(state.workspace.picker_selected, 3);
    state.reduce(Action::MovePicker(-10));
    assert_eq!(state.workspace.picker_selected, 0);
    state.reduce(Action::SetPickerQuery("other".to_owned()));
    assert_eq!(state.workspace.picker_selected, 0);
}

#[test]
fn picker_open_resets_query() {
    let mut state = tab_store(&["a"], "a");
    state.reduce(Action::SetPickerQuery("stale".to_owned()));
    state.reduce(Action::MovePicker(2));
    state.reduce(Action::TogglePicker);
    assert!(state.workspace.picker_open);
    assert_eq!(state.workspace.picker_query, String::new());
    assert_eq!(state.workspace.picker_selected, 0);
    state.reduce(Action::ClosePicker);
    assert!(!state.workspace.picker_open);
}

#[test]
fn comment_post_empty_sets_error() {
    let _guard = crate::cache::TEST_ENV_LOCK
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    let mut state = tab_store(&["a", "b"], "b");
    state.reduce(Action::OpenDetail("f-a1b2c3d4".to_owned()));
    state.reduce(Action::PostComment);
    assert!(!state.board.comment_posting);
    assert!(state.board.comment_error.is_some());
}

#[test]
fn comment_failed_clears_spinner_and_sets_composer_error() {
    let _guard = crate::cache::TEST_ENV_LOCK
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    let mut state = tab_store(&["a", "b"], "b");
    state.reduce(Action::OpenDetail("f-a1b2c3d4".to_owned()));
    state.board.comment_draft = "hello".to_owned();
    state.reduce(Action::PostComment);
    assert!(state.board.comment_posting);
    state.reduce(Action::Failed(
        Some(PathBuf::from("b")),
        "fabrik exited 1: boom".to_owned(),
    ));
    assert!(!state.board.comment_posting);
    assert_eq!(
        state.board.comment_error.as_deref(),
        Some("fabrik exited 1: boom")
    );
    assert_eq!(state.board.comment_draft, "hello");
}

#[test]
fn live_event_claim_toast_and_refresh_followup() {
    let mut state = AppState::new();
    let followups = state.reduce(Action::LiveEvent {
        topic: "issues".to_owned(),
        id: Some("f-abc".to_owned()),
        op: Some("claim".to_owned()),
    });
    assert_eq!(followups.len(), 1);
    assert!(matches!(followups[0], Action::Refresh));
    assert_eq!(state.toasts.items().len(), 1);
    assert_eq!(state.toasts.items()[0].message, "Issue f-abc claimed");
}

#[test]
fn duplicate_live_info_toast_does_not_stack() {
    let mut state = AppState::new();
    state.reduce(Action::LiveEvent {
        topic: "issues".to_owned(),
        id: None,
        op: None,
    });
    state.reduce(Action::LiveEvent {
        topic: "issues".to_owned(),
        id: None,
        op: None,
    });
    assert_eq!(state.toasts.items().len(), 1);
}

#[test]
fn failed_sets_error_toast_cleared_by_snapshot() {
    let mut state = tab_store(&["b"], "b");
    state.reduce(Action::Failed(Some(PathBuf::from("b")), "boom".to_owned()));
    assert_eq!(state.toasts.items().len(), 1);
    let snapshot = Snapshot {
        repo: PathBuf::from("b"),
        issues: vec![],
        ready_ids: HashSet::new(),
        fetched_at: Utc::now(),
    };
    state.reduce(Action::Snapshot(snapshot));
    assert!(state.toasts.items().is_empty());
}

#[test]
fn listen_unavailable_same_message_no_second_toast() {
    let mut state = AppState::new();
    state.workspace.listen_live = true;
    state.reduce(Action::ListenUnavailable("err".to_owned()));
    assert_eq!(state.toasts.items().len(), 1);
    state.reduce(Action::ListenUnavailable("err".to_owned()));
    assert_eq!(state.toasts.items().len(), 1);
}

fn lookup_ready_store() -> AppState {
    let mut state = tab_store(&["r"], "r");
    state.workspace.repo_initialized = true;
    state
}

#[test]
fn run_graph_build_sets_busy_when_initialized() {
    let mut state = lookup_ready_store();
    state.reduce(Action::RunGraphBuild);
    assert!(state.lookup.lookup_build_in_progress);
}

#[test]
fn run_graph_build_noop_without_init() {
    let mut state = tab_store(&["r"], "r");
    state.workspace.repo_initialized = false;
    state.reduce(Action::RunGraphBuild);
    assert!(!state.lookup.lookup_build_in_progress);
}

#[test]
fn graph_status_loaded_drives_build_banner_flags() {
    let mut state = lookup_ready_store();
    assert!(state.lookup.needs_graph_build());
    state.reduce(Action::GraphStatusLoaded(GraphStatus {
        indexed: false,
        files: 0,
        symbols: 0,
        stale_files: 0,
        last_build_at: String::new(),
    }));
    assert!(state.lookup.needs_graph_build());
    assert!(!state.lookup.graph_stale());

    state.reduce(Action::GraphStatusLoaded(GraphStatus {
        indexed: true,
        files: 10,
        symbols: 100,
        stale_files: 3,
        last_build_at: String::new(),
    }));
    assert!(!state.lookup.needs_graph_build());
    assert!(state.lookup.graph_stale());
}

#[test]
fn graph_build_done_requests_lookup_refresh() {
    let mut state = lookup_ready_store();
    state.lookup.lookup_build_in_progress = true;
    let followups = state.reduce(Action::GraphBuildDone(PathBuf::from("r")));
    assert!(!state.lookup.lookup_build_in_progress);
    assert_eq!(followups.len(), 1);
    assert!(matches!(followups[0], Action::RefreshLookup));
}

#[test]
fn set_screen_lookup_refreshes_with_empty_query() {
    let mut state = lookup_ready_store();
    let followups = state.reduce(Action::SetScreen(Screen::Lookup));
    assert_eq!(followups.len(), 1);
    assert!(matches!(followups[0], Action::RefreshLookup));
    assert!(state.lookup.lookup_status_loading);
}

#[test]
fn tick_removes_expired_info_toast() {
    let mut state = AppState::new();
    let now = Instant::now();
    state.toasts.push_info(
        "gone".to_owned(),
        now.checked_sub(Duration::from_secs(10))
            .expect("recent instant"),
    );
    state.toasts.tick(now);
    assert!(state.toasts.items().is_empty());
}

fn write_ready_store() -> AppState {
    let mut state = tab_store(&["r"], "r");
    state.workspace.repo_initialized = true;
    state
}

#[test]
fn submit_create_empty_title_sets_error_without_busy() {
    let mut state = write_ready_store();
    state.reduce(Action::OpenCreate);
    assert!(state.board.create_open);
    state.reduce(Action::SubmitCreate);
    assert!(!state.board.write_busy);
    assert_eq!(state.board.write_error.as_deref(), Some("Title is empty"));
}

#[test]
fn submit_create_needs_initialized_repo() {
    let mut state = tab_store(&["r"], "r");
    state.workspace.repo_initialized = false;
    state.reduce(Action::SetCreateTitle("hello".to_owned()));
    state.reduce(Action::SubmitCreate);
    assert!(!state.board.write_busy);
    assert!(state.board.write_error.is_some());
}

#[test]
fn submit_create_valid_sets_busy() {
    let mut state = write_ready_store();
    state.reduce(Action::SetCreateTitle("hello".to_owned()));
    state.reduce(Action::SetCreatePriority(9));
    assert_eq!(state.board.create_priority, 4);
    state.reduce(Action::SubmitCreate);
    assert!(state.board.write_busy);
    assert!(state.board.write_error.is_none());
}

#[test]
fn write_ok_toasts_and_refreshes() {
    let mut state = write_ready_store();
    state.board.write_busy = true;
    state.board.create_open = true;
    let followups = state.reduce(Action::WriteOk("Issue f-a1b2c3d4 created".to_owned()));
    assert!(!state.board.write_busy);
    assert!(!state.board.create_open);
    assert!(state.board.create_title.is_empty());
    assert_eq!(followups.len(), 1);
    assert!(matches!(followups[0], Action::Refresh));
    assert_eq!(state.toasts.items().len(), 1);
    assert_eq!(state.toasts.items()[0].message, "Issue f-a1b2c3d4 created");
}

#[test]
fn claim_invalid_id_sets_error_without_busy() {
    let mut state = write_ready_store();
    state.reduce(Action::ClaimIssue("bd-1".to_owned()));
    assert!(!state.board.write_busy);
    assert!(state.board.write_error.is_some());
    state.reduce(Action::ClaimIssue("f-a1b2c3d4".to_owned()));
    assert!(state.board.write_busy);
}

#[test]
fn toggle_write_force_flips() {
    let mut state = write_ready_store();
    assert!(!state.board.write_force);
    state.reduce(Action::ToggleWriteForce);
    assert!(state.board.write_force);
}

#[test]
fn submit_dep_add_invalid_blocker_sets_error() {
    let mut state = write_ready_store();
    state.board.detail_id = Some("f-a1b2c3d4".to_owned());
    state.reduce(Action::SetDepTarget("nope".to_owned()));
    state.reduce(Action::SubmitDepAdd);
    assert!(!state.board.write_busy);
    assert!(state.board.write_error.is_some());
    state.reduce(Action::SetDepTarget("f-b2c3d4e5".to_owned()));
    state.reduce(Action::SubmitDepAdd);
    assert!(state.board.write_busy);
}

#[test]
fn failed_during_write_clears_busy_and_sets_form_errors() {
    let mut state = write_ready_store();
    state.board.write_busy = true;
    state.memory.mem_write_busy = true;
    state.reduce(Action::Failed(Some(PathBuf::from("r")), "boom".to_owned()));
    assert!(!state.board.write_busy);
    assert_eq!(state.board.write_error.as_deref(), Some("boom"));
    assert!(!state.memory.mem_write_busy);
    assert_eq!(state.memory.mem_write_error.as_deref(), Some("boom"));
}

#[test]
fn mem_save_empty_title_sets_error() {
    let mut state = write_ready_store();
    state.reduce(Action::OpenMemForm);
    assert!(state.memory.mem_form_open);
    state.reduce(Action::SubmitMemSave);
    assert!(!state.memory.mem_write_busy);
    assert!(state.memory.mem_write_error.is_some());
}

#[test]
fn mem_save_valid_sets_busy() {
    let mut state = write_ready_store();
    state.reduce(Action::OpenMemForm);
    state.reduce(Action::SetMemTitle("t".to_owned()));
    state.reduce(Action::SetMemBody("b".to_owned()));
    state.reduce(Action::SubmitMemSave);
    assert!(state.memory.mem_write_busy);
    assert!(state.memory.mem_write_error.is_none());
}

#[test]
fn mem_write_ok_closes_forms_and_refreshes() {
    let mut state = write_ready_store();
    state.memory.mem_form_open = true;
    state.memory.mem_prompt_open = true;
    state.memory.mem_write_busy = true;
    let followups = state.reduce(Action::MemWriteOk("Observation o-abc saved".to_owned()));
    assert!(!state.memory.mem_write_busy);
    assert!(!state.memory.mem_form_open);
    assert!(!state.memory.mem_prompt_open);
    assert_eq!(followups.len(), 1);
    assert!(matches!(followups[0], Action::RefreshMemory));
    assert_eq!(state.toasts.items()[0].message, "Observation o-abc saved");
}

#[test]
fn mem_delete_confirm_flow_refreshes_and_clears_detail() {
    let mut state = write_ready_store();
    state.reduce(Action::RequestMemDelete("o-abc123".to_owned()));
    assert_eq!(state.memory.mem_delete_confirm.as_deref(), Some("o-abc123"));
    state.reduce(Action::ConfirmMemDelete);
    assert!(state.memory.mem_write_busy);
    state.memory.mem_detail_id = Some("o-abc123".to_owned());
    let followups = state.reduce(Action::MemDeleted("o-abc123".to_owned()));
    assert!(!state.memory.mem_write_busy);
    assert!(state.memory.mem_delete_confirm.is_none());
    assert!(state.memory.mem_detail_id.is_none());
    assert_eq!(followups.len(), 1);
    assert!(matches!(followups[0], Action::RefreshMemory));
}

#[test]
fn mem_delete_rejects_bad_id_without_confirm() {
    let mut state = write_ready_store();
    state.reduce(Action::RequestMemDelete("358".to_owned()));
    assert!(state.memory.mem_delete_confirm.is_none());
    state.reduce(Action::CancelMemDelete);
    assert!(state.memory.mem_delete_confirm.is_none());
}

#[test]
fn mem_edit_prefills_form_from_results() {
    let mut state = write_ready_store();
    state.memory.mem_results.push(crate::fabrik::Observation {
        id: "o-abc123".to_owned(),
        session_id: None,
        project: "p".to_owned(),
        kind: "decision".to_owned(),
        title: "t".to_owned(),
        body: "b".to_owned(),
        topic_key: Some("k".to_owned()),
        symbol_id: None,
        created_at: String::new(),
    });
    state.reduce(Action::OpenMemEdit("o-abc123".to_owned()));
    assert!(state.memory.mem_form_open);
    assert_eq!(state.memory.mem_editing.as_deref(), Some("o-abc123"));
    assert_eq!(state.memory.mem_form_title, "t");
    assert_eq!(state.memory.mem_form_topic, "k");
    state.reduce(Action::SubmitMemUpdate);
    assert!(state.memory.mem_write_busy);
}

#[test]
fn mem_prompt_empty_capture_sets_error() {
    let mut state = write_ready_store();
    state.reduce(Action::OpenMemPrompt);
    assert!(state.memory.mem_prompt_open);
    state.reduce(Action::SubmitMemPrompt);
    assert!(!state.memory.mem_write_busy);
    assert!(state.memory.mem_write_error.is_some());
    state.reduce(Action::CloseMemPrompt);
    assert!(!state.memory.mem_prompt_open);
}
