use std::collections::HashSet;

use crate::fabrik::{GraphContextResult, GraphImpactResult, GraphNode, PathResult, QueryResult};

use super::model::{Columns, EdgeTriples, MAX_EDGES, MAX_NODES};
/// Layered columns + edges (path > impact > context > overview > capped); never panics.
pub fn build_columns(
    nodes: &[GraphNode],
    anchor: Option<&str>,
    context: Option<&GraphContextResult>,
    impact: Option<&GraphImpactResult>,
    path: Option<&PathResult>,
    overview: Option<&QueryResult>,
) -> (Columns, EdgeTriples) {
    if let Some(p) = path
        && !p.nodes.is_empty()
    {
        let total = p.nodes.len();
        let shown: Vec<GraphNode> = p.nodes.iter().take(MAX_NODES).cloned().collect();
        let label = if total > shown.len() {
            format!("path ({} of {total})", shown.len())
        } else {
            format!("path ({})", shown.len())
        };
        let edges: EdgeTriples = p
            .edges
            .iter()
            .take(MAX_EDGES)
            .map(|e| {
                let label = if e.label.is_empty() {
                    e.kind.clone()
                } else {
                    e.label.clone()
                };
                (e.src.clone(), e.dst.clone(), label)
            })
            .collect();
        return (vec![(label, shown)], edges);
    }
    if let Some(imp) = impact
        && !imp.by_depth.is_empty()
    {
        return impact_columns(nodes, anchor, imp);
    }
    if let Some(ctx) = context
        && anchor.is_some()
    {
        return context_columns(ctx);
    }
    if let Some(ov) = overview
        && !ov.nodes.is_empty()
    {
        return overview_columns(nodes, ov);
    }
    let total = nodes.len();
    let shown: Vec<GraphNode> = nodes.iter().take(MAX_NODES).cloned().collect();
    let label = if total > shown.len() {
        format!("results ({} of {total})", shown.len())
    } else {
        format!("results ({})", shown.len())
    };
    (vec![(label, shown)], Vec::new())
}

fn overview_columns(nodes: &[GraphNode], ov: &QueryResult) -> (Columns, EdgeTriples) {
    let total = nodes.len();
    let shown: Vec<GraphNode> = nodes.iter().take(MAX_NODES).cloned().collect();
    let label = if total > shown.len() {
        format!("overview ({} of {total})", shown.len())
    } else {
        format!("overview ({})", shown.len())
    };
    let drawn: HashSet<&str> = shown.iter().map(|node| node.id.as_str()).collect();
    let mut edges: EdgeTriples = EdgeTriples::new();
    for edge in &ov.edges {
        if edges.len() >= MAX_EDGES {
            break;
        }
        // Mirror the canvas renderer (`build_lookup_graph` drops edges
        // whose endpoints are not drawn), so the title count matches the
        // drawn graph. Unresolved (`dst == ""`) edges never render.
        if edge.dst.is_empty()
            || !drawn.contains(edge.src.as_str())
            || !drawn.contains(edge.dst.as_str())
        {
            continue;
        }
        let label = if edge.label.is_empty() {
            edge.kind.clone()
        } else {
            edge.label.clone()
        };
        edges.push((edge.src.clone(), edge.dst.clone(), label));
    }
    (vec![(label, shown)], edges)
}

fn context_columns(ctx: &GraphContextResult) -> (Columns, EdgeTriples) {
    let mut budget = MAX_NODES;
    let callers_n = ctx.callers.len().min(budget);
    budget = budget.saturating_sub(callers_n);
    let callers: Vec<GraphNode> = ctx.callers.iter().take(callers_n).cloned().collect();
    budget = budget.saturating_sub(1);
    let downstream_n = ctx.callees.len().min(budget);
    let mut seen = HashSet::new();
    let mut downstream: Vec<GraphNode> = Vec::new();
    for link in ctx.callees.iter().take(downstream_n) {
        if link.dst.is_empty() || !seen.insert(link.dst.clone()) {
            continue;
        }
        downstream.push(GraphNode {
            id: link.dst.clone(),
            kind: link.kind.clone(),
            name: link.dst.clone(),
            path: String::new(),
            line: 0,
            lang: String::new(),
        });
    }
    let anchor_id = ctx.symbol.id.clone();
    let mut edges: EdgeTriples = EdgeTriples::new();
    for caller in &callers {
        if edges.len() >= MAX_EDGES {
            break;
        }
        edges.push((caller.id.clone(), anchor_id.clone(), "caller".to_owned()));
    }
    for node in &downstream {
        if edges.len() >= MAX_EDGES {
            break;
        }
        let label = ctx
            .callees
            .iter()
            .find(|e| e.dst == node.id)
            .map_or_else(String::new, |e| e.kind.clone());
        edges.push((anchor_id.clone(), node.id.clone(), label));
    }
    let columns = vec![
        (format!("callers ({})", callers.len()), callers),
        ("anchor".to_owned(), vec![ctx.symbol.clone()]),
        (format!("callees ({})", downstream.len()), downstream),
    ];
    (columns, edges)
}

fn impact_columns(
    nodes: &[GraphNode],
    anchor: Option<&str>,
    imp: &GraphImpactResult,
) -> (Columns, EdgeTriples) {
    let mut depths: Vec<&String> = imp.by_depth.keys().collect();
    depths.sort();
    let anchor_node: Option<GraphNode> = anchor
        .and_then(|id| nodes.iter().find(|n| n.id == id).cloned())
        .or_else(|| {
            imp.flattened()
                .into_iter()
                .find(|n| Some(n.id.as_str()) == anchor)
                .cloned()
        })
        .or_else(|| {
            let id = anchor.unwrap_or(imp.root.as_str());
            if id.is_empty() {
                None
            } else {
                Some(GraphNode {
                    id: id.to_owned(),
                    kind: "symbol".to_owned(),
                    name: id.to_owned(),
                    path: String::new(),
                    line: 0,
                    lang: String::new(),
                })
            }
        });
    let root_id = anchor_node
        .as_ref()
        .map_or_else(|| imp.root.clone(), |n| n.id.clone());
    let mut columns: Columns = Columns::new();
    let mut edges: EdgeTriples = EdgeTriples::new();
    let mut budget = anchor_node.as_ref().map_or(MAX_NODES, |_| MAX_NODES - 1);
    if let Some(node) = anchor_node {
        columns.push(("anchor".to_owned(), vec![node]));
    }
    let mut prev_ids: Vec<String> = if root_id.is_empty() {
        Vec::new()
    } else {
        vec![root_id.clone()]
    };
    for depth in depths {
        let Some(layer) = imp.by_depth.get(depth) else {
            continue;
        };
        let n = layer.len().min(budget);
        budget = budget.saturating_sub(n);
        let shown: Vec<GraphNode> = layer.iter().take(n).cloned().collect();
        let ids: Vec<String> = shown.iter().map(|node| node.id.clone()).collect();
        for dst in &ids {
            let src = prev_ids.first().cloned().unwrap_or_else(|| root_id.clone());
            if edges.len() >= MAX_EDGES || src.is_empty() || dst.is_empty() {
                continue;
            }
            edges.push((src, dst.clone(), format!("depth {depth}")));
            if edges.len() >= MAX_EDGES {
                break;
            }
        }
        columns.push((format!("depth {depth} ({})", shown.len()), shown));
        if !ids.is_empty() {
            prev_ids = ids;
        }
        if budget == 0 || edges.len() >= MAX_EDGES {
            break;
        }
    }
    (columns, edges)
}
#[cfg(test)]
mod tests {
    use super::super::model::{MAX_EDGES, MAX_NODES};
    use super::{EdgeTriples, build_columns};
    use crate::fabrik::{
        GraphContextResult, GraphEdge, GraphImpactResult, GraphNode, PathResult, QueryResult,
    };
    use std::collections::BTreeMap;

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

    #[test]
    fn capped_single_column_with_overflow_label() {
        let count = MAX_NODES.saturating_add(20);
        let nodes: Vec<GraphNode> = (0..count).map(|i| node(&format!("n{i}"))).collect();
        let (cols, edges) = build_columns(&nodes, None, None, None, None, None);
        assert_eq!(cols.len(), 1);
        assert_eq!(cols[0].1.len(), MAX_NODES);
        assert!(cols[0].0.contains(&format!("of {count}")));
        assert_eq!(edges, EdgeTriples::new());
    }

    #[test]
    fn context_builds_three_columns_with_edges() {
        let ctx = GraphContextResult {
            symbol: node("anchor"),
            callers: vec![node("c1"), node("c2")],
            callees: vec![GraphEdge {
                src: "anchor".to_owned(),
                dst: "d1".to_owned(),
                kind: "calls".to_owned(),
                label: String::new(),
                confidence: String::new(),
            }],
            observations: Vec::new(),
            tests: Vec::new(),
        };
        let (cols, edges) = build_columns(&[], Some("anchor"), Some(&ctx), None, None, None);
        assert_eq!(cols.len(), 3);
        assert_eq!(cols[0].0, "callers (2)");
        assert_eq!(cols[1].0, "anchor");
        assert_ne!(edges, EdgeTriples::new());
    }

    #[test]
    fn impact_builds_anchor_plus_depths_and_caps_edges() {
        let by_depth = BTreeMap::from([
            ("1".to_owned(), vec![node("a"), node("b")]),
            ("2".to_owned(), vec![node("c")]),
        ]);
        let imp = GraphImpactResult {
            root: "anchor".to_owned(),
            by_depth,
            total: 3,
        };
        let (cols, edges) = build_columns(
            &[node("anchor")],
            Some("anchor"),
            None,
            Some(&imp),
            None,
            None,
        );
        assert!(cols.len() >= 3);
        assert_eq!(cols[0].0, "anchor");
        assert_ne!(edges, EdgeTriples::new());
        assert!(edges.len() <= MAX_EDGES);
    }

    fn edge(src: &str, dst: &str, kind: &str, label: &str) -> GraphEdge {
        GraphEdge {
            src: src.to_owned(),
            dst: dst.to_owned(),
            kind: kind.to_owned(),
            label: label.to_owned(),
            confidence: String::new(),
        }
    }

    fn overview(nodes: &[&str], edges: Vec<GraphEdge>) -> QueryResult {
        QueryResult {
            nodes: nodes.iter().map(|id| node(id)).collect(),
            edges,
        }
    }

    #[test]
    fn overview_renders_edges_with_label_or_kind() {
        let nodes = vec![node("a"), node("b")];
        let ov = overview(
            &["a", "b"],
            vec![edge("a", "b", "calls", ""), edge("b", "a", "calls", "back")],
        );
        let (cols, edges) = build_columns(&nodes, None, None, None, None, Some(&ov));
        assert_eq!(cols.len(), 1);
        assert_eq!(cols[0].0, "overview (2)");
        assert_eq!(
            edges,
            vec![
                ("a".to_owned(), "b".to_owned(), "calls".to_owned()),
                ("b".to_owned(), "a".to_owned(), "back".to_owned()),
            ]
        );
        assert!(edges.len() <= MAX_EDGES);
    }

    #[test]
    fn overview_edges_filter_to_drawn_ids() {
        let nodes = vec![node("a"), node("b")];
        let ov = overview(
            &["a", "b"],
            vec![
                edge("a", "ghost", "calls", ""),
                edge("ghost", "a", "calls", ""),
                edge("a", "", "calls", "unresolved"),
                edge("a", "b", "calls", ""),
            ],
        );
        let (cols, edges) = build_columns(&nodes, None, None, None, None, Some(&ov));
        assert_eq!(cols[0].0, "overview (2)");
        assert_eq!(
            edges,
            vec![("a".to_owned(), "b".to_owned(), "calls".to_owned())]
        );
    }

    #[test]
    fn overview_yields_to_path_impact_and_context() {
        let nodes = vec![node("a"), node("b")];
        let ov = overview(&["a", "b"], vec![edge("a", "b", "calls", "")]);
        let path = PathResult {
            nodes: nodes.clone(),
            edges: vec![edge("a", "b", "calls", "hop")],
        };
        let (cols, edges) = build_columns(&nodes, None, None, None, Some(&path), Some(&ov));
        assert!(cols[0].0.starts_with("path"));
        assert_eq!(edges.len(), 1);

        let by_depth = BTreeMap::from([("1".to_owned(), vec![node("b")])]);
        let imp = GraphImpactResult {
            root: "a".to_owned(),
            by_depth,
            total: 1,
        };
        let (cols, _) = build_columns(&nodes, Some("a"), None, Some(&imp), None, Some(&ov));
        assert_eq!(cols[0].0, "anchor");

        let ctx = GraphContextResult {
            symbol: node("a"),
            callers: vec![node("c")],
            callees: Vec::new(),
            observations: Vec::new(),
            tests: Vec::new(),
        };
        let (cols, _) = build_columns(&nodes, Some("a"), Some(&ctx), None, None, Some(&ov));
        assert_eq!(cols.len(), 3);
    }

    #[test]
    fn empty_overview_falls_back_to_plain_nodes() {
        let nodes = vec![node("a")];
        let ov = overview(&[], Vec::new());
        let (cols, edges) = build_columns(&nodes, None, None, None, None, Some(&ov));
        assert_eq!(cols[0].0, "results (1)");
        assert_eq!(edges, EdgeTriples::new());
    }
}
