use eframe::egui::{self, ScrollArea, Ui};

use crate::model::{Action, IssueDetail, format_local};

use super::super::{badge, icons, theme};
use super::OpsForm;

pub(super) fn sidebar(ui: &mut Ui, detail: &IssueDetail, ops: &mut OpsForm<'_>) -> Option<Action> {
    let mut action = None;

    ScrollArea::vertical()
        .id_salt("issue_sidebar_scroll")
        .auto_shrink([false, false])
        .show(ui, |ui| {
            if let Some(act) = render_properties_section(ui, detail) {
                action = Some(act);
            }

            ui.add_space(theme::SPACING);
            ui.painter().hline(
                ui.max_rect().x_range(),
                ui.cursor().top(),
                theme::hairline(),
            );

            if let Some(act) = render_actions_section(ui, detail, ops) {
                action = Some(act);
            }

            ui.add_space(theme::SPACING);
            ui.painter().hline(
                ui.max_rect().x_range(),
                ui.cursor().top(),
                theme::hairline(),
            );

            render_relationships_section(ui, detail);
            render_timeline_section(ui, detail);
        });

    action
}

fn render_properties_section(ui: &mut Ui, detail: &IssueDetail) -> Option<Action> {
    let mut action = None;
    sidebar_section_header(ui, "PROPERTIES");
    sidebar_prop_row(ui, "Status", |ui| {
        badge::status_badge(ui, &detail.status);
    });
    sidebar_prop_row(ui, "Priority", |ui| {
        badge::priority_badge(ui, detail.priority);
    });
    sidebar_prop_row(ui, "Type", |ui| {
        badge::kind_badge(ui, &detail.kind);
    });

    let assignee = detail.assignee.as_deref().or(detail.owner.as_deref());
    sidebar_prop_row(ui, "Assignee", |ui| {
        if let Some(name) = assignee {
            badge::badge_color(ui, &format!("{} {name}", icons::USER), theme::neon_cyan());
        } else {
            ui.label(
                egui::RichText::new("Unassigned")
                    .small()
                    .color(theme::dim_text()),
            );
        }
    });

    if let Some(parent_id) = &detail.parent {
        sidebar_prop_row(ui, "Parent", |ui| {
            let btn = ui.add(
                egui::Button::new(
                    egui::RichText::new(format!("{} {parent_id}", icons::ARROW_UP_RIGHT))
                        .small()
                        .color(theme::neon_cyan()),
                )
                .fill(egui::Color32::TRANSPARENT)
                .stroke(egui::Stroke::NONE),
            );
            if theme::pointer(btn)
                .on_hover_text("Open parent bead")
                .clicked()
            {
                action = Some(Action::OpenDetail(parent_id.clone()));
            }
        });
    }

    ui.add_space(theme::SPACING);
    let explore_btn = ui.add(
        egui::Button::new(
            egui::RichText::new(format!("{} explore in graph", icons::MAGNIFYING_GLASS))
                .small()
                .color(theme::neon_cyan()),
        )
        .fill(egui::Color32::TRANSPARENT)
        .stroke(egui::Stroke::NONE),
    );
    if theme::pointer(explore_btn)
        .on_hover_text("Seed graph explore from this issue")
        .clicked()
    {
        action = Some(Action::ExploreIssueSeed {
            issue: detail.id.clone(),
            query: detail.title.clone(),
        });
    }

    action
}

fn render_actions_section(
    ui: &mut Ui,
    detail: &IssueDetail,
    ops: &mut OpsForm<'_>,
) -> Option<Action> {
    let mut action = None;
    sidebar_section_header(ui, "ACTIONS");
    ui.add_enabled_ui(!ops.busy, |ui| {
        ui.horizontal(|ui| {
            // Claim when open, release when active; close is always offered
            // and the CLI validates the transition.
            if ops.active {
                if ui.small_button("Unclaim").clicked() {
                    action = Some(Action::UnclaimIssue(detail.id.clone()));
                }
            } else if ui.small_button("Claim").clicked() {
                action = Some(Action::ClaimIssue(detail.id.clone()));
            }
            if ui.small_button("Close").clicked() {
                action = Some(Action::CloseIssue(detail.id.clone()));
            }
        });
    });
    if ops.busy {
        ui.horizontal(|ui| {
            ui.add(egui::Spinner::new());
            ui.label(
                egui::RichText::new("Working…")
                    .small()
                    .italics()
                    .color(theme::dim_text()),
            );
        });
    }
    ui.horizontal(|ui| {
        ui.label(egui::RichText::new("by").small().color(theme::dim_text()));
        let by_resp = ui.add(
            egui::TextEdit::singleline(ops.by)
                .hint_text("fabrik")
                .desired_width(120.0),
        );
        if by_resp.changed() {
            action = Some(Action::SetOpBy((*ops.by).clone()));
        }
    });
    let reason_resp = ui.add(
        egui::TextEdit::singleline(ops.close_reason)
            .hint_text("close reason — `--reason …`")
            .desired_width(f32::INFINITY),
    );
    if reason_resp.changed() {
        action = Some(Action::SetCloseReason((*ops.close_reason).clone()));
    }
    let mut force = *ops.force;
    if ui.checkbox(&mut force, "force").changed() {
        action = Some(Action::ToggleWriteForce);
    }
    if let Some(message) = ops.error.clone() {
        ui.label(egui::RichText::new(message).small().color(theme::sys_red()));
    }
    action
}

fn render_relationships_section(ui: &mut Ui, detail: &IssueDetail) {
    sidebar_section_header(ui, "RELATIONSHIPS");
    sidebar_prop_row(ui, "Dependencies", |ui| {
        let count = detail.dependencies.len().max(detail.dependency_count);
        badge::badge(ui, &count.to_string());
    });
    sidebar_prop_row(ui, "Dependents", |ui| {
        let count = detail.dependents.len().max(detail.dependent_count);
        badge::badge(ui, &count.to_string());
    });
    sidebar_prop_row(ui, "Comments", |ui| {
        badge::badge(ui, &detail.comment_count.to_string());
    });
}

fn render_timeline_section(ui: &mut Ui, detail: &IssueDetail) {
    if detail.created_at.is_none()
        && detail.started_at.is_none()
        && detail.updated_at.is_none()
        && detail.closed_at.is_none()
    {
        return;
    }

    ui.add_space(theme::SPACING);
    ui.painter().hline(
        ui.max_rect().x_range(),
        ui.cursor().top(),
        theme::hairline(),
    );

    sidebar_section_header(ui, "TIMELINE");
    if let Some(created) = detail.created_at {
        sidebar_prop_row(ui, "Created", |ui| {
            ui.label(
                egui::RichText::new(format_local(created))
                    .small()
                    .color(theme::foreground()),
            );
        });
    }
    if let Some(started) = detail.started_at {
        sidebar_prop_row(ui, "Started", |ui| {
            ui.label(
                egui::RichText::new(format_local(started))
                    .small()
                    .color(theme::neon_amber()),
            );
        });
    }
    if let Some(updated) = detail.updated_at {
        sidebar_prop_row(ui, "Updated", |ui| {
            ui.label(
                egui::RichText::new(format_local(updated))
                    .small()
                    .color(theme::foreground()),
            );
        });
    }
    if let Some(closed) = detail.closed_at {
        sidebar_prop_row(ui, "Closed", |ui| {
            ui.label(
                egui::RichText::new(format_local(closed))
                    .small()
                    .color(theme::neon_green()),
            );
        });
    }
}

fn sidebar_prop_row(ui: &mut Ui, label: &str, add_val: impl FnOnce(&mut Ui)) {
    ui.horizontal(|ui| {
        ui.add(
            egui::Label::new(egui::RichText::new(label).small().color(theme::dim_text()))
                .selectable(false),
        );
        ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
            add_val(ui);
        });
    });
}

fn sidebar_section_header(ui: &mut Ui, label: &str) {
    ui.add_space(theme::SPACING);
    ui.label(
        egui::RichText::new(label)
            .small()
            .strong()
            .color(theme::dim_text()),
    );
    ui.add_space(theme::SPACING);
}
