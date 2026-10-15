use eframe::egui::{self, ScrollArea, Ui};
use egui_commonmark::CommonMarkCache;

use crate::fabrik::{MemTimelineResult, Observation};
use crate::model::Action;

use super::{badge, button, icon_button, icons, memory_row, memory_sections, theme};

#[allow(clippy::too_many_lines)]
pub fn detail_sheet(
    ui: &mut Ui,
    detail: &Observation,
    markdown: &mut CommonMarkCache,
    timeline: Option<&MemTimelineResult>,
    timeline_id: Option<&str>,
    delete_confirm: Option<&str>,
    write_busy: bool,
) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.set_min_width(720.0);
    ui.set_max_width(800.0);
    ui.set_min_height(520.0);
    ui.set_max_height(640.0);
    theme::paint_dialog_top_highlight(ui);

    let mut close = false;
    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = theme::SPACING;
        badge::kind_badge(ui, &detail.kind);
        ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
            if icon_button::icon_button(
                ui,
                &icon_button::IconButtonProps {
                    label: icons::X,
                    selected: false,
                },
            ) {
                close = true;
            }
            if theme::pointer(
                ui.add(
                    egui::Button::new(egui::RichText::new("delete").small())
                        .fill(egui::Color32::TRANSPARENT)
                        .stroke(egui::Stroke::NONE),
                ),
            )
            .clicked()
            {
                actions.push(Action::RequestMemDelete(detail.id.clone()));
            }
            if theme::pointer(
                ui.add(
                    egui::Button::new(egui::RichText::new("edit").small())
                        .fill(egui::Color32::TRANSPARENT)
                        .stroke(egui::Stroke::NONE),
                ),
            )
            .clicked()
            {
                actions.push(Action::OpenMemEdit(detail.id.clone()));
            }
            if theme::pointer(
                ui.add(
                    egui::Button::new(egui::RichText::new("copy id").small())
                        .fill(egui::Color32::TRANSPARENT)
                        .stroke(egui::Stroke::NONE),
                ),
            )
            .on_hover_text(detail.id.clone())
            .clicked()
            {
                ui.ctx().copy_text(detail.id.clone());
            }
            if theme::pointer(
                ui.add(
                    egui::Button::new(egui::RichText::new("copy body").small())
                        .fill(egui::Color32::TRANSPARENT)
                        .stroke(egui::Stroke::NONE),
                ),
            )
            .clicked()
            {
                ui.ctx().copy_text(humanize_body(&detail.body));
            }
        });
    });
    ui.add_space(4.0);
    let title = if detail.title.trim().is_empty() {
        detail.id.clone()
    } else {
        detail.title.clone()
    };
    ui.add(
        egui::Label::new(
            egui::RichText::new(title)
                .size(18.0)
                .strong()
                .color(theme::text_primary()),
        )
        .wrap(),
    );
    ui.add_space(4.0);
    let humanized = humanize_body(&detail.body);
    let meta = ui.label(
        egui::RichText::new(format!(
            "{} · {} · {}",
            memory_row::short_project(&detail.project),
            memory_row::relative_time(&detail.created_at),
            memory_row::full_timestamp(&detail.created_at),
        ))
        .font(theme::font_caption2())
        .color(theme::text_tertiary()),
    );
    meta.on_hover_text(format!(
        "{}\n{}\n{}",
        detail.project, detail.created_at, detail.id,
    ));
    let mut chips: Vec<String> = Vec::new();
    if let Some(topic) = detail.topic_key.as_deref() {
        let text = topic.trim();
        if !text.is_empty() {
            chips.push(format!("topic:{text}"));
        }
    }
    if let Some(symbol) = detail.symbol_id.as_deref() {
        let text = symbol.trim();
        if !text.is_empty() {
            chips.push(format!("sym:{text}"));
        }
    }
    if let Some(session) = detail.session_id.as_deref() {
        let text = session.trim();
        if !text.is_empty() {
            let short: String = text.chars().take(12).collect();
            chips.push(format!("sess:{short}"));
        }
    }
    if !chips.is_empty() {
        ui.add_space(2.0);
        ui.horizontal_wrapped(|ui| {
            ui.spacing_mut().item_spacing = egui::vec2(4.0, 4.0);
            for chip in &chips {
                badge::badge_color(ui, chip, theme::text_tertiary());
            }
        });
    }
    ui.add_space(theme::SPACING);
    ui.painter().hline(
        ui.max_rect().x_range(),
        ui.cursor().top(),
        theme::hairline(),
    );
    ui.painter().hline(
        ui.max_rect().x_range(),
        ui.cursor().top() + 1.0,
        theme::dialog_highlight_stroke(),
    );
    ui.add_space(theme::SPACING);

    if close {
        return vec![Action::MemClearDetail];
    }
    if !actions.is_empty() {
        return actions;
    }

    ScrollArea::vertical()
        .id_salt("memory_detail_scroll")
        .auto_shrink([false, false])
        .show(ui, |ui| {
            theme::fill_width(ui);
            if detail.body.trim().is_empty() {
                theme::muted_frame()
                    .inner_margin(egui::Margin::same(theme::PAD_I8))
                    .show(ui, |ui| {
                        theme::fill_width(ui);
                        ui.label(
                            egui::RichText::new("No body recorded.")
                                .font(theme::font_footnote())
                                .italics()
                                .color(theme::text_tertiary()),
                        );
                    });
            } else {
                let (preamble, sections) = memory_sections::parse_sections(&humanized);
                if sections.len() >= 2 {
                    render_structured_body(ui, markdown, &preamble, &sections);
                } else {
                    render_plain_body(ui, markdown, &humanized);
                }
            }
            if let Some(timeline) = timeline {
                ui.add_space(theme::SPACING);
                timeline_card(ui, timeline, timeline_id);
            }
            if let Some(confirm) = delete_confirm {
                ui.add_space(theme::SPACING);
                actions.extend(delete_inline(ui, confirm, write_busy));
            }
        });

    ui.add_space(theme::SPACING);
    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = theme::SPACING;
        if button::button(
            ui,
            &button::ButtonProps::new("timeline")
                .variant(button::ButtonVariant::Outline)
                .min_size(egui::vec2(84.0, theme::CONTROL_H)),
        )
        .clicked()
        {
            actions.push(Action::FetchMemTimeline {
                id: detail.id.clone(),
                limit: 5,
            });
        }
        ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
            if button::button(
                ui,
                &button::ButtonProps::new("close")
                    .variant(button::ButtonVariant::Secondary)
                    .min_size(egui::vec2(84.0, theme::CONTROL_H)),
            )
            .clicked()
            {
                actions.push(Action::MemClearDetail);
            }
        });
    });
    actions
}

fn delete_inline(ui: &mut Ui, id: &str, busy: bool) -> Vec<Action> {
    let mut actions = Vec::new();
    theme::card_frame().show(ui, |ui| {
        ui.label(
            egui::RichText::new(format!("Delete {id}?"))
                .font(theme::font_headline())
                .color(theme::text_primary()),
        );
        ui.label(
            egui::RichText::new("Soft-deletes the observation (`fabrik mem delete --id …`).")
                .font(theme::font_footnote())
                .color(theme::text_secondary()),
        );
        ui.add_space(4.0);
        ui.horizontal(|ui| {
            if busy {
                ui.add(egui::Spinner::new());
                ui.label(
                    egui::RichText::new("Deleting…")
                        .small()
                        .italics()
                        .color(theme::dim_text()),
                );
            } else {
                if button::button(
                    ui,
                    &button::ButtonProps::new("Cancel")
                        .variant(button::ButtonVariant::Outline)
                        .min_size(egui::vec2(84.0, theme::CONTROL_H)),
                )
                .clicked()
                {
                    actions.push(Action::CancelMemDelete);
                }
                if button::button(
                    ui,
                    &button::ButtonProps::new("Delete")
                        .variant(button::ButtonVariant::Secondary)
                        .min_size(egui::vec2(84.0, theme::CONTROL_H)),
                )
                .clicked()
                {
                    actions.push(Action::ConfirmMemDelete);
                }
            }
        });
    });
    actions
}

pub fn timeline_card(ui: &mut Ui, timeline: &MemTimelineResult, timeline_id: Option<&str>) {
    theme::card_frame().show(ui, |ui| {
        let title = timeline_id.unwrap_or(timeline.id.as_str());
        ui.label(
            egui::RichText::new(format!(
                "timeline {title} ({} before · {} after)",
                timeline.before.len(),
                timeline.after.len(),
            ))
            .font(theme::font_headline())
            .color(theme::text_primary()),
        );
        let center = if timeline.observation.title.trim().is_empty() {
            timeline.observation.id.clone()
        } else {
            timeline.observation.title.clone()
        };
        ui.add(
            egui::Label::new(
                egui::RichText::new(center)
                    .font(theme::font_footnote())
                    .color(theme::text_secondary()),
            )
            .wrap()
            .selectable(false),
        );
        for row in timeline.before.iter().chain(timeline.after.iter()) {
            let row_title = if row.title.trim().is_empty() {
                row.id.clone()
            } else {
                row.title.clone()
            };
            let relative = memory_row::relative_time(&row.created_at);
            let prefix: String = if relative.is_empty() {
                row.id.chars().take(10).collect()
            } else {
                relative
            };
            ui.add(
                egui::Label::new(
                    egui::RichText::new(format!("{prefix} · {row_title}"))
                        .font(theme::font_caption2())
                        .color(theme::text_tertiary()),
                )
                .wrap()
                .selectable(false),
            );
        }
    });
}

fn render_plain_body(ui: &mut Ui, markdown: &mut CommonMarkCache, humanized: &str) {
    theme::muted_frame()
        .inner_margin(egui::Margin::same(theme::PAD_I8))
        .show(ui, |ui| {
            theme::fill_width(ui);
            render_body_text(ui, markdown, humanized);
        });
}

fn render_structured_body(
    ui: &mut Ui,
    markdown: &mut CommonMarkCache,
    preamble: &str,
    sections: &[memory_sections::MemSection],
) {
    if !preamble.trim().is_empty() {
        ui.add(
            egui::Label::new(
                egui::RichText::new(preamble.trim())
                    .font(theme::font_footnote())
                    .color(theme::text_primary()),
            )
            .wrap()
            .selectable(true),
        );
        ui.add_space(theme::SPACING);
    }
    let mut first = preamble.trim().is_empty();
    for section in sections {
        if !first {
            ui.add_space(theme::SPACING);
        }
        first = false;
        theme::card_frame()
            .inner_margin(egui::Margin::same(theme::PAD_I8))
            .show(ui, |ui| {
                theme::fill_width(ui);
                ui.horizontal(|ui| {
                    ui.spacing_mut().item_spacing.x = 6.0;
                    ui.label(
                        egui::RichText::new(memory_sections::section_icon(&section.key))
                            .color(theme::text_tertiary()),
                    );
                    ui.label(
                        egui::RichText::new(section.label.to_uppercase())
                            .font(theme::font_caption2())
                            .strong()
                            .color(theme::text_tertiary()),
                    );
                });
                ui.add_space(4.0);
                let body = section.body.trim();
                if body.is_empty() {
                    ui.label(
                        egui::RichText::new("—")
                            .font(theme::font_footnote())
                            .italics()
                            .color(theme::text_tertiary()),
                    );
                } else {
                    render_body_text(ui, markdown, body);
                }
            });
    }
}

fn render_body_text(ui: &mut Ui, markdown: &mut CommonMarkCache, text: &str) {
    if is_markdown(text) {
        egui_commonmark::CommonMarkViewer::new().show(ui, markdown, text);
    } else {
        let mut first = true;
        for paragraph in text.split("\n\n") {
            let part = paragraph.trim();
            if part.is_empty() {
                continue;
            }
            if !first {
                ui.add_space(theme::SPACING);
            }
            first = false;
            ui.add(
                egui::Label::new(
                    egui::RichText::new(part)
                        .font(theme::font_footnote())
                        .color(theme::text_primary()),
                )
                .wrap()
                .selectable(true),
            );
        }
    }
}

#[must_use]
pub fn humanize_body(body: &str) -> String {
    let step = body.replace("\r\n", "\n").replace('\r', "\n");
    let mut collapsed = String::with_capacity(step.len());
    let mut run = 0;
    for ch in step.chars() {
        if ch == '\n' {
            run += 1;
            if run <= 2 {
                collapsed.push(ch);
            }
        } else {
            run = 0;
            collapsed.push(ch);
        }
    }
    let trimmed = collapsed.trim().to_owned();
    if trimmed.is_empty() {
        "No body recorded.".to_owned()
    } else {
        trimmed
    }
}

fn is_markdown(body: &str) -> bool {
    for line in body.lines() {
        let text = line.trim_start();
        if text.starts_with("```")
            || text.starts_with('#')
            || text.starts_with('>')
            || text.starts_with("- ")
            || text.starts_with("* ")
            || text.starts_with("+ ")
        {
            return true;
        }
        let bytes = text.as_bytes();
        let mut idx = 0;
        while idx < bytes.len() && bytes[idx].is_ascii_digit() {
            idx += 1;
        }
        if idx > 0 && idx < bytes.len() && (bytes[idx] == b'.' || bytes[idx] == b')') {
            return true;
        }
    }
    body.contains("**")
        || body.contains("__")
        || body.contains('`')
        || body.contains("](")
        || body.contains("![")
        || body
            .lines()
            .any(|line| line.trim_start_matches('|').contains('|'))
}
