mod action;
mod issue;
mod json;
mod query;
mod screen;

pub use action::Action;
pub use issue::*;
#[allow(unused_imports)]
pub use json::{
    DepFields, DepJson, IssueJson, IssueRefJson, NoteJson, ShowIssueJson, deserialize_dep_list,
};
#[allow(unused_imports)]
pub use query::{
    DEFAULT_KIND, DEFAULT_PRIORITY, IssueIdParts, format_clock, format_local, is_file_anchor,
    is_issue_id, is_observation_id, is_symbol_anchor, matches_filter, matches_search,
    project_label, recent_from_path, split_issue_id,
};
pub use screen::*;

#[cfg(test)]
mod tests {
    use std::path::PathBuf;

    use super::*;

    #[test]
    fn split_f_id() {
        let parts = split_issue_id("f-a1b2c3d4");
        assert_eq!(parts.prefix, "f");
        assert_eq!(parts.number, "a1b2c3d4");
    }

    #[test]
    fn split_prefix_body() {
        let parts = split_issue_id("custom");
        assert_eq!(parts.prefix, "custom");
        assert_eq!(parts.number, "");
    }

    #[test]
    fn issue_id_validator_matches_cli() {
        assert!(is_issue_id("f-a1b2c3d4"));
        assert!(is_issue_id("f-0123456789abcdef0123456789abcdef"));
        assert!(!is_issue_id("f-a1b2c3d"));
        assert!(!is_issue_id("f-0123456789abcdef0123456789abcdef0"));
        assert!(!is_issue_id("f-A1B2C3D4"));
        assert!(!is_issue_id("f-k8m2n4p1"));
        assert!(!is_issue_id("bd-1"));
        assert!(!is_issue_id("f-"));
        assert!(!is_issue_id(""));
    }

    #[test]
    fn observation_id_validator_matches_cli() {
        assert!(is_observation_id("o-8eee7c16"));
        assert!(is_observation_id("o-a"));
        assert!(!is_observation_id("o-"));
        assert!(!is_observation_id("o-XYZ"));
        assert!(!is_observation_id("358"));
        assert!(!is_observation_id(""));
    }

    #[test]
    fn graph_anchor_classifiers_gate_context_fetches() {
        assert!(is_symbol_anchor("sym:main.go:main"));
        assert!(is_symbol_anchor("  sym:x  "));
        assert!(!is_symbol_anchor("file:cli/go.mod"));
        assert!(!is_symbol_anchor(""));
        assert!(!is_symbol_anchor("sym"));
        assert!(is_file_anchor("file:cli/go.mod"));
        assert!(is_file_anchor("  file:x  "));
        assert!(!is_file_anchor("sym:main.go:main"));
        assert!(!is_file_anchor(""));
    }

    #[test]
    fn project_label_passes_absolute_paths_through() {
        // Drive-qualified: absolute on every platform running these tests.
        let absolute = PathBuf::from(r"C:\tmp\repo");
        assert!(absolute.is_absolute());
        assert_eq!(project_label(&absolute), r"C:\tmp\repo");
        let relative = project_label(&PathBuf::from("some/relative"));
        assert!(PathBuf::from(&relative).is_absolute());
        assert!(relative.contains("some") && relative.contains("relative"));
    }

    #[test]
    fn parse_dep_objects_and_ids() {
        let raw = r#"[{"id":"f-a1b2c3d4","title":"first","type":"task","status":"open","priority":1,"dep_type":"blocks"},{"id":"f-b2c3d4e5"}]"#;
        let deps: Vec<DepNode> = serde_json::from_str::<Vec<DepJson>>(raw)
            .expect("valid")
            .into_iter()
            .map(DepJson::into_node)
            .collect();
        assert_eq!(deps.len(), 2);
        assert_eq!(deps.first().map(|d| d.id.as_str()), Some("f-a1b2c3d4"));
        assert_eq!(deps.first().map(|d| d.title.as_str()), Some("first"));
        assert_eq!(deps.get(1).map(|d| d.id.as_str()), Some("f-b2c3d4e5"));
    }

    #[test]
    fn parse_live_issue_list_shape() {
        // Mirrors `issue list --json` (full Issue: ready flag, nullable
        // started_at/closed_at, optional blockers/notes).
        let raw = r#"[{"id":"f-a1b2c3d4","title":"t","body":"b","acceptance":"","status":"open","priority":2,"kind":"task","assignee":null,"created_at":"2026-10-14T08:00:00Z","updated_at":"2026-10-14T09:00:00Z","started_at":null,"closed_at":null,"ready":true}]"#;
        let items: Vec<IssueJson> = serde_json::from_str(raw).expect("valid");
        assert_eq!(items.len(), 1);
        assert!(items[0].ready);
        let issue = items.into_iter().next().expect("one").into_issue();
        assert_eq!(issue.id, "f-a1b2c3d4");
        assert_eq!(issue.description.as_deref(), Some("b"));
    }

    #[test]
    fn parse_live_issue_show_shape() {
        // Mirrors `issue show --json` (blockers/blocking refs plus notes
        // carrying issue_id).
        let raw = r#"{"id":"f-a1b2c3d4","title":"t","body":"","acceptance":"","status":"blocked","priority":1,"kind":"bug","assignee":"alice","created_at":"2026-10-14T08:00:00Z","updated_at":"2026-10-14T09:00:00Z","started_at":"2026-10-14T08:30:00Z","closed_at":null,"ready":false,"blockers":[{"id":"f-b2c3d4e5","title":"blocker","status":"open","priority":0,"kind":"task"}],"blocking":[],"notes":[{"id":"n-a1b2c3d4","issue_id":"f-a1b2c3d4","body":"first **bold**","created_at":"2026-10-14T09:00:00Z"}]}"#;
        let item: ShowIssueJson = serde_json::from_str(raw).expect("valid");
        let detail = item.into_detail();
        assert_eq!(detail.dependencies.len(), 1);
        assert_eq!(
            detail.dependencies.first().map(|d| d.id.as_str()),
            Some("f-b2c3d4e5")
        );
        assert_eq!(detail.comment_count, 1);
        assert_eq!(
            detail.comments.first().map(|c| c.body.as_str()),
            Some("first **bold**")
        );
        assert!(detail.started_at.is_some());
        assert_eq!(detail.assignee.as_deref(), Some("alice"));
    }

    #[test]
    fn parse_show_notes_shape() {
        let raw = r#"{"id":"f-a8b2c4d1","title":"t","body":"","acceptance":"","status":"open","priority":2,"kind":"task","notes":[{"id":"n-a1b2c3d4","body":"first **bold**","created_at":"2026-10-09T14:35:49Z"},{"id":"n-b2c3d4e5","body":"second","created_at":null}]}"#;
        let item: ShowIssueJson = serde_json::from_str(raw).expect("valid");
        let detail = item.into_detail();
        assert_eq!(detail.comment_count, 2);
        assert_eq!(detail.comments.len(), 2);
        assert_eq!(
            detail.comments.first().map(|c| c.body.as_str()),
            Some("first **bold**")
        );
        assert!(detail.comments.first().unwrap().created_at.is_some());
    }

    #[test]
    fn old_detail_cache_without_comments_defaults() {
        // Disk caches written before comments existed lack the field.
        let raw = r#"{"id":"f-a1b2c3d4","title":"t","description":null,"acceptance_criteria":null,"status":"open","priority":2,"kind":"task","assignee":null,"owner":null,"parent":null,"created_at":null,"updated_at":null,"started_at":null,"closed_at":null,"dependencies":[],"dependents":[],"dependency_count":0,"dependent_count":0,"comment_count":0}"#;
        let detail: IssueDetail = serde_json::from_str(raw).expect("valid");
        assert_eq!(detail.comments, Vec::<Comment>::new());
    }
}
