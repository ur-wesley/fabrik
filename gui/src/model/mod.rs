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
    DEFAULT_KIND, DEFAULT_PRIORITY, format_clock, format_local, matches_filter, matches_search,
    recent_from_path, split_issue_id, IssueIdParts,
};
pub use screen::*;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn split_prefix_body_rev() {
        let parts = split_issue_id("bd-neon-12");
        assert_eq!(parts.prefix, "bd-neon");
        assert_eq!(parts.number, "12");
    }

    #[test]
    fn split_prefix_body() {
        let parts = split_issue_id("custom");
        assert_eq!(parts.prefix, "custom");
        assert_eq!(parts.number, "");
    }

    #[test]
    fn parse_dep_objects_and_ids() {
        let raw = r#"[{"id":"bd-1","title":"first","type":"task","status":"open","priority":1,"dep_type":"blocks"},{"id":"bd-2"}]"#;
        let deps: Vec<DepNode> = serde_json::from_str::<Vec<DepJson>>(raw)
            .expect("valid")
            .into_iter()
            .map(DepJson::into_node)
            .collect();
        assert_eq!(deps.len(), 2);
        assert_eq!(deps.first().map(|d| d.id.as_str()), Some("bd-1"));
        assert_eq!(deps.first().map(|d| d.title.as_str()), Some("first"));
        assert_eq!(deps.get(1).map(|d| d.id.as_str()), Some("bd-2"));
    }

    #[test]
    fn parse_show_notes_shape() {
        let raw = r#"{"id":"f-k8m2n4p1","title":"t","body":"","acceptance":"","status":"open","priority":2,"kind":"task","notes":[{"id":"n-a1b2c3d4","body":"first **bold**","created_at":"2026-10-09T14:35:49Z"},{"id":"n-b2c3d4e5","body":"second","created_at":null}]}"#;
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
        let raw = r#"{"id":"bd-1","title":"t","description":null,"acceptance_criteria":null,"status":"open","priority":2,"kind":"task","assignee":null,"owner":null,"parent":null,"created_at":null,"updated_at":null,"started_at":null,"closed_at":null,"dependencies":[],"dependents":[],"dependency_count":0,"dependent_count":0,"comment_count":0}"#;
        let detail: IssueDetail = serde_json::from_str(raw).expect("valid");
        assert_eq!(detail.comments, Vec::<Comment>::new());
    }
}
