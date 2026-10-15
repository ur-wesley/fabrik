use eframe::egui::{self, RichText, ScrollArea, Ui};

use crate::fabrik::{ExplainResult, GraphNode, PathResult};
use crate::model::Action;

use super::{button, chrome, graph_view, grouped_list, project_setup, theme};

#[allow(clippy::struct_excessive_bools)]
pub struct LookupView<'a> {
    pub active_path: Option<std::path::PathBuf>,
    pub tabs: Vec<crate::model::RecentRepo>,
    pub last_ok_label: String,
    pub live: bool,
    pub issue_count: usize,
    pub repo_path: String,
    /// Combined busy flag: board + lookup + status + inspect + build.
    /// This is the single loading indicator, shown in the footer only.
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
    pub has_query: bool,
    pub anchor: Option<&'a str>,
    pub context: Option<&'a crate::fabrik::GraphContextResult>,
    pub context_symbol: Option<&'a str>,
    pub impact: Option<&'a crate::fabrik::GraphImpactResult>,
    pub impact_symbol: Option<&'a str>,
    pub impact_depth: u32,
    pub detect: Option<&'a crate::fabrik::DetectChangesResult>,
    pub detect_ref: Option<&'a str>,
    pub issue_seed: Option<&'a str>,
    pub stale_files: i32,
    pub status_indexed: bool,
    pub status_files: i32,
    pub status_symbols: i32,
    pub status_last_build_at: String,
    pub explore: Option<&'a crate::fabrik::ExploreResult>,
    pub graph_columns: Vec<(String, Vec<GraphNode>)>,
    pub graph_edges: Vec<(String, String, String)>,
    /// Persistent instant-overview failure (full message plus binary path).
    /// Rendered as an error card with Retry until an overview succeeds.
    pub overview_error: Option<String>,
    /// Budget the Retry button refetches with (`AppState::OVERVIEW_BUDGET`).
    pub overview_budget: u32,
}

/// Right sidebar width: canvas takes the central remainder.
const SIDEBAR_W: f32 = 300.0;
/// Minimum canvas width before the sidebar squeezes it.
const CANVAS_MIN_W: f32 = 160.0;
/// Minimum split height so the canvas stays usable in tiny windows.
const SPLIT_MIN_H: f32 = 160.0;

pub fn lookup_pane(
    ui: &mut Ui,
    view: &LookupView<'_>,
    route: &mut chrome::RouteToolbarView,
) -> Vec<Action> {
    let (actions, ()) = chrome::pane_shell(
        ui,
        &chrome::PaneFooterProps {
            live: view.live,
            loading: view.loading,
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
            actions.extend(super::lookup_header::header_strip(ui, view));
            actions.extend(overview_error_card(ui, view));
            ui.add_space(theme::SPACING);
            // Center + sidebar: the canvas takes the central remainder at
            // full pane height OUTSIDE any ScrollArea (it owns the wheel
            // for zoom); the fixed-width sidebar scrolls independently.
            let pane_h = ui.available_height().max(SPLIT_MIN_H);
            ui.horizontal_top(|ui| {
                let gap = ui.spacing().item_spacing.x;
                let canvas_w = (ui.available_width() - SIDEBAR_W - gap).max(CANVAS_MIN_W);
                ui.allocate_ui_with_layout(
                    egui::vec2(canvas_w, pane_h),
                    egui::Layout::top_down(egui::Align::Min),
                    |ui| {
                        actions.extend(graph_view::graph_canvas(
                            ui,
                            &view.graph_columns,
                            &view.graph_edges,
                            view.anchor,
                        ));
                    },
                );
                ui.allocate_ui_with_layout(
                    egui::vec2(SIDEBAR_W, pane_h),
                    egui::Layout::top_down(egui::Align::Min),
                    |ui| {
                        ScrollArea::vertical()
                            .id_salt("lookup_sidebar")
                            .auto_shrink([false, false])
                            .show(ui, |ui| {
                                theme::fill_width(ui);
                                super::lookup_sidebar::sidebar_sections(ui, view, &mut actions);
                            });
                    },
                );
            });
            (actions, ())
        },
    );
    actions
}

fn overview_error_card(ui: &mut Ui, view: &LookupView<'_>) -> Vec<Action> {
    let mut actions = Vec::new();
    let Some(error) = view.overview_error.clone() else {
        return actions;
    };
    theme::card_frame().show(ui, |ui| {
        head(ui, "instant overview failed".to_owned());
        if crate::lookup::is_unknown_command(&error) {
            foot(
                ui,
                "CLI too old for instant overview (needs `graph overview`) — rebuild CLI or set FABRIK_BIN to a fresh build.".to_owned(),
            );
        }
        cap(ui, error);
        ui.add_space(theme::SPACING);
        if let Some(repo) = view.active_path.clone() {
            let btn = button::button(
                ui,
                &button::ButtonProps::new("Retry overview")
                    .variant(button::ButtonVariant::Secondary)
                    .min_size(egui::vec2(180.0, theme::CONTROL_H)),
            );
            if btn.clicked() {
                actions.push(Action::FetchGraphOverview {
                    repo,
                    budget: view.overview_budget,
                });
            }
        }
    });
    actions
}

pub(super) fn head(ui: &mut Ui, text: String) {
    ui.label(head_rich(text));
}

pub(super) fn foot(ui: &mut Ui, text: String) {
    let rich = RichText::new(text).font(theme::font_footnote());
    ui.label(rich.color(theme::text_secondary()));
}

pub(super) fn cap(ui: &mut Ui, text: String) {
    let rich = RichText::new(text).font(theme::font_caption2());
    ui.label(rich.color(theme::text_tertiary()));
}

pub(super) fn section(ui: &mut Ui, text: &str) {
    foot(ui, text.to_owned());
}

fn head_rich(text: String) -> RichText {
    let rich = RichText::new(text).font(theme::font_headline());
    rich.color(theme::text_primary())
}

pub(super) fn edge_row(ui: &mut Ui, edge: &crate::fabrik::GraphEdge) {
    let text = if edge.dst.is_empty() {
        if edge.label.is_empty() {
            format!("{} · {}", edge.src, edge.kind)
        } else {
            format!("{} · {} · {}", edge.src, edge.kind, edge.label)
        }
    } else if edge.label.is_empty() {
        format!("{} · {} → {}", edge.src, edge.kind, edge.dst)
    } else {
        format!(
            "{} · {} · {} → {}",
            edge.src, edge.kind, edge.label, edge.dst
        )
    };
    foot(ui, text);
}

#[allow(clippy::too_many_arguments)]
pub(super) fn node_row(
    ui: &mut Ui,
    title: String,
    subtitle: String,
    copy_text: String,
    id: &str,
    anchor: Option<&str>,
    last: bool,
    actions: &mut Vec<Action>,
) {
    let response = grouped_list::grouped_row(
        ui,
        &grouped_list::GroupedRowProps {
            title,
            subtitle: Some(subtitle),
            trailing: None,
            selected: anchor == Some(id),
            show_separator: !last,
        },
    );
    if response.secondary_clicked() {
        ui.ctx().copy_text(copy_text);
    }
    if response.clicked() {
        actions.push(Action::LookupSelectNode(Some(id.to_owned())));
    }
}
