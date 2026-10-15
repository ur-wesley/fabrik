use std::collections::HashSet;

use eframe::egui::{Color32, RichText, Ui};

use crate::fabrik::GraphNode;
use crate::ui::theme;

use super::model::{MAX_NODES, column_tint};
pub(super) const LEGEND_MAX: usize = 8;
/// Deduped `(kind label, dot color)` pairs in draw order, mirroring the
/// node dedupe + [`MAX_NODES`] cap of [`build_lookup_graph`]. Unknown kinds
/// reuse the column-role tint of their first occurrence.
pub(super) fn legend_entries(columns: &[(String, Vec<GraphNode>)]) -> Vec<(String, Color32)> {
    let mut seen_id: HashSet<&str> = HashSet::new();
    let mut seen_kind: HashSet<String> = HashSet::new();
    let mut entries: Vec<(String, Color32)> = Vec::new();
    let mut shown: usize = 0;
    'columns: for (col, (_, nodes)) in columns.iter().enumerate() {
        for node in nodes {
            if shown >= MAX_NODES {
                break 'columns;
            }
            if seen_id.insert(node.id.as_str()) {
                shown = shown.saturating_add(1);
                let label = if node.kind.trim().is_empty() {
                    "unknown".to_owned()
                } else {
                    node.kind.clone()
                };
                if seen_kind.insert(label.clone()) {
                    let color = theme::kind_color(&node.kind).unwrap_or_else(|| column_tint(col));
                    entries.push((label, color));
                }
            }
        }
    }
    entries
}

/// Legend row: color dot + kind label per kind present, capped at
/// [`LEGEND_MAX`] entries with an overflow counter. Renders nothing when
/// the graph is empty. The compact header shows this inside a popover
/// instead of a permanent row.
pub fn show_legend(ui: &mut Ui, columns: &[(String, Vec<GraphNode>)]) {
    let entries = legend_entries(columns);
    if entries.is_empty() {
        return;
    }
    let shown = entries.len().min(LEGEND_MAX);
    ui.horizontal_wrapped(|ui| {
        for (label, color) in entries.iter().take(shown) {
            ui.label(RichText::new("●").color(*color));
            ui.label(
                RichText::new(label)
                    .font(theme::font_footnote())
                    .color(theme::text_secondary()),
            );
        }
        let overflow = entries.len().saturating_sub(shown);
        if overflow > 0 {
            ui.label(
                RichText::new(format!("+{overflow} more"))
                    .font(theme::font_footnote())
                    .color(theme::text_tertiary()),
            );
        }
    });
}
#[cfg(test)]
mod tests {
    use super::{LEGEND_MAX, legend_entries};
    use crate::fabrik::GraphNode;
    use crate::ui::theme;

    fn kinded_node(id: &str, kind: &str) -> GraphNode {
        GraphNode {
            id: id.to_owned(),
            kind: kind.to_owned(),
            name: id.to_owned(),
            path: "src/a.rs".to_owned(),
            line: 1,
            lang: "rust".to_owned(),
        }
    }

    #[test]
    fn legend_dedupes_kinds_and_labels_blank_kind_unknown() {
        let columns = vec![
            (
                "one".to_owned(),
                vec![
                    kinded_node("a", "fn"),
                    kinded_node("b", "fn"),
                    kinded_node("blank", ""),
                ],
            ),
            ("two".to_owned(), vec![kinded_node("c", "mystery-kind")]),
        ];
        let entries = legend_entries(&columns);
        assert_eq!(entries.len(), 3);
        assert_eq!(entries.first().map(|(kind, _)| kind.as_str()), Some("fn"));
        assert_eq!(
            entries.get(1).map(|(kind, _)| kind.as_str()),
            Some("unknown")
        );
        assert_eq!(
            entries.get(1).map(|(_, color)| *color),
            Some(theme::text_primary())
        );
        assert_eq!(
            entries.get(2).map(|(_, color)| *color),
            Some(theme::text_secondary())
        );
    }

    #[test]
    fn legend_covers_many_kinds_for_capped_render() {
        let nodes: Vec<GraphNode> = (0..LEGEND_MAX.saturating_add(4))
            .map(|i| kinded_node(&format!("n{i}"), &format!("kind-{i}")))
            .collect();
        let columns = vec![("results".to_owned(), nodes)];
        let entries = legend_entries(&columns);
        assert_eq!(entries.len(), LEGEND_MAX.saturating_add(4));
        let overflow = entries.len().saturating_sub(LEGEND_MAX);
        assert!(overflow > 0);
    }
}
