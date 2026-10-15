use eframe::egui::Ui;
use egui_graphs::{
    AnimatedState, FruchtermanReingold, FruchtermanReingoldState, GraphView, LayoutForceDirected,
    get_layout_state, set_layout_state,
};

use super::model::LookupGraph;

/// Fixed force-layout budget applied once per data change, before the
/// layout freezes. Large enough to untangle the capped 300-node canvas,
/// small enough to stay hitch-free in a single frame.
pub(super) const SETTLE_STEPS: u32 = 150;
const _: () = assert!(SETTLE_STEPS >= 60 && SETTLE_STEPS <= 500);

type ForceView = GraphView<FruchtermanReingoldState, LayoutForceDirected<FruchtermanReingold>>;

/// Settle a freshly built graph, then freeze it: run [`SETTLE_STEPS`]
/// force steps up front (forcing `running` even if a previous freeze is
/// persisted), then persist `running = false` so the per-frame
/// `sync_layout` becomes a no-op and nodes stop drifting. Drags still
/// move nodes directly, and Reset rebuilds + re-settles via this path.
pub(super) fn settle_fresh_graph(ui: &mut Ui, graph: &mut LookupGraph, id: &str) {
    if graph.node_count() == 0 {
        return;
    }
    ForceView::fast_forward_force_run(ui, graph, SETTLE_STEPS, Some(id.to_owned()));
    let mut state = get_layout_state::<FruchtermanReingoldState>(ui, Some(id.to_owned()));
    state.set_running(false);
    set_layout_state(ui, state, Some(id.to_owned()));
}

#[cfg(test)]
mod tests {
    use eframe::egui;
    use egui_graphs::{AnimatedState, FruchtermanReingoldState, get_layout_state};

    use super::{SETTLE_STEPS, settle_fresh_graph};
    use crate::fabrik::GraphNode;

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
    fn settle_budget_runs_then_freezes_force_layout() {
        let ctx = egui::Context::default();
        let columns = vec![("results".to_owned(), vec![node("a"), node("b"), node("c")])];
        let edges = vec![
            ("a".to_owned(), "b".to_owned(), "calls".to_owned()),
            ("b".to_owned(), "c".to_owned(), "calls".to_owned()),
        ];
        ctx.run_ui(egui::RawInput::default(), |ui| {
            let mut graph = super::super::model::build_lookup_graph(&columns, &edges, None);
            settle_fresh_graph(ui, &mut graph, "lookup-settle-test");
            assert_eq!(graph.node_count(), 3);
            let state = get_layout_state::<FruchtermanReingoldState>(
                ui,
                Some("lookup-settle-test".to_owned()),
            );
            assert!(!state.is_running());
            assert!(state.step_count() >= u64::from(SETTLE_STEPS));
        })
        .drop_without_applying_deltas();
    }
}
