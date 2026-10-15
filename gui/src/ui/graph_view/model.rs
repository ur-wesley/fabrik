use std::collections::{HashMap, HashSet, hash_map::DefaultHasher};
use std::hash::{Hash, Hasher};

use eframe::egui::{self, Color32};
use egui_graphs::Graph;

use crate::fabrik::GraphNode;
use crate::ui::theme;
pub const MAX_NODES: usize = 300;
pub const MAX_EDGES: usize = 400;
pub(super) const VIEW_H: f32 = 320.0;
const SEED_R: f32 = 120.0;
const LABEL_MAX: usize = 32;

pub(super) type Columns = Vec<(String, Vec<GraphNode>)>;
pub(super) type EdgeTriples = Vec<(String, String, String)>;
#[derive(Clone, Debug)]
pub(super) struct NodePayload {
    pub(super) id: String,
    pub(super) path_line: String,
}

pub(super) type LookupGraph = Graph<NodePayload, String, petgraph::Directed>;

/// Frame-persistent (fingerprint, settled, graph): drags + layout survive frames.
pub(super) type CachedGraph = (u64, bool, LookupGraph);

pub(super) fn fingerprint(
    columns: &[(String, Vec<GraphNode>)],
    edges: &[(String, String, String)],
    anchor: Option<&str>,
) -> u64 {
    let mut hasher = DefaultHasher::new();
    anchor.hash(&mut hasher);
    for (label, nodes) in columns {
        label.hash(&mut hasher);
        for node in nodes {
            node.id.hash(&mut hasher);
            node.name.hash(&mut hasher);
            node.path.hash(&mut hasher);
            node.line.hash(&mut hasher);
            node.kind.hash(&mut hasher);
        }
    }
    for (src, dst, label) in edges {
        src.hash(&mut hasher);
        dst.hash(&mut hasher);
        label.hash(&mut hasher);
    }
    hasher.finish()
}

/// Column-role fallback tint for node kinds without a dedicated color.
pub(super) fn column_tint(col: usize) -> Color32 {
    match col {
        0 => theme::text_primary(),
        1 => theme::text_secondary(),
        _ => theme::text_tertiary(),
    }
}
/// Flattened, capped, circle-seeded graph for the force-directed view.
pub(super) fn build_lookup_graph(
    columns: &[(String, Vec<GraphNode>)],
    edges: &[(String, String, String)],
    anchor: Option<&str>,
) -> LookupGraph {
    let mut flat: Vec<(usize, &GraphNode)> = Vec::new();
    let mut seen = HashSet::new();
    for (col, (_, nodes)) in columns.iter().enumerate() {
        for node in nodes {
            if seen.insert(node.id.clone()) {
                flat.push((col, node));
            }
        }
    }
    let shown = flat.len().min(MAX_NODES);
    let denom = u16::try_from(shown.max(1)).map_or(1.0, f32::from);
    let mut graph = LookupGraph::new();
    let mut index_of = HashMap::new();
    for (pos, (col, node)) in flat.iter().take(shown).enumerate() {
        let pos_f = u16::try_from(pos).map_or_else(|_| f32::from(u16::MAX), f32::from);
        let angle = std::f32::consts::TAU * pos_f / denom;
        let location = egui::pos2(angle.cos() * SEED_R, angle.sin() * SEED_R);
        let tint = if anchor == Some(node.id.as_str()) {
            theme::accent()
        } else {
            theme::kind_color(&node.kind).unwrap_or_else(|| column_tint(*col))
        };
        let path_line = if node.line > 0 {
            format!("{}:{}", node.path, node.line)
        } else if node.path.is_empty() {
            node.kind.clone()
        } else {
            node.path.clone()
        };
        let label: String = node.name.chars().take(LABEL_MAX).collect();
        let payload = NodePayload {
            id: node.id.clone(),
            path_line,
        };
        let idx = graph.add_node_with_label_and_location(payload, label, location);
        if let Some(g) = graph.node_mut(idx) {
            g.set_color(tint);
            g.set_selected(anchor == Some(node.id.as_str()));
        }
        index_of.insert(node.id.clone(), idx);
    }
    for (src, dst, label) in edges.iter().take(MAX_EDGES) {
        let (Some(a), Some(b)) = (index_of.get(src), index_of.get(dst)) else {
            continue;
        };
        graph.add_edge_with_label(*a, *b, label.clone(), label.clone());
    }
    graph
}
#[cfg(test)]
mod tests {
    use super::super::columns::build_columns;
    use super::{MAX_EDGES, MAX_NODES};
    use super::{build_lookup_graph, fingerprint};
    use crate::fabrik::GraphNode;
    use crate::ui::theme;

    fn node(id: &str) -> GraphNode {
        GraphNode {
            id: id.to_owned(),
            kind: "fn".to_owned(),
            name: id.to_owned(),
            path: "src/a.rs".to_owned(),
            line: 1,
            lang: "rust".to_owned(),
        }
    }

    fn kinded_node(id: &str, kind: &str) -> GraphNode {
        let mut kept = node(id);
        kept.kind = kind.to_owned();
        kept
    }

    #[test]
    fn interactive_graph_caps_nodes_and_edges() {
        let count = MAX_NODES.saturating_add(50);
        let nodes: Vec<GraphNode> = (0..count).map(|i| node(&format!("n{i}"))).collect();
        let columns = vec![("results".to_owned(), nodes)];
        let edges: Vec<(String, String, String)> = (0..MAX_EDGES.saturating_add(50))
            .map(|i| {
                let n = i % MAX_NODES;
                (
                    format!("n{n}"),
                    format!("n{}", n.saturating_add(1)),
                    "calls".to_owned(),
                )
            })
            .collect();
        let graph = build_lookup_graph(&columns, &edges, None);
        assert_eq!(graph.node_count(), MAX_NODES);
        assert!(graph.edge_count() <= MAX_EDGES);
    }

    #[test]
    fn interactive_graph_dedupes_and_marks_anchor() {
        let first = node("dup");
        let mut second = node("dup");
        second.name = "other".to_owned();
        let columns = vec![
            ("one".to_owned(), vec![first]),
            ("two".to_owned(), vec![second, node("solo")]),
        ];
        let graph = build_lookup_graph(&columns, &[], Some("solo"));
        assert_eq!(graph.node_count(), 2);
        assert!(graph.nodes_iter().any(|(_, n)| n.label() == "dup"));
        assert!(graph.nodes_iter().all(|(_, n)| n.label() != "other"));
        assert!(graph.nodes_iter().any(|(_, n)| n.selected()));
    }

    #[test]
    fn fingerprint_stable_but_anchor_sensitive() {
        let nodes = vec![node("a"), node("b")];
        let print = |anchor: Option<&str>| {
            let (cols, edges) = build_columns(&nodes, anchor, None, None, None, None);
            fingerprint(&cols, &edges, anchor)
        };
        assert_eq!(print(Some("a")), print(Some("a")));
        assert_ne!(print(Some("a")), print(Some("b")));
    }

    #[test]
    fn known_kinds_use_theme_colors_and_anchor_keeps_accent() {
        let kinds = [
            "fn", "struct", "enum", "trait", "file", "symbol", "calls", "module",
        ];
        let nodes: Vec<GraphNode> = kinds
            .iter()
            .enumerate()
            .map(|(i, kind)| kinded_node(&format!("n{i}"), kind))
            .collect();
        let columns = vec![("results".to_owned(), nodes.clone())];
        let graph = build_lookup_graph(&columns, &[], Some("n0"));
        assert_eq!(graph.node_count(), kinds.len());
        for (_, entry) in graph.nodes_iter() {
            let kind = nodes
                .iter()
                .find(|kept| kept.id == entry.payload().id)
                .map_or_else(String::new, |kept| kept.kind.clone());
            if entry.payload().id == "n0" {
                assert_eq!(entry.color(), Some(theme::accent()));
                assert!(entry.selected());
            } else {
                assert_eq!(entry.color(), theme::kind_color(&kind));
                assert!(entry.color().is_some());
                assert!(!entry.selected());
            }
        }
    }

    #[test]
    fn unknown_kinds_fall_back_by_column_role() {
        let columns = vec![
            ("one".to_owned(), vec![kinded_node("a", "mystery-kind")]),
            ("two".to_owned(), vec![kinded_node("b", "mystery-kind")]),
            ("three".to_owned(), vec![kinded_node("c", "mystery-kind")]),
        ];
        assert_eq!(theme::kind_color("mystery-kind"), None);
        let graph = build_lookup_graph(&columns, &[], None);
        let mut tints: Vec<eframe::egui::Color32> = Vec::new();
        for (_, entry) in graph.nodes_iter() {
            tints.push(entry.color().unwrap_or_default());
        }
        assert!(tints.contains(&theme::text_primary()));
        assert!(tints.contains(&theme::text_secondary()));
        assert!(tints.contains(&theme::text_tertiary()));
    }
}
