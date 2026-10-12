use eframe::egui::{self, ScrollArea, Ui};

use crate::model::{Action, Filter, IssueRow, ViewMode};

use super::{
    chip, chrome, footer, issue_row, kanban, project_setup, project_tabs, theme,
};

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

pub fn board(
    ui: &mut Ui,
    view: &BoardView,
    route: &mut chrome::RouteToolbarView,
) -> Vec<Action> {
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
                .filter(|label| *label != "pick a beads repo")
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
                    ui.label(egui::RichText::new("no issues match").color(theme::dim_text()));
                }
            });
        });
    actions
}
