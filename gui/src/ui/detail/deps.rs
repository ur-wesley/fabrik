use eframe::egui::{self, Ui};

use crate::model::{Action, DepNode, IssueDetail};

use super::super::{badge, icons, theme};
use super::OpsForm;

pub(super) fn dependency_section(
    ui: &mut Ui,
    detail: &IssueDetail,
    ops: &mut OpsForm<'_>,
) -> Option<Action> {
    let mut action = None;

    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = theme::SPACING;
        ui.label(
            egui::RichText::new(icons::LINK)
                .small()
                .color(theme::dim_text()),
        );
        ui.label(
            egui::RichText::new("DEPENDENCIES & BLOCKERS")
                .small()
                .strong()
                .color(theme::dim_text()),
        );
    });
    ui.add_space(theme::SPACING);

    let _ = theme::muted_frame()
        .inner_margin(egui::Margin::same(theme::PAD_I8))
        .show(ui, |ui| {
            theme::fill_width(ui);

            ui.horizontal(|ui| {
                ui.label(
                    egui::RichText::new("Depends on (incoming blockers)")
                        .small()
                        .strong()
                        .color(theme::foreground()),
                );
                badge::badge(ui, &detail.dependencies.len().to_string());
            });
            ui.add_space(theme::SPACING);

            if detail.dependencies.is_empty() {
                ui.label(
                    egui::RichText::new("No blocking dependencies.")
                        .small()
                        .italics()
                        .color(theme::dim_text()),
                );
            } else {
                for node in &detail.dependencies {
                    if let Some(act) = dep_card_row(ui, node) {
                        action = Some(act);
                    }
                    ui.add_space(theme::SPACING);
                }
            }

            ui.add_space(theme::SPACING);
            ui.painter().hline(
                ui.max_rect().x_range(),
                ui.cursor().top(),
                theme::hairline(),
            );
            ui.add_space(theme::SPACING);

            ui.horizontal(|ui| {
                ui.label(
                    egui::RichText::new("Blocks (outgoing dependents & subtasks)")
                        .small()
                        .strong()
                        .color(theme::foreground()),
                );
                badge::badge(ui, &detail.dependents.len().to_string());
            });
            ui.add_space(theme::SPACING);

            if detail.dependents.is_empty() {
                ui.label(
                    egui::RichText::new("No issues blocked by this.")
                        .small()
                        .italics()
                        .color(theme::dim_text()),
                );
            } else {
                for node in &detail.dependents {
                    if let Some(act) = dep_card_row(ui, node) {
                        action = Some(act);
                    }
                    ui.add_space(theme::SPACING);
                }
            }

            ui.add_space(theme::SPACING);
            if let Some(act) = dep_add_row(ui, ops) {
                action = Some(act);
            }
        });

    action
}

fn dep_add_row(ui: &mut Ui, ops: &mut OpsForm<'_>) -> Option<Action> {
    let mut action = None;
    ui.horizontal(|ui| {
        ui.label(
            egui::RichText::new("blocker")
                .small()
                .color(theme::dim_text()),
        );
        let target_resp = ui.add(
            egui::TextEdit::singleline(ops.dep_target)
                .hint_text("f-…")
                .desired_width(130.0),
        );
        if target_resp.changed() {
            action = Some(Action::SetDepTarget((*ops.dep_target).clone()));
        }
        for kind in crate::fabrik::DEP_KINDS {
            if ui
                .selectable_value(ops.dep_kind, kind.to_owned(), kind)
                .changed()
            {
                action = Some(Action::SetDepKind((*ops.dep_kind).clone()));
            }
        }
        let ready = !ops.busy && !ops.dep_target.trim().is_empty();
        let add = ui.add_enabled(ready, egui::Button::new("Add"));
        if add.clicked() {
            action = Some(Action::SubmitDepAdd);
        }
    });
    ui.label(
        egui::RichText::new("`fabrik issue dep <blocked> <blocker> --kind blocks|relates`")
            .small()
            .color(theme::dim_text()),
    );
    action
}

fn dep_card_row(ui: &mut Ui, node: &DepNode) -> Option<Action> {
    let node_id = node.id.clone();
    let card_id = ui.make_persistent_id(&node_id);
    let was_hovered = ui.data_mut(|d| d.get_temp::<bool>(card_id).unwrap_or(false));

    let (fill, stroke) = if was_hovered {
        (
            theme::surface_hover(),
            egui::Stroke::new(1.0_f32, theme::border_hover()),
        )
    } else {
        (theme::surface_elevated(), theme::hairline())
    };

    let inner = theme::muted_frame()
        .fill(fill)
        .stroke(stroke)
        .inner_margin(egui::Margin::symmetric(theme::PAD_I8, theme::SPACING_I8))
        .show(ui, |ui| {
            theme::fill_width(ui);
            ui.horizontal(|ui| {
                ui.spacing_mut().item_spacing = egui::vec2(theme::SPACING, 0.0);

                if let Some(status) = &node.status {
                    badge::status_badge(ui, status);
                }
                if let Some(priority) = node.priority {
                    badge::priority_badge(ui, priority);
                }
                badge::issue_id(ui, &node.id);
                if let Some(kind) = &node.kind {
                    badge::kind_badge(ui, kind);
                }
                if let Some(dep_type) = &node.dep_type {
                    badge::badge_color(ui, dep_type, theme::dim_text());
                }
                if !node.title.is_empty() {
                    ui.add(
                        egui::Label::new(
                            egui::RichText::new(&node.title)
                                .small()
                                .color(theme::foreground()),
                        )
                        .truncate()
                        .selectable(false),
                    );
                }

                ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
                    ui.label(egui::RichText::new(icons::ARROW_UP_RIGHT).small().color(
                        if was_hovered {
                            theme::neon_cyan()
                        } else {
                            theme::dim_text()
                        },
                    ));
                });
            });
        });

    let response = inner.response.interact(egui::Sense::click());
    let is_now_hovered = response.hovered();
    if is_now_hovered != was_hovered {
        ui.data_mut(|d| d.insert_temp(card_id, is_now_hovered));
        ui.ctx().request_repaint();
    }
    let response = theme::pointer(response);
    if response.clicked() {
        Some(Action::OpenDetail(node_id))
    } else {
        None
    }
}
