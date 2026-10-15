use chrono::{DateTime, Utc};
use eframe::egui::{self, Ui};

use crate::fabrik::Observation;

use super::{badge, memory_sections, theme};

pub struct MemoryRowProps {
    pub obs: Observation,
    pub selected: bool,
}

#[must_use]
pub fn short_project(project: &str) -> String {
    let trimmed = project.trim();
    if trimmed.is_empty() {
        return "no project".to_owned();
    }
    let normalized = trimmed.replace('\\', "/");
    let raw_parts: Vec<&str> = normalized.split('/').collect();
    // Prefer the leaf name (`file_name`); fall back to the last two
    // components when no non-empty leaf exists.
    if let Some(leaf) = raw_parts.iter().rev().find(|part| !part.trim().is_empty()) {
        return leaf.trim().to_owned();
    }
    let tail: Vec<&str> = raw_parts
        .iter()
        .rev()
        .take(2)
        .map(|part| part.trim())
        .filter(|part| !part.is_empty())
        .collect();
    if tail.is_empty() {
        trimmed.to_owned()
    } else {
        tail.into_iter().rev().collect::<Vec<_>>().join("/")
    }
}

#[must_use]
pub fn strip_markdown_inline(line: &str) -> String {
    let mut text = line.trim();
    // Peel one leading markdown marker at a time (`#`, `>`, `-`, `*`,
    // `+`, backticks, emphasis), plus numbered-list prefixes.
    loop {
        let before_len = text.len();
        text = text
            .trim_start_matches(['#', '>', '-', '*', '+', '`', '_', '~'])
            .trim_start();
        let bytes = text.as_bytes();
        let mut idx = 0;
        while idx < bytes.len() && bytes[idx].is_ascii_digit() {
            idx += 1;
        }
        if idx > 0 && idx < bytes.len() && (bytes[idx] == b'.' || bytes[idx] == b')') {
            text = text[idx + 1..].trim_start();
        }
        if text.len() == before_len {
            break;
        }
    }
    text.replace("**", "")
        .replace("__", "")
        .replace('`', "")
        .split_whitespace()
        .collect::<Vec<_>>()
        .join(" ")
}

#[must_use]
pub fn preview(body: &str) -> String {
    const LIMIT: usize = 140;
    if memory_sections::is_structured(body) {
        let (_, sections) = memory_sections::parse_sections(body);
        let picked = sections
            .iter()
            .find(|s| s.key == "what")
            .or_else(|| sections.iter().find(|s| s.key == "goal"))
            .or_else(|| sections.first());
        if let Some(section) = picked {
            let joined = section
                .body
                .lines()
                .map(strip_markdown_inline)
                .filter(|line| !line.is_empty())
                .collect::<Vec<_>>()
                .join(" ");
            let collapsed = joined.split_whitespace().collect::<Vec<_>>().join(" ");
            if !collapsed.is_empty() {
                let taken: String = collapsed.chars().take(LIMIT).collect();
                if collapsed.chars().count() > LIMIT {
                    return format!("{taken}…");
                }
                return taken;
            }
        }
    }
    // First two meaningful lines, markdown markers stripped, joined
    // with an em dash so `**What**`-style bodies stay readable.
    let lead: Vec<String> = body
        .lines()
        .map(strip_markdown_inline)
        .filter(|line| !line.is_empty())
        .take(2)
        .collect();
    let joined = if lead.is_empty() {
        // Fallback: collapse all whitespace (legacy single-line shape).
        body.split_whitespace().collect::<Vec<&str>>().join(" ")
    } else {
        lead.join(" — ")
    };
    if joined.is_empty() {
        return String::new();
    }
    let taken: String = joined.chars().take(LIMIT).collect();
    if joined.chars().count() > LIMIT {
        format!("{taken}…")
    } else {
        taken
    }
}

#[must_use]
pub fn full_timestamp(created_at: &str) -> String {
    const OUT: &str = "%d.%m.%Y %H:%M";
    let trimmed = created_at.trim();
    if trimmed.is_empty() {
        return String::new();
    }
    if let Ok(parsed) = DateTime::parse_from_rfc3339(trimmed) {
        return parsed.with_timezone(&chrono::Local).format(OUT).to_string();
    }
    // SQLite shape (`2026-10-01 14:12:50`) has no zone: format as-is.
    for format in ["%Y-%m-%d %H:%M:%S", "%Y-%m-%dT%H:%M:%S", "%Y-%m-%d %H:%M"] {
        if let Ok(naive) = chrono::NaiveDateTime::parse_from_str(trimmed, format) {
            return naive.format(OUT).to_string();
        }
    }
    // Fallback: date part before `T`, else raw.
    if let Some((date, _)) = trimmed.split_once('T') {
        date.to_owned()
    } else {
        trimmed.to_owned()
    }
}

#[must_use]
pub fn relative_time(created_at: &str) -> String {
    let trimmed = created_at.trim();
    if trimmed.is_empty() {
        return String::new();
    }
    if let Ok(parsed) = DateTime::parse_from_rfc3339(trimmed) {
        let parsed_utc: DateTime<Utc> = parsed.with_timezone(&Utc);
        let now = Utc::now();
        let delta = now.signed_duration_since(parsed_utc);
        let mins = delta.num_minutes();
        if mins < 1 {
            return "just now".to_owned();
        }
        if mins < 60 {
            return format!("{mins}m");
        }
        let hours = delta.num_hours();
        if hours < 24 {
            return format!("{hours}h");
        }
        let days = delta.num_days();
        if days < 30 {
            return format!("{days}d");
        }
        return parsed_utc.format("%d.%m.%Y").to_string();
    }
    // Fallback: date part before `T`, else raw.
    if let Some((date, _)) = trimmed.split_once('T') {
        date.to_owned()
    } else {
        trimmed.to_owned()
    }
}

pub fn memory_row(ui: &mut Ui, props: &MemoryRowProps) -> bool {
    let row_id = ui.make_persistent_id(&props.obs.id);
    let was_hovered = ui.data_mut(|d| d.get_temp::<bool>(row_id).unwrap_or(false));

    let (fill, stroke) = if props.selected {
        (
            theme::row_selected_fill(),
            egui::Stroke::new(1.0_f32, theme::accent()),
        )
    } else if was_hovered {
        (
            theme::surface_hover(),
            egui::Stroke::new(1.0_f32, theme::border_hover()),
        )
    } else {
        (theme::surface_elevated(), theme::hairline())
    };

    let title = if props.obs.title.trim().is_empty() {
        props.obs.id.clone()
    } else {
        props.obs.title.clone()
    };
    let preview_text = preview(&props.obs.body);
    let relative = relative_time(&props.obs.created_at);

    let inner = theme::card_frame()
        .fill(fill)
        .stroke(stroke)
        .inner_margin(egui::Margin::same(theme::PAD_I8))
        .show(ui, |ui| {
            theme::fill_width(ui);
            ui.set_min_height(theme::ROW_HEIGHT);
            ui.vertical(|ui| {
                ui.spacing_mut().item_spacing.y = 2.0;
                ui.horizontal(|ui| {
                    ui.spacing_mut().item_spacing.x = theme::SPACING;
                    badge::kind_badge(ui, &props.obs.kind);
                    ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
                        ui.label(
                            egui::RichText::new(relative)
                                .font(theme::font_caption2())
                                .color(theme::text_tertiary()),
                        );
                    });
                });
                ui.add(
                    egui::Label::new(
                        egui::RichText::new(title)
                            .font(theme::font_callout())
                            .color(theme::text_primary()),
                    )
                    .truncate()
                    .selectable(false),
                );
                if !preview_text.is_empty() {
                    ui.add(
                        egui::Label::new(
                            egui::RichText::new(preview_text)
                                .font(theme::font_footnote())
                                .color(theme::text_secondary()),
                        )
                        .truncate()
                        .selectable(false),
                    );
                }
                ui.label(
                    egui::RichText::new(format!(
                        "{} · {}",
                        short_project(&props.obs.project),
                        props.obs.id
                    ))
                    .font(theme::font_caption2())
                    .color(theme::text_quaternary()),
                )
                .on_hover_text(hover_text(&props.obs));
            });
        });

    let response = inner.response.interact(egui::Sense::click());
    let is_now_hovered = response.hovered();
    if is_now_hovered != was_hovered {
        ui.data_mut(|d| d.insert_temp(row_id, is_now_hovered));
        ui.ctx().request_repaint();
    }
    let response = theme::pointer(response);
    response.clicked()
}

#[must_use]
fn hover_text(obs: &Observation) -> String {
    let project = if obs.project.trim().is_empty() {
        "no project".to_owned()
    } else {
        obs.project.trim().to_owned()
    };
    let mut hover = format!("{project}\n{}", full_timestamp(&obs.created_at));
    for (label, value) in [
        ("topic", obs.topic_key.as_deref()),
        ("symbol", obs.symbol_id.as_deref()),
        ("session", obs.session_id.as_deref()),
    ] {
        if let Some(raw) = value {
            let text = raw.trim();
            if !text.is_empty() {
                hover.push_str(&format!("\n{label}: {text}"));
            }
        }
    }
    hover
}

#[cfg(test)]
mod tests {
    use super::{full_timestamp, preview, relative_time, short_project, strip_markdown_inline};

    #[test]
    fn relative_fallback_date_part() {
        assert_eq!(relative_time("2026-10-01T10:00:00"), "2026-10-01");
    }

    #[test]
    fn relative_fallback_raw() {
        assert_eq!(relative_time("yesterday"), "yesterday");
    }

    #[test]
    fn relative_empty() {
        assert!(relative_time("").is_empty());
        assert!(relative_time("   ").is_empty());
    }

    #[test]
    fn relative_just_now() {
        let now = chrono::Utc::now().format("%Y-%m-%dT%H:%M:%SZ").to_string();
        assert_eq!(relative_time(&now), "just now");
    }

    #[test]
    fn preview_truncates() {
        let long = "word ".repeat(50);
        let out = preview(&long);
        assert!(out.ends_with('…'));
        assert!(out.chars().count() <= 141);
    }

    #[test]
    fn preview_normalizes_whitespace() {
        assert_eq!(preview("a\n  b\tc"), "a — b c");
    }

    #[test]
    fn preview_strips_markdown_two_lines() {
        let body = "## **What** happened\n\n- fixed `mem get`\n- more";
        assert_eq!(preview(body), "What happened — fixed mem get");
    }

    #[test]
    fn preview_skips_blank_markdown_lines() {
        assert_eq!(preview("# Title\n\nplain words"), "Title — plain words");
    }

    #[test]
    fn preview_empty() {
        assert!(preview("").is_empty());
        assert!(preview("   \n\t ").is_empty());
    }

    #[test]
    fn preview_structured_prefers_what() {
        let body = "**What**: fixed login timeout\n**Why**: session expired\n**Where**: auth.rs";
        assert_eq!(preview(body), "fixed login timeout");
    }

    #[test]
    fn preview_structured_prefers_goal() {
        let body = "## Goal\nship the release\n## Next Steps\n- verify\n## Files\nmain.rs";
        assert_eq!(preview(body), "ship the release");
    }

    #[test]
    fn short_project_unix() {
        assert_eq!(short_project("/home/user/fabrik"), "fabrik");
    }

    #[test]
    fn short_project_windows() {
        assert_eq!(short_project("C:\\Arbeit\\fabrik"), "fabrik");
        assert_eq!(short_project("C:/Arbeit/fabrik/"), "fabrik");
    }

    #[test]
    fn short_project_empty() {
        assert_eq!(short_project(""), "no project");
        assert_eq!(short_project("   "), "no project");
    }

    #[test]
    fn strip_markdown_markers() {
        assert_eq!(strip_markdown_inline("## **What**"), "What");
        assert_eq!(strip_markdown_inline("> quoted"), "quoted");
        assert_eq!(strip_markdown_inline("- [x] done"), "[x] done");
        assert_eq!(strip_markdown_inline("1. first"), "first");
    }

    #[test]
    fn full_timestamp_rfc3339() {
        let raw = "2026-10-01T14:12:50Z";
        let parsed = chrono::DateTime::parse_from_rfc3339(raw).unwrap();
        let expected = parsed
            .with_timezone(&chrono::Local)
            .format("%d.%m.%Y %H:%M")
            .to_string();
        assert_eq!(full_timestamp(raw), expected);
    }

    #[test]
    fn full_timestamp_sqlite() {
        assert_eq!(full_timestamp("2026-10-01 14:12:50"), "01.10.2026 14:12");
    }

    #[test]
    fn full_timestamp_fallback() {
        assert_eq!(full_timestamp("2026-10-01T10:00:00"), "01.10.2026 10:00");
        assert_eq!(full_timestamp("yesterday"), "yesterday");
        assert!(full_timestamp("  ").is_empty());
    }
}
