use eframe::egui::Ui;

use crate::fabrik::{ExplainResult, GraphNode, PathResult};
use crate::model::Action;

use super::lookup::{LookupView, cap, edge_row, foot, head, node_row};
use super::lookup_inspect::{context_card, detect_card, explore_section, impact_card};
use super::theme;

/// The independent sidebar column: nodes/explain/path/context/impact/
/// detect/explore cards. The caller wraps this in its own `ScrollArea`
/// so it scrolls independently of the canvas. Selection stays synced
/// both ways through the shared anchor action (`LookupSelectNode` —
/// the same action the canvas click pushes).
pub(super) fn sidebar_sections(ui: &mut Ui, view: &LookupView<'_>, actions: &mut Vec<Action>) {
    nodes_section(ui, view, actions);
    if let Some(explain) = view.explain {
        ui.add_space(theme::SPACING);
        explain_panel(ui, explain);
    }
    if let Some(path) = view.path {
        ui.add_space(theme::SPACING);
        path_panel(ui, path, view.anchor, actions);
    }
    if let Some(symbol) = view.context_symbol
        && let Some(ctx) = view.context
    {
        ui.add_space(theme::SPACING);
        context_card(ui, symbol, ctx, view.anchor, actions);
    }
    if let Some(symbol) = view.impact_symbol
        && let Some(impact) = view.impact
    {
        ui.add_space(theme::SPACING);
        impact_card(ui, symbol, impact, view.anchor, actions);
    }
    if let Some(base) = view.detect_ref
        && let Some(detect) = view.detect
    {
        ui.add_space(theme::SPACING);
        detect_card(ui, base, detect);
    }
    if let Some(explore) = view.explore {
        ui.add_space(theme::SPACING);
        explore_section(ui, explore);
    }
}

fn nodes_section(ui: &mut Ui, view: &LookupView<'_>, actions: &mut Vec<Action>) {
    if view.nodes.is_empty() {
        if view.has_query {
            foot(ui, "no matches".to_owned());
            cap(
                ui,
                "Try another symbol, or `fabrik graph explore <query> --json`.".to_owned(),
            );
        } else {
            foot(ui, "Search a symbol to explore the graph.".to_owned());
        }
        return;
    }
    node_list(ui, view.nodes, view.anchor, actions);
}

fn node_list(ui: &mut Ui, nodes: &[GraphNode], anchor: Option<&str>, actions: &mut Vec<Action>) {
    theme::grouped_inset_frame().show(ui, |ui| {
        for (index, node) in nodes.iter().enumerate() {
            let loc = format!("{}:{}", node.path, node.line);
            let mut subtitle = format!("{} · {loc}", node.kind);
            if !node.lang.is_empty() {
                subtitle = format!("{subtitle} · {}", node.lang);
            }
            subtitle = format!("{subtitle} · {}", node.id);
            node_row(
                ui,
                node.name.clone(),
                subtitle,
                loc,
                &node.id,
                anchor,
                index.saturating_add(1) == nodes.len(),
                actions,
            );
        }
    });
}

fn explain_panel(ui: &mut Ui, explain: &ExplainResult) {
    theme::card_frame().show(ui, |ui| {
        let loc = format!("{}:{}", explain.node.path, explain.node.line);
        ui.horizontal(|ui| {
            head(ui, explain.node.name.clone());
            if theme::pointer(ui.small_button(loc.clone())).clicked() {
                ui.ctx().copy_text(loc.clone());
            }
        });
        foot(ui, format!("{} · {loc}", explain.node.kind));
        if explain.truncated {
            cap(ui, "neighborhood truncated".to_owned());
        }
        ui.add_space(theme::SPACING);
        for edge in &explain.edges {
            edge_row(ui, edge);
        }
    });
}

fn path_panel(ui: &mut Ui, path: &PathResult, anchor: Option<&str>, actions: &mut Vec<Action>) {
    theme::card_frame().show(ui, |ui| {
        head(ui, "path".to_owned());
        if path.nodes.is_empty() {
            foot(ui, "no path".to_owned());
        } else {
            for (index, node) in path.nodes.iter().enumerate() {
                let loc = format!("{}:{}", node.path, node.line);
                node_row(
                    ui,
                    node.name.clone(),
                    loc.clone(),
                    loc,
                    &node.id,
                    anchor,
                    index.saturating_add(1) == path.nodes.len(),
                    actions,
                );
            }
        }
        if !path.edges.is_empty() {
            ui.add_space(theme::SPACING);
            for edge in &path.edges {
                edge_row(ui, edge);
            }
        }
    });
}
