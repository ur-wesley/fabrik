use eframe::egui::{self, ScrollArea, Ui};

use crate::model::{Action, Filter, IssueRow, ViewMode};

use super::{chip, chrome, footer, issue_row, kanban, project_setup, project_tabs, theme};

pub struct CreateForm<'a> {
    pub title: &'a mut String,
    pub body: &'a mut String,
    pub acceptance: &'a mut String,
    pub priority: &'a mut u8,
    pub kind: &'a mut String,
    pub busy: bool,
    pub error: Option<String>,
}

pub fn can_submit_create(title: &str, busy: bool) -> bool {
    !busy && !title.trim().is_empty()
}

pub fn create_dialog(ui: &mut Ui, form: &mut CreateForm<'_>) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.set_min_width(560.0);
    ui.set_max_width(600.0);
    ui.label(
        egui::RichText::new("New issue")
            .size(18.0)
            .strong()
            .color(theme::foreground()),
    );
    ui.add_space(theme::SPACING);

    let title_resp = ui.add(
        egui::TextEdit::singleline(form.title)
            .hint_text("Title (required)")
            .desired_width(f32::INFINITY),
    );
    if title_resp.changed() {
        actions.push(Action::SetCreateTitle(form.title.clone()));
    }
    ui.add_space(theme::SPACING);
    let body_resp = ui.add(
        egui::TextEdit::multiline(form.body)
            .hint_text("Body — `fabrik issue create --body …`")
            .desired_rows(3)
            .desired_width(f32::INFINITY),
    );
    if body_resp.changed() {
        actions.push(Action::SetCreateBody(form.body.clone()));
    }
    ui.add_space(theme::SPACING);
    let acc_resp = ui.add(
        egui::TextEdit::multiline(form.acceptance)
            .hint_text("Acceptance criteria — `--acceptance …`")
            .desired_rows(2)
            .desired_width(f32::INFINITY),
    );
    if acc_resp.changed() {
        actions.push(Action::SetCreateAcceptance(form.acceptance.clone()));
    }
    ui.add_space(theme::SPACING);

    ui.horizontal(|ui| {
        ui.label(
            egui::RichText::new("Priority")
                .small()
                .color(theme::dim_text()),
        );
        for priority in 0..=4u8 {
            if ui
                .selectable_value(form.priority, priority, priority.to_string())
                .changed()
            {
                actions.push(Action::SetCreatePriority(*form.priority));
            }
        }
    });
    ui.add_space(4.0);
    ui.horizontal(|ui| {
        ui.label(egui::RichText::new("Kind").small().color(theme::dim_text()));
        for kind in crate::fabrik::ISSUE_KINDS {
            if ui
                .selectable_value(form.kind, kind.to_owned(), kind)
                .changed()
            {
                actions.push(Action::SetCreateKind(form.kind.clone()));
            }
        }
    });
    ui.add_space(theme::SPACING);

    if let Some(message) = form.error.clone() {
        ui.label(egui::RichText::new(message).small().color(theme::sys_red()));
        ui.add_space(theme::SPACING);
    }

    ui.horizontal(|ui| {
        if form.busy {
            ui.add(egui::Spinner::new());
            ui.label(
                egui::RichText::new("Creating…")
                    .small()
                    .italics()
                    .color(theme::dim_text()),
            );
        } else {
            if ui.button("Cancel").clicked() {
                actions.push(Action::CloseCreate);
            }
            let create = ui.add_enabled(
                can_submit_create(form.title, false),
                egui::Button::new(egui::RichText::new("Create issue").small().strong()),
            );
            if create.clicked() {
                actions.push(Action::SubmitCreate);
            }
            ui.label(
                egui::RichText::new("priority 0-4 · `--kind task|bug|feature|epic|chore`")
                    .small()
                    .color(theme::dim_text()),
            );
        }
    });
    actions
}

#[allow(clippy::struct_excessive_bools)]
pub struct BoardView {
    pub current_repo: Option<String>,
    pub active_path: Option<std::path::PathBuf>,
    pub tabs: Vec<crate::model::RecentRepo>,
    pub last_ok_label: String,
    pub live: bool,
    pub view_mode: ViewMode,
    pub filter: Filter,
    pub selected: usize,
    pub search: String,
    pub rows: Vec<IssueRow>,
    pub lanes: Vec<crate::model::KanbanLane>,
    pub collapsed: Vec<Filter>,
    pub repo_path: String,
    pub issue_count: usize,
    pub loading: bool,
    pub stale: bool,
    pub needs_init: bool,
    pub init_in_progress: bool,
    pub git_repo: bool,
}

pub fn board(ui: &mut Ui, view: &BoardView, route: &mut chrome::RouteToolbarView) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.with_layout(egui::Layout::bottom_up(egui::Align::Min), |ui| {
        ui.spacing_mut().item_spacing.y = 0.0;
        footer::footer(
            ui,
            &footer::FooterProps {
                live: view.live,
                loading: view.loading,
                stale: view.stale,
                updated: view.last_ok_label.clone(),
                project: view.repo_path.clone(),
                issues: view.issue_count,
            },
        );
        egui::Frame::new()
            .fill(egui::Color32::TRANSPARENT)
            .inner_margin(egui::Margin::ZERO)
            .show(ui, |ui| {
                // Content stacks top-down (tabs -> header -> body). The outer
                // `bottom_up` layout only pins the footer; inheriting it here
                // would stack the content in reverse and overlap the sections.
                ui.with_layout(egui::Layout::top_down(egui::Align::Min), |ui| {
                    ui.spacing_mut().item_spacing.y = 0.0;
                    actions.extend(project_tabs::project_tabs(
                        ui,
                        &project_tabs::ProjectTabsProps {
                            tabs: view.tabs.clone(),
                            active: view.active_path.clone(),
                            loading: view.loading,
                        },
                    ));
                    egui::Frame::new()
                        .fill(egui::Color32::TRANSPARENT)
                        .inner_margin(egui::Margin {
                            left: theme::PAD_I8,
                            right: theme::PAD_I8,
                            top: theme::SPACING_I8,
                            bottom: 0,
                        })
                        .show(ui, |ui| {
                            actions.extend(chrome::route_toolbar(ui, route));
                        });
                    egui::Frame::new()
                        .fill(egui::Color32::TRANSPARENT)
                        .inner_margin(egui::Margin {
                            left: theme::PAD_I8,
                            right: theme::PAD_I8,
                            top: theme::SPACING_I8,
                            bottom: theme::PAD_I8,
                        })
                        .show(ui, |ui| {
                            ui.spacing_mut().item_spacing.y = 0.0;
                            if view.needs_init {
                                ui.add_space(theme::SPACING);
                                actions.extend(project_setup::project_setup_banner(
                                    ui,
                                    &project_setup::ProjectSetupProps {
                                        path: view.repo_path.clone(),
                                        git: view.git_repo,
                                        busy: view.init_in_progress,
                                    },
                                ));
                            }
                            ui.add_space(6.0);
                            theme::remaining_body(ui, |ui| {
                                if view.needs_init {
                                    return;
                                }
                                if view.loading && view.issue_count == 0 {
                                    loading_body(ui, view);
                                } else if view.view_mode == ViewMode::List {
                                    actions.extend(list_body(ui, view));
                                } else {
                                    actions.extend(kanban::kanban(
                                        ui,
                                        &kanban::KanbanView {
                                            lanes: view.lanes.clone(),
                                            collapsed: view.collapsed.clone(),
                                        },
                                    ));
                                }
                            });
                        });
                });
            });
    });
    actions
}

fn loading_body(ui: &mut Ui, view: &BoardView) {
    ui.allocate_ui_with_layout(
        ui.available_size(),
        egui::Layout::top_down(egui::Align::Center),
        |ui| {
            ui.add_space(120.0);
            ui.add(egui::Spinner::new().size(24.0));
            ui.add_space(theme::SPACING);
            let name = view
                .current_repo
                .as_deref()
                .filter(|label| *label != "pick a fabrik repo")
                .unwrap_or(view.repo_path.as_str());
            ui.label(egui::RichText::new(format!("loading {name}…")).color(theme::dim_text()));
        },
    );
}

fn list_body(ui: &mut Ui, view: &BoardView) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.horizontal_wrapped(|ui| {
        for filter in crate::model::ALL_LANES {
            if let Some(picked) = chip::chip(
                ui,
                &chip::ChipProps {
                    filter,
                    active: view.filter == filter,
                    label: filter.label(),
                },
            ) {
                actions.push(Action::SetFilter(picked));
            }
        }
    });
    ui.add_space(theme::SPACING);
    let body_h = ui.available_height();
    ScrollArea::vertical()
        .id_salt("issue_list")
        .auto_shrink([false, false])
        .max_height(body_h)
        .show(ui, |ui| {
            theme::fill_width(ui);
            ui.vertical(|ui| {
                theme::fill_width(ui);
                for (index, row) in view.rows.iter().enumerate() {
                    if let Some(id) = issue_row::issue_row(
                        ui,
                        &issue_row::IssueRowProps {
                            row: row.clone(),
                            selected: index == view.selected,
                        },
                    ) {
                        actions.push(Action::OpenDetail(id));
                    }
                    ui.add_space(theme::SPACING);
                }
                if view.rows.is_empty() {
                    if view.search.trim().is_empty() && view.issue_count == 0 {
                        ui.label(egui::RichText::new("No issues yet").color(theme::dim_text()));
                        ui.label(
                            egui::RichText::new(
                                "Create one with + New, or `fabrik issue create \"title\" --json`.",
                            )
                            .small()
                            .color(theme::dim_text()),
                        );
                    } else {
                        ui.label(egui::RichText::new("no issues match").color(theme::dim_text()));
                    }
                }
            });
        });
    actions
}

#[cfg(test)]
mod tests {
    use super::can_submit_create;

    #[test]
    fn create_submit_gate() {
        assert!(can_submit_create("title", false));
        assert!(!can_submit_create("   ", false));
        assert!(!can_submit_create("", false));
        assert!(!can_submit_create("title", true));
    }
}
