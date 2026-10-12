use std::collections::BTreeMap;

use eframe::egui::{self, ScrollArea, Ui};
use egui_commonmark::CommonMarkCache;

use crate::fabrik::{MemContext, MemSession, Observation};
use crate::model::Action;

use super::{badge, chrome, grouped_list, theme, toggle};

#[allow(clippy::struct_excessive_bools)]
pub struct MemoryView<'a> {
    pub active_path: Option<std::path::PathBuf>,
    pub tabs: Vec<crate::model::RecentRepo>,
    pub last_ok_label: String,
    pub live: bool,
    pub issue_count: usize,
    pub repo_path: String,
    pub loading: bool,
    pub stale: bool,
    pub context: Option<&'a MemContext>,
    pub results: &'a [Observation],
    pub mem_loading: bool,
    pub mem_searching: bool,
    pub filter_project: bool,
    pub repo_selected: bool,
    pub detail: Option<&'a Observation>,
    pub detail_loading: bool,
    pub selected_id: Option<&'a str>,
}

pub fn memory_pane(
    ui: &mut Ui,
    view: &MemoryView<'_>,
    route: &mut chrome::RouteToolbarView,
    markdown: &mut CommonMarkCache,
) -> Vec<Action> {
    let (actions, ()) = chrome::pane_shell(
        ui,
        &chrome::PaneFooterProps {
            live: view.live,
            loading: view.loading || view.mem_loading,
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
            ui.horizontal(|ui| {
                if let Some(on) = toggle::toggle(
                    ui,
                    &toggle::ToggleProps {
                        on: view.filter_project,
                        label: "this project",
                        enabled: view.repo_selected,
                    },
                ) {
                    if on != view.filter_project {
                        actions.push(Action::ToggleMemProjectFilter);
                    }
                }
            });
            ui.add_space(theme::SPACING);
            if view.mem_loading && view.context.is_none() && view.results.is_empty() {
                ui.add(egui::Spinner::new());
                return (actions, ());
            }
            let searching = view.mem_searching;
            let rows: Vec<&Observation> = if searching {
                view.results.iter().collect()
            } else {
                view.context
                    .map(|c| c.observations.iter().collect())
                    .unwrap_or_default()
            };
            ScrollArea::vertical()
                .id_salt("memory_scroll")
                .auto_shrink([false, false])
                .show(ui, |ui| {
                    theme::fill_width(ui);
                    let bg = ui.max_rect();
                    ui.painter()
                        .rect_filled(bg, theme::radius_card(), theme::grouped_bg());
                    if !searching {
                        if let Some(ctx) = view.context {
                            if let Some(session) = &ctx.session {
                                session_card(ui, session);
                                ui.add_space(theme::SPACING);
                            }
                        }
                    }
                    if rows.is_empty() {
                        ui.label(
                            egui::RichText::new("no observations")
                                .font(theme::font_footnote())
                                .color(theme::text_tertiary()),
                        );
                    } else {
                        grouped_observations(ui, &rows, view.selected_id, &mut actions);
                    }
                    if let Some(ctx) = view.context {
                        ui.add_space(theme::SPACING);
                        ui.label(
                            egui::RichText::new(format!(
                                "{} observations · {} sessions",
                                ctx.stats.observation_count,
                                ctx.stats.session_count,
                            ))
                            .font(theme::font_footnote())
                            .color(theme::text_secondary()),
                        );
                    }
                    if view.detail_loading {
                        ui.add_space(theme::SPACING);
                        ui.add(egui::Spinner::new());
                    } else if let Some(detail) = view.detail {
                        ui.add_space(theme::SPACING);
                        observation_detail(ui, detail, markdown, &mut actions);
                    }
                });
            (actions, ())
        },
    );
    actions
}

fn session_card(ui: &mut Ui, session: &MemSession) {
    theme::card_frame().show(ui, |ui| {
        ui.label(
            egui::RichText::new("Session")
                .font(theme::font_title2())
                .color(theme::text_primary()),
        );
        ui.add_space(4.0);
        ui.label(
            egui::RichText::new(&session.id)
                .font(theme::font_footnote())
                .color(theme::text_secondary()),
        );
        if !session.goal.is_empty() {
            ui.label(
                egui::RichText::new(&session.goal)
                    .font(theme::font_large_title())
                    .color(theme::text_primary()),
            );
        }
        for (label, value) in [
            ("discoveries", &session.discoveries),
            ("accomplished", &session.accomplished),
            ("next steps", &session.next_steps),
            ("files", &session.files),
        ] {
            if !value.is_empty() {
                ui.label(
                    egui::RichText::new(label)
                        .font(theme::font_caption2())
                        .color(theme::text_tertiary()),
                );
                ui.label(
                    egui::RichText::new(value.as_str())
                        .font(theme::font_footnote())
                        .color(theme::text_secondary()),
                );
            }
        }
        let times = format!(
            "{} → {}",
            session.started_at,
            session.ended_at.as_deref().unwrap_or("…")
        );
        ui.label(
            egui::RichText::new(times)
                .font(theme::font_caption2())
                .color(theme::text_quaternary()),
        );
    });
}

fn grouped_observations(
    ui: &mut Ui,
    rows: &[&Observation],
    selected_id: Option<&str>,
    actions: &mut Vec<Action>,
) {
    let mut groups: BTreeMap<String, Vec<&Observation>> = BTreeMap::new();
    for row in rows {
        let key = if row.kind.is_empty() {
            "other".to_owned()
        } else {
            row.kind.clone()
        };
        groups.entry(key).or_default().push(row);
    }
    theme::grouped_inset_frame().show(ui, |ui| {
        let mut first_group = true;
        for (kind, items) in groups {
            if !first_group {
                ui.add_space(6.0);
            }
            first_group = false;
            ui.label(
                egui::RichText::new(kind.to_uppercase())
                    .font(theme::font_caption2())
                    .color(theme::text_tertiary()),
            );
            for (index, row) in items.iter().enumerate() {
                let body_preview = row.body.chars().take(120).collect::<String>();
                let subtitle = if body_preview.is_empty() {
                    row.project.clone()
                } else {
                    format!("{}\n{}", row.project, body_preview)
                };
                let trailing = row.created_at.clone();
                let selected = selected_id == Some(row.id.as_str());
                let title = if row.title.is_empty() {
                    row.id.clone()
                } else {
                    row.title.clone()
                };
                let response = grouped_list::grouped_row(
                    ui,
                    &grouped_list::GroupedRowProps {
                        title,
                        subtitle: Some(subtitle),
                        trailing: Some(trailing),
                        selected,
                        show_separator: index + 1 < items.len(),
                    },
                );
                if response.clicked() {
                    actions.push(Action::MemSelect(Some(row.id.clone())));
                }
            }
        }
    });
}

fn observation_detail(
    ui: &mut Ui,
    obs: &Observation,
    markdown: &mut CommonMarkCache,
    actions: &mut Vec<Action>,
) {
    theme::card_frame().show(ui, |ui| {
        ui.horizontal(|ui| {
            badge::kind_badge(ui, &obs.kind);
            ui.label(
                egui::RichText::new(&obs.title)
                    .font(theme::font_headline())
                    .color(theme::text_primary()),
            );
            ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
                if theme::pointer(ui.small_button("close")).clicked() {
                    actions.push(Action::MemClearDetail);
                }
            });
        });
        ui.label(
            egui::RichText::new(format!("{} · {}", obs.project, obs.created_at))
                .font(theme::font_footnote())
                .color(theme::text_secondary()),
        );
        ui.add_space(theme::SPACING);
        egui_commonmark::CommonMarkViewer::new().show(ui, markdown, &obs.body);
    });
}
