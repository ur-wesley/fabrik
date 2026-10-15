mod columns;
mod controls;
mod legend;
mod model;
mod settle;

pub use columns::build_columns;
pub use controls::{HINT_TEXT, zoom_buttons};
pub use legend::show_legend;

use eframe::egui::{self, Id, RichText, Ui};
use egui_graphs::{
    FruchtermanReingold, FruchtermanReingoldState, GraphChange, GraphView, LayoutForceDirected,
    SettingsInteraction, SettingsNavigation, SettingsStyle, reset_layout, reset_metadata,
};

use crate::fabrik::GraphNode;
use crate::model::Action;
use crate::ui::theme;

use self::controls::{CACHE_KEY, FIT_KEY, GRAPH_ID, RESET_KEY, apply_wheel_zoom, take_flag};
use self::model::{CachedGraph, VIEW_H, build_lookup_graph, fingerprint};
use self::settle::settle_fresh_graph;

type ForceView = GraphView<FruchtermanReingoldState, LayoutForceDirected<FruchtermanReingold>>;
/// Twice the card inner margin, subtracted so the canvas fills the card
/// without overflowing the fixed-height column.
const CARD_PAD: f32 = 24.0;
/// Bare force-directed canvas for the centered lookup pane: no title, no
/// legend row, no controls row, no hint line (those live in the compact
/// header strip). The caller places this OUTSIDE any `ScrollArea` at full
/// pane height so wheel events reach the graph for zoom. Click selects via
/// the shared anchor action, right-click copies `path:line`.
/// Empty input renders a hint, never blank, never panics.
pub fn graph_canvas(
    ui: &mut Ui,
    columns: &[(String, Vec<GraphNode>)],
    edges: &[(String, String, String)],
    anchor: Option<&str>,
) -> Vec<Action> {
    let mut actions = Vec::new();
    let total: usize = columns.iter().map(|(_, nodes)| nodes.len()).sum();
    theme::card_frame().show(ui, |ui| {
        ui.set_min_width(ui.available_width());
        if total == 0 {
            ui.label(
                RichText::new("No graph to draw yet — search or select a symbol.")
                    .font(theme::font_footnote())
                    .color(theme::text_secondary()),
            );
            ui.label(
                RichText::new(HINT_TEXT)
                    .font(theme::font_caption2())
                    .color(theme::text_tertiary()),
            );
            return;
        }
        show_interactive(ui, columns, edges, anchor, &mut actions);
    });
    actions
}

fn show_interactive(
    ui: &mut Ui,
    columns: &[(String, Vec<GraphNode>)],
    edges: &[(String, String, String)],
    anchor: Option<&str>,
    actions: &mut Vec<Action>,
) {
    let print = fingerprint(columns, edges, anchor);
    let reset_requested = take_flag(ui, RESET_KEY);
    let fit_requested = take_flag(ui, FIT_KEY);
    if reset_requested {
        let gid = Some(GRAPH_ID.to_owned());
        reset_layout::<FruchtermanReingoldState>(ui, gid.clone());
        reset_metadata(ui, gid);
    }
    let cache_id = Id::new(CACHE_KEY);
    let cached_opt: Option<CachedGraph> = if reset_requested {
        None
    } else {
        ui.ctx()
            .data_mut(|data| data.get_temp::<CachedGraph>(cache_id))
    };
    let fresh = reset_requested || cached_opt.as_ref().is_none_or(|entry| entry.0 != print);
    let mut cached: CachedGraph = cached_opt
        .filter(|entry| entry.0 == print)
        .unwrap_or_else(|| (print, false, build_lookup_graph(columns, edges, anchor)));
    if fresh {
        settle_fresh_graph(ui, &mut cached.2, GRAPH_ID);
    }
    let interactions = SettingsInteraction::new()
        .with_dragging_enabled(true)
        .with_node_selection_enabled(true);
    let navigations = SettingsNavigation::new()
        .with_fit_to_screen_enabled(!cached.1 || fit_requested)
        .with_zoom_and_pan_enabled(true);
    // Fill the card remainder (minus its inner padding); the caller sized
    // the column to the full pane height, so the canvas owns the wheel.
    let canvas_h = (ui.available_height() - CARD_PAD).max(VIEW_H);
    ui.allocate_ui(egui::vec2(ui.available_width(), canvas_h), |ui| {
        let view: ForceView = ForceView::new()
            .with_interactions(&interactions)
            .with_navigations(&navigations)
            .with_styles(&SettingsStyle::new().with_labels_always(true))
            .with_id(Some(GRAPH_ID.to_owned()));
        let response = view.show(ui, &mut cached.2);
        // Manual wheel-zoom on the canvas response via the same
        // MetadataFrame path as the +/- buttons (same factor/clamp).
        // The canvas owns the wheel (it lives outside any ScrollArea),
        // so a hovered wheel that the widget ignores still zooms. See
        // `apply_wheel_zoom`: drop this if runtime shows double-zoom.
        if response.response.hovered() {
            let delta_y = ui.input(|input| input.smooth_scroll_delta.y);
            if delta_y != 0.0 {
                apply_wheel_zoom(ui, delta_y);
            }
        }
        for change in &response.changes {
            if let GraphChange::NodeClicked { node } = change
                && let Some(entry) = cached.2.node(*node)
            {
                actions.push(Action::LookupSelectNode(Some(entry.payload().id.clone())));
            }
        }
        if let Some(hovered) = cached.2.hovered_node()
            && let Some(entry) = cached.2.node(hovered)
        {
            if response.response.secondary_clicked() {
                ui.ctx().copy_text(entry.payload().path_line.clone());
            }
            let tip = format!("{}\n{}", entry.payload().id, entry.payload().path_line);
            let _ = response.response.on_hover_text_at_pointer(tip);
        }
    });
    cached.1 = true;
    ui.ctx().data_mut(|data| data.insert_temp(cache_id, cached));
}
