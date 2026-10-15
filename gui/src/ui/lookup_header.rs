use eframe::egui::{self, Ui};

use crate::model::Action;

use super::lookup::LookupView;
use super::{button, graph_view, theme};

/// The single lookup header strip, replacing the old stack of status
/// card, viewer title, legend row, zoom row, and hint line. It holds
/// status chips with a contextual Build button, the graph-ops group,
/// zoom buttons, and a legend popover. Last-build, seed, and hint
/// details live in tooltips.
///
/// Layout: one fixed-height (`CONTROL_H`) horizontal row,
/// `left_to_right(Align::Center)` with no wrapping, so labels, buttons,
/// and separators render as a single vertically centered line. The
/// `interact_size.y` override gives every button the uniform control
/// height (the `chrome::route_toolbar` pattern); separators are explicit
/// sized vlines spanning the same height.
pub(super) fn header_strip(ui: &mut Ui, view: &LookupView<'_>) -> Vec<Action> {
    let mut actions = Vec::new();
    theme::card_frame().show(ui, |ui| {
        ui.allocate_ui_with_layout(
            egui::vec2(ui.available_width(), theme::CONTROL_H),
            egui::Layout::left_to_right(egui::Align::Center),
            |ui| {
                ui.spacing_mut().item_spacing.x = 6.0;
                ui.spacing_mut().interact_size.y = theme::CONTROL_H;
                status_chips(ui, view);
                vline(ui);
                if view.show_graph_setup {
                    actions.extend(build_button(ui, view));
                    vline(ui);
                }
                actions.extend(graph_ops(ui, view));
                vline(ui);
                graph_view::zoom_buttons(ui);
                vline(ui);
                legend_popover(ui, view);
            },
        );
    });
    actions
}

/// Centered vertical divider spanning the fixed strip height, keeping
/// every header element on the shared `CONTROL_H` line.
fn vline(ui: &mut Ui) {
    ui.add_sized(
        egui::vec2(2.0, theme::CONTROL_H),
        egui::Separator::default().vertical(),
    );
}

fn chip(ui: &mut Ui, text: String, color: egui::Color32, tip: String) {
    theme::muted_frame()
        .inner_margin(egui::Margin::symmetric(6, 2))
        .corner_radius(theme::radius_sm())
        .show(ui, |ui| {
            let _ = ui
                .label(egui::RichText::new(text).small().color(color))
                .on_hover_text(tip);
        });
}

fn status_tip(view: &LookupView<'_>) -> String {
    let mut tip = if view.status_last_build_at.is_empty() {
        "never built".to_owned()
    } else {
        format!("last build {}", view.status_last_build_at)
    };
    if let Some(seed) = view.issue_seed {
        tip = format!("{tip} · seeded from {seed}");
    }
    if view.show_graph_setup && !view.graph_setup_body.is_empty() {
        tip = format!("{tip} · {}", view.graph_setup_body);
    }
    tip
}

fn status_chips(ui: &mut Ui, view: &LookupView<'_>) {
    let tip = status_tip(view);
    let (state_text, state_color) = if view.status_indexed {
        ("indexed", theme::sys_green())
    } else {
        ("not indexed", theme::sys_orange())
    };
    chip(ui, format!("graph {state_text}"), state_color, tip.clone());
    chip(
        ui,
        format!("{} files", view.status_files),
        theme::text_secondary(),
        tip.clone(),
    );
    chip(
        ui,
        format!("{} symbols", view.status_symbols),
        theme::text_secondary(),
        tip.clone(),
    );
    chip(
        ui,
        format!("{} stale", view.stale_files),
        if view.stale_files > 0 {
            theme::sys_orange()
        } else {
            theme::text_tertiary()
        },
        tip,
    );
}

/// Contextual single Build, folded in from the old setup banner: visible
/// only while unbuilt/stale (the route toolbar owns the fresh-state
/// Rebuild), disabled + spinner while busy, banner body as tooltip.
fn build_button(ui: &mut Ui, view: &LookupView<'_>) -> Vec<Action> {
    let mut actions = Vec::new();
    let label = if view.graph_setup_busy {
        if view.graph_setup_rebuild {
            "Rebuilding…"
        } else {
            "Building…"
        }
    } else if view.graph_setup_rebuild {
        "Rebuild graph"
    } else {
        "Build graph"
    };
    ui.add_enabled_ui(!view.loading, |ui| {
        let btn = button::button(
            ui,
            &button::ButtonProps::new(label)
                .variant(button::ButtonVariant::Secondary)
                .min_size(egui::vec2(140.0, theme::CONTROL_H)),
        );
        if btn.on_hover_text(view.graph_setup_body.clone()).clicked() {
            actions.push(Action::RunGraphBuild);
        }
    });
    if view.graph_setup_busy {
        ui.add(egui::Spinner::new());
    }
    actions
}

fn legend_popover(ui: &mut Ui, view: &LookupView<'_>) {
    let total: usize = view
        .graph_columns
        .iter()
        .map(|(_, nodes)| nodes.len())
        .sum();
    let label = if total == 0 {
        "legend".to_owned()
    } else {
        format!("legend · {total}")
    };
    let _ = ui
        .menu_button(label, |ui| {
            graph_view::show_legend(ui, &view.graph_columns);
            ui.label(
                egui::RichText::new(graph_view::HINT_TEXT)
                    .font(theme::font_caption2())
                    .color(theme::text_tertiary()),
            );
        })
        .response
        .on_hover_text("Node kind colors — click to open");
}

/// The one graph-ops group: context / impact (+depth stepper) /
/// changes (HEAD, folded-in stale review) / clear (inspect + path).
/// Context/impact enable only for `sym:` anchors; a selected file
/// anchor shows path info + hint instead of firing (file anchors fail
/// live with `node sym:file:… not found`). Busy disables everything.
/// Renders bare buttons so the header strip stays a single row.
fn graph_ops(ui: &mut Ui, view: &LookupView<'_>) -> Vec<Action> {
    let mut actions = Vec::new();
    let busy = view.loading;
    let sym_anchored = view.anchor.is_some_and(crate::model::is_symbol_anchor);
    let file_anchored = view.anchor.is_some_and(crate::model::is_file_anchor);
    ui.add_enabled_ui(!busy && sym_anchored, |ui| {
        if ui.small_button("context").clicked()
            && let Some(anchor) = view.anchor
        {
            actions.push(Action::FetchGraphContext(anchor.to_owned()));
        }
        if ui.small_button("impact").clicked()
            && let Some(anchor) = view.anchor
        {
            actions.push(Action::FetchGraphImpact {
                symbol: anchor.to_owned(),
                depth: view.impact_depth,
            });
        }
        if ui.small_button("−").clicked()
            && let Some(anchor) = view.anchor
        {
            actions.push(Action::FetchGraphImpact {
                symbol: anchor.to_owned(),
                depth: view.impact_depth.saturating_sub(1).max(1),
            });
        }
        ui.label(
            egui::RichText::new(format!("depth {}", view.impact_depth))
                .font(theme::font_caption2())
                .color(theme::text_tertiary()),
        );
        if ui.small_button("+").clicked()
            && let Some(anchor) = view.anchor
        {
            actions.push(Action::FetchGraphImpact {
                symbol: anchor.to_owned(),
                depth: view.impact_depth.saturating_add(1).min(10),
            });
        }
    });
    let changes_label = if view.stale_files > 0 {
        format!("{} changed", view.stale_files)
    } else {
        "changes".to_owned()
    };
    // File-node anchor: no fetch buttons (they are gated to `sym:`
    // above) — show the path plus the symbol hint instead of firing.
    if file_anchored && let Some(anchor) = view.anchor {
        let path = anchor.trim().strip_prefix("file:").unwrap_or(anchor);
        ui.label(
            egui::RichText::new(format!("file {path}"))
                .font(theme::font_caption2())
                .color(theme::text_secondary()),
        )
        .on_hover_text(
            "File node: context and impact need a symbol anchor — select a symbol node.",
        );
    }
    ui.add_enabled_ui(!busy, |ui| {
        if ui.small_button(changes_label).clicked() {
            actions.push(Action::FetchDetectChanges("HEAD".to_owned()));
        }
    });
    let has_inspect = view.context.is_some()
        || view.impact.is_some()
        || view.detect.is_some()
        || view.explain.is_some()
        || view.path.is_some()
        || view.anchor.is_some()
        || view.issue_seed.is_some();
    ui.add_enabled_ui(!busy && has_inspect, |ui| {
        if ui.small_button("clear").clicked() {
            actions.push(Action::LookupClearInspect);
            actions.push(Action::LookupClearPath);
        }
    });
    actions
}
