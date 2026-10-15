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
        blast_radius: None,
        candidates: Vec::new(),
    };
    assert!(nodes_from_explore(&explore).is_empty());
}

#[test]
fn mem_get_rejects_non_observation_ids_without_spawning() {
    // Legacy ID split: context/search tolerate legacy numeric ids (see
    // `parse_live_mem_context_shape` with `"358"`), but get is strict
    // `o-<hex>` and must fail client-side without spawning.
    for bad in ["358", "f-a1b2c3d4", "", "o-", "o-XYZ"] {
        assert!(fetch_mem_get(bad).is_err(), "{bad:?} must fail validation");
    }
}

#[test]
fn parse_live_mem_context_shape() {
    // Captured from `mem context --json`: legacy numeric ids (tolerant
    // here for pre-hash DBs — strict `o-<hex>` applies only to
    // get/timeline), missing topic_key/symbol_id, non-RFC3339 sqlite
    // timestamps, nullable session.
    let raw = r#"{"session":{"id":"ses_abc","actor":"default","started_at":"2026-10-14 07:09:53.013701000","ended_at":null,"goal":"","discoveries":"","accomplished":"","next_steps":"","files":""},"observations":[{"id":"358","session_id":"manual-save-fabrik","project":"fabrik","kind":"handoff","title":"summary","body":"text","created_at":"2026-10-01 14:12:50"}],"stats":{"observation_count":358,"session_count":12,"projects":["fabrik"]}}"#;
    let ctx: MemContext = serde_json::from_str(raw).expect("valid");
    assert_eq!(ctx.observations.len(), 1);
    assert_eq!(ctx.observations[0].topic_key, None);
    assert_eq!(ctx.stats.projects, vec!["fabrik".to_owned()]);
    assert!(ctx.session.is_some());
}

#[test]
fn parse_mem_context_without_session() {
    let raw = r#"{"session":null,"observations":[],"stats":{"observation_count":0,"session_count":0,"projects":[]}}"#;
    let ctx: MemContext = serde_json::from_str(raw).expect("valid");
    assert!(ctx.session.is_none());
    assert!(ctx.observations.is_empty());
}

#[test]
fn parse_live_mem_search_and_get_shapes() {
    // `mem search --json` (array, topic_key/symbol_id omitted when
    // unset) and `mem get --json` (object, fields present).
    let search = r#"[{"id":"o-8eee7c16","session_id":"s","project":"","kind":"discovery","title":"t","body":"b","created_at":"2026-10-14T08:47:29Z"}]"#;
    let rows: Vec<Observation> = serde_json::from_str(search).expect("valid");
    assert_eq!(rows.len(), 1);
    assert_eq!(rows[0].symbol_id, None);
    let get = r#"{"id":"o-8eee7c16","session_id":"s","project":"C:\\work","kind":"decision","title":"t","body":"b","topic_key":"auth/model","symbol_id":"sym:main","created_at":"2026-10-14T08:47:29Z"}"#;
    let obs: Observation = serde_json::from_str(get).expect("valid");
    assert_eq!(obs.topic_key.as_deref(), Some("auth/model"));
    assert_eq!(obs.symbol_id.as_deref(), Some("sym:main"));
}

#[test]
fn parse_live_graph_status_shapes() {
    // `graph status --json`: counters are omitempty (absent when zero).
    let cold: GraphStatus = serde_json::from_str(r#"{"indexed":false}"#).expect("valid");
    assert!(!cold.indexed);
    let warm: GraphStatus = serde_json::from_str(
        r#"{"indexed":true,"files":1,"last_build_at":"2026-10-14T08:47:56Z"}"#,
    )
    .expect("valid");
    assert!(warm.indexed);
    assert_eq!(warm.files, 1);
    assert_eq!(warm.symbols, 0);
    assert_eq!(warm.stale_files, 0);
}

#[test]
fn parse_live_graph_explore_shape() {
    // Captured from `graph explore --json`: files/paths/candidates
    // omitted when empty, blast_radius.callers explicit null.
    let raw = r#"{"indexed":true,"blast_radius":{"callers":null,"dependents_count":0}}"#;
    let explore: ExploreResult = serde_json::from_str(raw).expect("valid");
    assert!(explore.indexed);
    assert!(explore.files.is_empty());
    assert!(nodes_from_explore(&explore).is_empty());
}

#[test]
fn parse_live_graph_query_and_explain_shapes() {
    // Captured from `graph query --json` / `graph explain --json`:
    // edges carry confidence, explain carries truncated.
    let query = r#"{"nodes":[{"id":"file:main.go","kind":"file","name":"main.go","path":"main.go","line":0,"lang":"go"}],"edges":[{"src":"file:main.go","dst":"","kind":"calls","label":"main","confidence":"inferred"}]}"#;
    let result: QueryResult = serde_json::from_str(query).expect("valid");
    assert_eq!(result.nodes.len(), 1);
    assert_eq!(result.edges[0].confidence, "inferred");
    let explain = r#"{"node":{"id":"file:main.go","kind":"file","name":"main.go","path":"main.go","line":0,"lang":"go"},"edges":[],"truncated":false}"#;
    let explained: ExplainResult = serde_json::from_str(explain).expect("valid");
    assert_eq!(explained.node.id, "file:main.go");
    assert!(!explained.truncated);
    let path = r#"{"nodes":[],"edges":[]}"#;
    let path_result: PathResult = serde_json::from_str(path).expect("valid");
    assert!(path_result.nodes.is_empty());
}

#[test]
fn parse_live_graph_overview_shape() {
    // `graph overview --json` reuses the QueryResult shape: degree-ordered
    // symbol nodes plus edges among the chosen set.
    let raw = r#"{"nodes":[{"id":"sym:a.go:Alpha","kind":"symbol","name":"Alpha","path":"a.go","line":1,"lang":"go"},{"id":"sym:a.go:Beta","kind":"symbol","name":"Beta","path":"a.go","line":2,"lang":"go"}],"edges":[{"src":"sym:a.go:Alpha","dst":"sym:a.go:Beta","kind":"calls","label":"","confidence":"extracted"}]}"#;
    let result: QueryResult = serde_json::from_str(raw).expect("valid");
    assert_eq!(result.nodes.len(), 2);
    assert_eq!(result.nodes[0].id, "sym:a.go:Alpha");
    assert_eq!(result.edges.len(), 1);
    assert_eq!(result.edges[0].kind, "calls");
}

#[test]
fn parse_graph_build_shape() {
    let raw = r#"{"files":1,"parsed":1,"skipped":0,"nodes":1,"edges":2}"#;
    let built: GraphBuildResult = serde_json::from_str(raw).expect("valid");
    assert_eq!((built.files, built.parsed, built.nodes), (1, 1, 1));
}

#[test]
fn parse_init_output_finds_last_json_line() {
    // `init --json` prints human lines first, the result object last.
    let raw = "Not a git repository (no .git found), continuing\ndolt /tmp/x/.fabrik/dolt\n{\"repo\":\"/tmp/x\",\"dolt\":\"/tmp/x/.fabrik/dolt\",\"graph\":\"/tmp/x/.fabrik/graph.db\",\"created\":true}\n";
    let result = parse_init_output(raw).expect("valid");
    assert_eq!(
        result,
        InitResult {
            repo: "/tmp/x".to_owned(),
            dolt: "/tmp/x/.fabrik/dolt".to_owned(),
            graph: "/tmp/x/.fabrik/graph.db".to_owned(),
            created: true,
        }
    );
    assert!(parse_init_output("no json here\n").is_err());
    assert!(parse_init_output("").is_err());
}

#[test]
fn parse_live_graph_context_shape() {
    let raw = r#"{"symbol":{"id":"sym:main","kind":"symbol","name":"main","path":"main.go","line":10,"lang":"go"},"callers":[{"id":"sym:other","kind":"symbol","name":"other","path":"other.go","line":3,"lang":"go"}],"callees":[{"src":"sym:main","dst":"sym:help","kind":"calls","label":"help","confidence":"extracted"}],"observations":[{"id":"o-abc","title":"t","kind":"decision"}],"tests":[{"path":"main_test.go","symbol":"TestMain"}]}"#;
    let ctx: GraphContextResult = serde_json::from_str(raw).expect("valid");
    assert_eq!(ctx.symbol.id, "sym:main");
    assert_eq!(ctx.callers.len(), 1);
    assert_eq!(ctx.callees.len(), 1);
    assert_eq!(ctx.observations.len(), 1);
    assert_eq!(ctx.tests.len(), 1);
}

#[test]
fn parse_live_graph_impact_shape() {
    let raw = r#"{"root":"sym:main","by_depth":{"1":[{"id":"sym:a","kind":"symbol","name":"a","path":"a.go","line":1,"lang":"go"}],"2":[]},"total":1}"#;
    let impact: GraphImpactResult = serde_json::from_str(raw).expect("valid");
    assert_eq!(impact.root, "sym:main");
    assert_eq!(impact.total, 1);
    assert_eq!(impact.flattened().len(), 1);
    let empty: GraphImpactResult = serde_json::from_str(r#"{"root":"x"}"#).expect("valid");
    assert!(empty.flattened().is_empty());
}

#[test]
fn parse_live_detect_changes_shape() {
    let raw = r#"{"changed_files":["a.go"],"touched_symbols":["sym:a"],"impacted_symbols":["sym:b"],"suggested_checks":["go test ./..."]}"#;
    let detect: DetectChangesResult = serde_json::from_str(raw).expect("valid");
    assert_eq!(detect.changed_files, vec!["a.go".to_owned()]);
    assert_eq!(detect.suggested_checks.len(), 1);
    let empty: DetectChangesResult = serde_json::from_str(r#"{}"#).expect("valid");
    assert!(empty.changed_files.is_empty());
}

#[test]
fn parse_live_mem_check_stats_timeline_current_shapes() {
    let check = r#"{"memory":"ok","memory_path":"/tmp/memory.db","schema_memory":2,"observations":12,"sessions":9,"projects":["fabrik"]}"#;
    let checked: MemCheckResult = serde_json::from_str(check).expect("valid");
    assert_eq!(checked.memory, "ok");
    assert_eq!(checked.projects, vec!["fabrik".to_owned()]);
    let stats = r#"{"observation_count":11,"session_count":9,"projects":["fabrik"]}"#;
    let totals: ContextStats = serde_json::from_str(stats).expect("valid");
    assert_eq!(totals.observation_count, 11);
    let timeline = r#"{"id":"o-1","before":[],"after":[],"observation":{"id":"o-1","session_id":"s","project":"p","kind":"decision","title":"t","body":"b","created_at":"2026-10-14T08:47:29Z"}}"#;
    let tl: MemTimelineResult = serde_json::from_str(timeline).expect("valid");
    assert_eq!(tl.id, "o-1");
    assert!(tl.before.is_empty());
    let current = r#"{"project":"C:\\work","source":"cwd"}"#;
    let cur: CurrentProjectResult = serde_json::from_str(current).expect("valid");
    assert_eq!(cur.source, "cwd");
    let none: CurrentProjectResult =
        serde_json::from_str(r#"{"project":"","source":"none"}"#).expect("valid");
    assert_eq!(none.source, "none");
}

#[test]
fn explore_issue_rejects_bad_ids_without_spawning() {
    let repo = std::path::Path::new(".");
    assert!(fetch_graph_explore_issue(repo, "q", "").is_err());
    assert!(fetch_graph_explore_issue(repo, "q", "bd-1").is_err());
    assert!(fetch_graph_explore_issue(repo, "q", "f-XYZ").is_err());
    assert!(fetch_graph_context(repo, "   ").is_err());
    assert!(fetch_graph_impact(repo, "", 3).is_err());
    assert!(fetch_mem_timeline("", 5, None).is_err());
}

#[test]
fn graph_context_impact_reject_file_anchors_without_spawning() {
    // Live-confirmed: `graph context 'file:cli/go.mod'` fails with
    // `node sym:file:cli/go.mod not found` — the GUI must never spawn
    // context/impact for file anchors (validation precedes `run_fabrik`).
    let repo = std::path::Path::new(".");
    for bad in [
        "file:cli/go.mod",
        "file:main.go",
        "  file:x  ",
        "main.go",
        "sym",
    ] {
        assert!(
            fetch_graph_context(repo, bad).is_err(),
            "{bad:?} must fail validation"
        );
        assert!(
            fetch_graph_impact(repo, bad, 3).is_err(),
            "{bad:?} must fail validation"
        );
    }
    // (`sym:` anchors pass validation and reach `run_fabrik`, so they
    // are not asserted here — no spawning in unit tests.)
}
