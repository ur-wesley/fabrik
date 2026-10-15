use eframe::egui::{self, ScrollArea, Ui};
use egui_commonmark::CommonMarkCache;

use crate::fabrik::{
    ContextStats, CurrentProjectResult, MemCheckResult, MemContext, MemSession, MemTimelineResult,
    Observation,
};
use crate::model::Action;

use super::mem_form::MemForm;
use super::{button, chrome, mem_form, memory_row, theme, toggle};

pub use super::memory_detail::detail_sheet;

#[allow(clippy::struct_excessive_bools)]
#[allow(dead_code)]
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
    pub project_label: Option<String>,
    pub project_filter_enabled: bool,
    pub project_filter_note: Option<String>,
    pub detail: Option<&'a Observation>,
    pub detail_loading: bool,
    pub selected_id: Option<&'a str>,
    pub check: Option<&'a MemCheckResult>,
    pub stats: Option<&'a ContextStats>,
    pub timeline: Option<&'a MemTimelineResult>,
    pub timeline_id: Option<&'a str>,
    pub current_project: Option<&'a CurrentProjectResult>,
    pub diagnostics_loading: bool,
    pub diagnostics_summary: String,
    pub form_open: bool,
    pub editing: Option<&'a str>,
    pub write_busy: bool,
    pub write_error: Option<String>,
    pub delete_confirm: Option<String>,
    pub prompt_open: bool,
    pub kind_filter: Option<String>,
    pub kinds: Vec<String>,
    pub diagnostics_open: bool,
}

#[allow(clippy::needless_pass_by_ref_mut)]
pub fn memory_pane(
    ui: &mut Ui,
    view: &MemoryView<'_>,
    route: &mut chrome::RouteToolbarView,
    _markdown: &mut CommonMarkCache,
    form: &mut MemForm,
) -> Vec<Action> {
    let _ = form;
    let (actions, ()) = chrome::pane_shell(
        ui,
        &chrome::PaneFooterProps {
            live: view.live,
            loading: view.loading || view.mem_loading || view.write_busy,
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
            ui.spacing_mut().item_spacing.y = 0.0;
            let actions = theme::remaining_body(ui, |ui| {
                if view.mem_loading && view.context.is_none() && view.results.is_empty() {
                    loading_body(ui, view);
                    return Vec::new();
                }
                list_body(ui, view)
            });
            (actions, ())
        },
    );
    actions
}

fn loading_body(ui: &mut Ui, view: &MemoryView<'_>) {
    ui.allocate_ui_with_layout(
        ui.available_size(),
        egui::Layout::top_down(egui::Align::Center),
        |ui| {
            ui.add_space(120.0);
            ui.add(egui::Spinner::new().size(24.0));
            ui.add_space(theme::SPACING);
            ui.label(
                egui::RichText::new(format!("loading {}…", view.repo_path))
                    .color(theme::dim_text()),
            );
        },
    );
}

fn list_body(ui: &mut Ui, view: &MemoryView<'_>) -> Vec<Action> {
    let mut actions = Vec::new();
    actions.extend(mem_form::write_row(ui, view));
    ui.add_space(theme::SPACING);
    actions.extend(kind_chip_row(ui, view));
    ui.add_space(theme::SPACING);
    actions.extend(filter_diagnostics_row(ui, view));
    if view.diagnostics_open {
        ui.add_space(4.0);
        diagnostics_details(ui, view, &mut actions);
    }
    let searching = view.mem_searching;
    if !searching
        && let Some(ctx) = view.context
        && let Some(session) = &ctx.session
    {
        ui.add_space(theme::SPACING);
        session_banner(ui, session);
    }
    ui.add_space(theme::SPACING);
    let rows: Vec<&Observation> = if searching {
        view.results.iter().collect()
    } else {
        view.context
            .map(|c| c.observations.iter().collect())
            .unwrap_or_default()
    };
    let filtered: Vec<&Observation> = rows
        .into_iter()
        .filter(|obs| view.kind_filter.as_ref().is_none_or(|k| &obs.kind == k))
        .collect();
    let body_h = ui.available_height();
    ScrollArea::vertical()
        .id_salt("memory_list")
        .auto_shrink([false, false])
        .max_height(body_h)
        .show(ui, |ui| {
            theme::fill_width(ui);
            ui.vertical(|ui| {
                theme::fill_width(ui);
                for row in &filtered {
                    let selected = view.selected_id == Some(row.id.as_str());
                    if memory_row::memory_row(
                        ui,
                        &memory_row::MemoryRowProps {
                            obs: (*row).clone(),
                            selected,
                        },
                    ) {
                        actions.push(Action::MemSelect(Some(row.id.clone())));
                    }
                    ui.add_space(theme::SPACING);
                }
                if filtered.is_empty() {
                    if searching || view.kind_filter.is_some() {
                        ui.label(
                            egui::RichText::new("no observations match")
                                .color(theme::dim_text()),
                        );
                    } else {
                        ui.label(
                            egui::RichText::new("No observations yet").color(theme::dim_text()),
                        );
                        ui.label(
                            egui::RichText::new(
                                "Save one with + New, or `fabrik mem save --title <t> --body <b> --json`.",
                            )
                            .small()
                            .color(theme::dim_text()),
                        );
                        // Failed loads land here (`mem_loading` false, no
                        // context, no results) instead of the spinner above,
                        // so offer an explicit retry rather than nothing.
                        if !view.mem_loading {
                            ui.add_space(theme::SPACING);
                            if button::button(
                                ui,
                                &button::ButtonProps::new("Retry")
                                    .variant(button::ButtonVariant::Outline)
                                    .min_size(egui::vec2(96.0, theme::CONTROL_H)),
                            )
                            .clicked()
                            {
                                actions.push(Action::RefreshMemory);
                                actions.push(Action::RefreshDiagnostics);
                            }
                        }
                    }
                }
            });
        });
    if let Some(ctx) = view.context {
        ui.add_space(theme::SPACING);
        ui.label(
            egui::RichText::new(format!(
                "{} observations · {} sessions",
                ctx.stats.observation_count, ctx.stats.session_count,
            ))
            .font(theme::font_footnote())
            .color(theme::text_secondary()),
        );
    }
    actions
}

fn kind_chip_row(ui: &mut Ui, view: &MemoryView<'_>) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.horizontal_wrapped(|ui| {
        ui.spacing_mut().item_spacing = egui::vec2(theme::SPACING, theme::SPACING);
        let all_active = view.kind_filter.is_none();
        if button::button(
            ui,
            &button::ButtonProps::new("All")
                .variant(button::ButtonVariant::Secondary)
                .selected(all_active)
                .min_size(egui::vec2(72.0, theme::CONTROL_H)),
        )
        .clicked()
        {
            actions.push(Action::SetMemKindFilter(None));
        }
        for kind in &view.kinds {
            let active = view.kind_filter.as_deref() == Some(kind.as_str());
            if button::button(
                ui,
                &button::ButtonProps::new(kind)
                    .variant(button::ButtonVariant::Secondary)
                    .selected(active)
                    .min_size(egui::vec2(72.0, theme::CONTROL_H)),
            )
            .clicked()
            {
                if active {
                    actions.push(Action::SetMemKindFilter(None));
                } else {
                    actions.push(Action::SetMemKindFilter(Some(kind.clone())));
                }
            }
        }
    });
    actions
}

fn filter_diagnostics_row(ui: &mut Ui, view: &MemoryView<'_>) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = theme::SPACING;
        if let Some(on) = toggle::toggle(
            ui,
            &toggle::ToggleProps {
                on: view.filter_project,
                label: "this project",
                enabled: view.project_filter_enabled,
            },
        ) && on != view.filter_project
        {
            actions.push(Action::ToggleMemProjectFilter);
        }
        if let Some(label) = view.project_label.as_deref() {
            ui.label(
                egui::RichText::new(label)
                    .font(theme::font_footnote())
                    .color(theme::text_secondary()),
            );
        } else if let Some(note) = view.project_filter_note.as_deref() {
            ui.label(
                egui::RichText::new(note)
                    .font(theme::font_footnote())
                    .italics()
                    .color(theme::text_tertiary()),
            );
        }
        ui.label(
            egui::RichText::new(view.diagnostics_summary.clone())
                .font(theme::font_caption2())
                .color(theme::text_tertiary()),
        );
        ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
            if view.diagnostics_loading {
                ui.add(egui::Spinner::new());
            } else if button::button(
                ui,
                &button::ButtonProps::new("diagnostics")
                    .variant(button::ButtonVariant::Outline)
                    .selected(view.diagnostics_open)
                    .min_size(egui::vec2(96.0, theme::CONTROL_H)),
            )
            .clicked()
            {
                actions.push(Action::ToggleMemDiagnostics);
            }
        });
    });
    actions
}

fn diagnostics_details(ui: &mut Ui, view: &MemoryView<'_>, actions: &mut Vec<Action>) {
    if let Some(check) = view.check {
        ui.label(
            egui::RichText::new(format!(
                "db {} · schema {} · {} obs · {} sessions",
                check.memory_path, check.schema_memory, check.observations, check.sessions,
            ))
            .font(theme::font_caption2())
            .color(theme::text_tertiary()),
        );
    }
    if let Some(stats) = view.stats {
        ui.label(
            egui::RichText::new(format!(
                "totals {} obs · {} sessions · {} projects",
                stats.observation_count,
                stats.session_count,
                stats.projects.len(),
            ))
            .font(theme::font_caption2())
            .color(theme::text_tertiary()),
        );
    }
    if let Some(current) = view.current_project {
        let label = if current.project.trim().is_empty() {
            format!("project: not a Fabrik repo (source: {})", current.source)
        } else {
            format!("cwd: {} ({})", current.project, current.source)
        };
        ui.label(
            egui::RichText::new(label)
                .font(theme::font_caption2())
                .color(theme::text_tertiary()),
        );
    } else if view.diagnostics_loading {
        ui.label(
            egui::RichText::new("project: resolving…")
                .font(theme::font_caption2())
                .color(theme::text_tertiary()),
        );
    }
    if view.check.is_none() && view.stats.is_none() && view.current_project.is_none() {
        ui.label(
            egui::RichText::new(if view.diagnostics_loading {
                "loading diagnostics…"
            } else {
                "no diagnostics yet"
            })
            .font(theme::font_caption2())
            .color(theme::text_tertiary()),
        );
    }
    ui.horizontal(|ui| {
        if button::button(
            ui,
            &button::ButtonProps::new("refresh")
                .variant(button::ButtonVariant::Ghost)
                .min_size(egui::vec2(72.0, theme::CONTROL_H)),
        )
        .clicked()
        {
            actions.push(Action::RefreshDiagnostics);
        }
    });
}

fn session_banner(ui: &mut Ui, session: &MemSession) {
    let goal = session.goal.trim();
    let headline = if goal.is_empty() {
        let short: String = session.id.chars().take(12).collect();
        let when = memory_row::full_timestamp(&session.started_at);
        if when.is_empty() {
            format!("session {short}")
        } else {
            format!("session {short} · {when}")
        }
    } else {
        session.goal.clone()
    };
    theme::card_frame().show(ui, |ui| {
        ui.horizontal(|ui| {
            ui.spacing_mut().item_spacing.x = theme::SPACING;
            ui.label(
                egui::RichText::new("SESSION")
                    .font(theme::font_caption2())
                    .color(theme::text_tertiary()),
            );
            ui.add(
                egui::Label::new(
                    egui::RichText::new(headline)
                        .font(theme::font_headline())
                        .color(theme::text_primary()),
                )
                .truncate()
                .selectable(false),
            );
            ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
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
        });
    });
}
