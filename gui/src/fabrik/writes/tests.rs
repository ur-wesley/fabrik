use super::{
    DEP_KINDS, ISSUE_KINDS, IssueCreateParams, MEM_KINDS, MemSaveParams, MemUpdateParams, add_dep,
    claim_args, claim_issue, close_issue, create_args, create_issue, delete_observation, dep_args,
    mem_save_args, mem_update_args, normalize_assignee, normalize_dep_kind, normalize_issue_kind,
    normalize_mem_kind, save_observation,
};

#[test]
fn create_args_forward_body_acceptance_priority_kind() {
    let args = create_args(&IssueCreateParams {
        title: "  hello  ".to_owned(),
        body: "b".to_owned(),
        acceptance: "a".to_owned(),
        priority: 1,
        kind: "bug".to_owned(),
    });
    assert_eq!(
        args,
        vec![
            "issue",
            "create",
            "hello",
            "--body",
            "b",
            "--acceptance",
            "a",
            "--priority",
            "1",
            "--kind",
            "bug",
            "--json",
        ]
    );
}

#[test]
fn create_args_omit_empty_body_and_clamp_priority() {
    let args = create_args(&IssueCreateParams {
        title: "t".to_owned(),
        body: "   ".to_owned(),
        acceptance: String::new(),
        priority: 9,
        kind: "nope".to_owned(),
    });
    assert_eq!(
        args,
        vec![
            "issue",
            "create",
            "t",
            "--priority",
            "4",
            "--kind",
            "task",
            "--json",
        ]
    );
}

#[test]
fn claim_close_args_carry_by_reason_force() {
    assert_eq!(
        claim_args("claim", "f-a1b2c3d4", "alice", "", false),
        vec!["issue", "claim", "f-a1b2c3d4", "--by", "alice", "--json"]
    );
    assert_eq!(
        claim_args("unclaim", "f-a1b2c3d4", "fabrik", "", true),
        vec![
            "issue",
            "unclaim",
            "f-a1b2c3d4",
            "--by",
            "fabrik",
            "--force",
            "--json"
        ]
    );
    // Empty `--by` is omitted (callers pass the normalized assignee).
    assert_eq!(
        claim_args("claim", "f-a1b2c3d4", "  ", "", false),
        vec!["issue", "claim", "f-a1b2c3d4", "--json"]
    );
    assert_eq!(
        claim_args("close", "f-a1b2c3d4", "bob", "done", true),
        vec![
            "issue",
            "close",
            "f-a1b2c3d4",
            "--by",
            "bob",
            "--reason",
            "done",
            "--force",
            "--json",
        ]
    );
    // Close without reason omits the flag (CLI stores no note then).
    assert_eq!(
        claim_args("close", "f-a1b2c3d4", "bob", "  ", false),
        vec!["issue", "close", "f-a1b2c3d4", "--by", "bob", "--json"]
    );
}

#[test]
fn dep_args_normalize_kind() {
    assert_eq!(
        dep_args("f-a1b2c3d4", "f-b2c3d4e5", "relates"),
        vec![
            "issue",
            "dep",
            "f-a1b2c3d4",
            "f-b2c3d4e5",
            "--kind",
            "relates",
            "--json",
        ]
    );
    assert_eq!(
        dep_args("f-a1b2c3d4", "f-b2c3d4e5", "bogus"),
        vec![
            "issue",
            "dep",
            "f-a1b2c3d4",
            "f-b2c3d4e5",
            "--kind",
            "blocks",
            "--json",
        ]
    );
}

#[test]
fn mem_save_args_forward_all_optional_fields() {
    let args = mem_save_args(&MemSaveParams {
        title: " t ".to_owned(),
        body: "b".to_owned(),
        kind: "decision".to_owned(),
        topic_key: Some("auth/model".to_owned()),
        symbol_id: Some("sym:main".to_owned()),
    });
    assert_eq!(
        args,
        vec![
            "mem",
            "save",
            "--title",
            "t",
            "--body",
            "b",
            "--kind",
            "decision",
            "--topic-key",
            "auth/model",
            "--symbol-id",
            "sym:main",
            "--json",
        ]
    );
    let minimal = mem_save_args(&MemSaveParams {
        title: "t".to_owned(),
        body: "b".to_owned(),
        kind: String::new(),
        topic_key: None,
        symbol_id: Some("  ".to_owned()),
    });
    assert_eq!(
        minimal,
        vec![
            "mem",
            "save",
            "--title",
            "t",
            "--body",
            "b",
            "--kind",
            "discovery",
            "--json",
        ]
    );
}

#[test]
fn mem_save_args_preserves_prompt_kind() {
    let args = mem_save_args(&MemSaveParams {
        title: "t".to_owned(),
        body: "b".to_owned(),
        kind: "prompt".to_owned(),
        topic_key: None,
        symbol_id: None,
    });
    assert_eq!(
        args,
        vec![
            "mem", "save", "--title", "t", "--body", "b", "--kind", "prompt", "--json",
        ]
    );
}

#[test]
fn mem_update_args_only_send_set_fields() {
    let args = mem_update_args(&MemUpdateParams {
        id: "o-abc123".to_owned(),
        title: Some("new".to_owned()),
        body: None,
        kind: Some(String::new()),
        topic_key: Some("k".to_owned()),
        symbol_id: None,
    });
    assert_eq!(
        args,
        vec![
            "mem",
            "update",
            "--id",
            "o-abc123",
            "--title",
            "new",
            "--topic-key",
            "k",
            "--json",
        ]
    );
}

#[test]
fn kind_normalizers_fall_back_to_defaults() {
    assert_eq!(normalize_issue_kind("bug"), "bug");
    assert_eq!(normalize_issue_kind("bogus"), "task");
    assert_eq!(normalize_mem_kind("handoff"), "handoff");
    assert_eq!(normalize_mem_kind("prompt"), "prompt");
    assert_eq!(normalize_mem_kind("  prompt  "), "prompt");
    assert_eq!(normalize_mem_kind(""), "discovery");
    assert_eq!(normalize_dep_kind("relates"), "relates");
    assert_eq!(normalize_dep_kind("x"), "blocks");
    assert_eq!(normalize_assignee("  "), "fabrik");
    assert_eq!(normalize_assignee("alice"), "alice");
    assert_eq!(ISSUE_KINDS.len(), 5);
    assert_eq!(MEM_KINDS.len(), 6);
    assert_eq!(DEP_KINDS.len(), 2);
}

#[test]
fn write_validators_reject_bad_ids_without_spawning() {
    let repo = std::path::Path::new(".");
    for bad in ["", "bd-1", "f-XYZ", "f-short"] {
        assert!(claim_issue(repo, bad, "fabrik").is_err());
        assert!(close_issue(repo, bad, "r", "fabrik", false).is_err());
        assert!(add_dep(repo, bad, "f-a1b2c3d4", "blocks").is_err());
        assert!(add_dep(repo, "f-a1b2c3d4", bad, "blocks").is_err());
    }
    assert!(
        create_issue(
            repo,
            &IssueCreateParams {
                title: "   ".to_owned(),
                body: String::new(),
                acceptance: String::new(),
                priority: 2,
                kind: "task".to_owned(),
            }
        )
        .is_err()
    );
    assert!(
        save_observation(
            None,
            &MemSaveParams {
                title: String::new(),
                body: "b".to_owned(),
                kind: "discovery".to_owned(),
                topic_key: None,
                symbol_id: None,
            }
        )
        .is_err()
    );
    assert!(delete_observation("358").is_err());
    assert!(delete_observation("f-a1b2c3d4").is_err());
}

#[test]
fn parse_live_issue_write_shape_into_detail() {
    // `issue create/claim/close/dep --json` encode the same Issue
    // struct as `issue show --json`.
    let raw = r#"{"id":"f-a1b2c3d4","title":"t","body":"b","acceptance":"a","status":"active","priority":1,"kind":"bug","assignee":"alice","created_at":"2026-10-14T08:00:00Z","updated_at":"2026-10-14T09:00:00Z","started_at":"2026-10-14T08:30:00Z","closed_at":null,"ready":false,"blockers":[{"id":"f-b2c3d4e5","title":"blocker","status":"open","priority":0,"kind":"task"}],"blocking":[],"notes":[{"id":"n-a1b2c3d4","issue_id":"f-a1b2c3d4","body":"note","created_at":"2026-10-14T09:00:00Z"}]}"#;
    let item: crate::model::ShowIssueJson = serde_json::from_str(raw).expect("valid");
    let detail = item.into_detail();
    assert_eq!(detail.assignee.as_deref(), Some("alice"));
    assert_eq!(detail.dependencies.len(), 1);
    assert_eq!(detail.comment_count, 1);
}

#[test]
fn parse_live_mem_save_shape() {
    let raw = r#"{"id":"o-8eee7c16","session_id":"s","project":"C:\\work","kind":"decision","title":"t","body":"b","topic_key":"auth/model","symbol_id":"sym:main","created_at":"2026-10-14T08:47:29Z"}"#;
    let obs: crate::fabrik::Observation = serde_json::from_str(raw).expect("valid");
    assert_eq!(obs.topic_key.as_deref(), Some("auth/model"));
}
