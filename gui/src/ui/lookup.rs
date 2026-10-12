use eframe::egui::{self, ScrollArea, Ui};

use crate::fabrik::{ExplainResult, GraphEdge, GraphNode, PathResult};
use crate::model::Action;

use super::{chrome, graph_setup, grouped_list, project_setup, theme};

#[allow(clippy::struct_excessive_bools)]
pub struct LookupView<'a> {
    pub active_path: Option<std::path::PathBuf>,
    pub tabs: Vec<crate::model::RecentRepo>,
    pub last_ok_label: String,
    pub live: bool,
    pub issue_count: usize,
    pub repo_path: String,
    pub loading: bool,
    pub stale: bool,
    pub needs_init: bool,
    pub init_in_progress: bool,
    pub git_repo: bool,
    pub show_graph_setup: bool,
    pub graph_setup_rebuild: bool,
    pub graph_setup_busy: bool,
    pub graph_setup_body: String,
    pub nodes: &'a [GraphNode],
    pub explain: Option<&'a ExplainResult>,
    pub path: Option<&'a PathResult>,
    pub lookup_loading: bool,
    pub has_query: bool,
    pub anchor: Option<&'a str>,
}

pub fn lookup_pane(
    ui: &mut Ui,
    view: &LookupView<'_>,
    route: &mut chrome::RouteToolbarView,
) -> Vec<Action> {
    let (actions, ()) = chrome::pane_shell(
        ui,
        &chrome::PaneFooterProps {
            live: view.live,
            loading: view.loading || view.lookup_loading,
            stale: view.stale,
            updated: view.last_ok_label.clone(),
            project: view.repo_path.clone(),
            issues: view.issue_count,
        },
        |ui| {
            super::project_tabs::project_tabs(
                ui,
                &super::project_tabs::ProjectTabsProps {
                    tabs: view.tabs.clone(),
                    active: view.active_path.clone(),
                    loading: view.loading,
                },
            )
        },
        |ui| chrome::route_toolbar(ui, route),
        |ui| {
            let mut actions = Vec::new();
            ui.add_space(theme::SPACING);
            if view.needs_init {
                actions.extend(project_setup::project_setup_banner(
                    ui,
                    &project_setup::ProjectSetupProps {
                        path: view.repo_path.clone(),
                        git: view.git_repo,
                        busy: view.init_in_progress,
                    },
                ));
                return (actions, ());
            }
            if view.show_graph_setup {
                actions.extend(graph_setup::graph_setup_banner(
                    ui,
                    &graph_setup::GraphSetupProps {
                        rebuild: view.graph_setup_rebuild,
                        busy: view.graph_setup_busy,
                        body: view.graph_setup_body.clone(),
                    },
                ));
                return (actions, ());
            }
            if view.lookup_loading && view.path.is_none() && view.explain.is_none() {
                ui.add(egui::Spinner::new());
                return (actions, ());
            }
            ScrollArea::vertical()
                .id_salt("lookup_scroll")
                .auto_shrink([false, false])
                .show(ui, |ui| {
                    theme::fill_width(ui);
                    if let Some(path) = view.path {
                        path_panel(ui, path, view.anchor, &mut actions);
                        return;
                    }
                    if let Some(explain) = view.explain {
                        explain_panel(ui, explain, &mut actions);
                        return;
                    }
                    if view.nodes.is_empty() {
                        if view.has_query {
                            ui.label(
                                egui::RichText::new("no matches")
                                    .font(theme::font_callout())
                                    .color(theme::text_secondary()),
                            );
                        }
                        return;
                    }
                    node_list(ui, view.nodes, view.anchor, &mut actions);
                });
            (actions, ())
        },
    );
    actions
}

fn node_list(
    ui: &mut Ui,
    nodes: &[GraphNode],
    anchor: Option<&str>,
    actions: &mut Vec<Action>,
) {
    theme::grouped_inset_frame().show(ui, |ui| {
        for (index, node) in nodes.iter().enumerate() {
            let loc = format!("{}:{}", node.path, node.line);
            let subtitle = format!("{} · {}", node.kind, loc);
            let selected = anchor == Some(node.id.as_str());
            let response = grouped_list::grouped_row(
                ui,
                &grouped_list::GroupedRowProps {
                    title: node.name.clone(),
                    subtitle: Some(subtitle),
                    trailing: None,
                    selected,
                    show_separator: index + 1 < nodes.len(),
                },
            );
            if response.secondary_clicked() {
                ui.ctx().copy_text(loc);
            }
            if response.clicked() {
                actions.push(Action::LookupSelectNode(Some(node.id.clone())));
            }
        }
    });
}

fn explain_panel(ui: &mut Ui, explain: &ExplainResult, actions: &mut Vec<Action>) {
    theme::card_frame().show(ui, |ui| {
        let loc = format!("{}:{}", explain.node.path, explain.node.line);
        ui.horizontal(|ui| {
            ui.label(
                egui::RichText::new(&explain.node.name)
                    .font(theme::font_headline())
                    .color(theme::text_primary()),
            );
            if theme::pointer(ui.small_button(loc.clone())).clicked() {
                ui.ctx().copy_text(loc.clone());
            }
        });
        ui.label(
            egui::RichText::new(format!("{} · {}", explain.node.kind, loc))
                .font(theme::font_footnote())
                .color(theme::text_secondary()),
        );
        if explain.truncated {
            ui.label(
                egui::RichText::new("neighborhood truncated")
                    .font(theme::font_caption2())
                    .color(theme::text_tertiary()),
            );
        }
        ui.add_space(theme::SPACING);
        for edge in &explain.edges {
            edge_row(ui, edge);
        }
        ui.add_space(theme::SPACING);
        if ui.small_button("clear selection").clicked() {
            actions.push(Action::LookupClearPath);
        }
    });
}

fn edge_row(ui: &mut Ui, edge: &GraphEdge) {
    let target = if edge.dst.is_empty() {
        edge.label.clone()
    } else {
        format!("{} → {}", edge.label, edge.dst)
    };
    ui.label(
        egui::RichText::new(target)
            .font(theme::font_footnote())
            .color(theme::text_secondary()),
    );
}

fn path_panel(
    ui: &mut Ui,
    path: &PathResult,
    anchor: Option<&str>,
    actions: &mut Vec<Action>,
) {
    theme::card_frame().show(ui, |ui| {
        if path.nodes.is_empty() {
            ui.label(
                egui::RichText::new("no path")
                    .font(theme::font_callout())
                    .color(theme::text_secondary()),
            );
        } else {
            for (index, node) in path.nodes.iter().enumerate() {
                let loc = format!("{}:{}", node.path, node.line);
                let selected = anchor == Some(node.id.as_str());
                let response = grouped_list::grouped_row(
                    ui,
                    &grouped_list::GroupedRowProps {
                        title: node.name.clone(),
                        subtitle: Some(loc.clone()),
                        trailing: None,
                        selected,
                        show_separator: index + 1 < path.nodes.len(),
                    },
                );
                if response.secondary_clicked() {
                    ui.ctx().copy_text(loc);
                }
                if response.clicked() {
                    actions.push(Action::LookupSelectNode(Some(node.id.clone())));
                }
            }
        }
        if ui.small_button("back to list").clicked() {
            actions.push(Action::LookupClearPath);
        }
    });
}
