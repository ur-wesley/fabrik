use eframe::egui::{self, Sense, Ui};

use super::theme;

pub struct GroupedRowProps {
    pub title: String,
    pub subtitle: Option<String>,
    pub trailing: Option<String>,
    pub selected: bool,
    pub show_separator: bool,
}

pub fn grouped_row(ui: &mut Ui, props: &GroupedRowProps) -> egui::Response {
    let width = ui.available_width();
    let (rect, response) = ui.allocate_exact_size(
        egui::vec2(width, theme::ROW_HEIGHT.max(44.0)),
        Sense::click(),
    );
    if props.selected {
        ui.painter()
            .rect_filled(rect, egui::CornerRadius::ZERO, theme::row_selected_fill());
    }
    // Clip all painted text to the row rect so long subtitles can never
    // overflow into neighboring cards (the sidebar column is narrow).
    let mut child = ui.new_child(
        egui::UiBuilder::new()
            .max_rect(rect)
            .layout(egui::Layout::top_down(egui::Align::Min)),
    );
    child.set_clip_rect(rect);
    let inner = rect.shrink2(egui::vec2(theme::PAD, 6.0));
    let chevron_w = 14.0;
    let text_right = inner.max.x - chevron_w - 4.0;
    // Measured text budget: the title/subtitle share the space left of the
    // chevron; anything longer is ellipsized, full text stays in the hover
    // tooltip below.
    let budget = (text_right - inner.min.x).max(0.0);
    let title_font = theme::font_callout();
    let sub_font = theme::font_footnote();
    let title = ellipsize(&child, &props.title, &title_font, budget);
    child.painter().text(
        egui::pos2(inner.min.x, inner.min.y),
        egui::Align2::LEFT_TOP,
        title,
        title_font,
        theme::text_primary(),
    );
    let mut tip = props.title.clone();
    if let Some(sub) = &props.subtitle {
        let short = ellipsize(&child, sub, &sub_font, budget);
        child.painter().text(
            egui::pos2(inner.min.x, inner.min.y + 18.0),
            egui::Align2::LEFT_TOP,
            short,
            sub_font,
            theme::text_secondary(),
        );
        tip = format!("{tip}\n{sub}");
    }
    if let Some(trail) = &props.trailing {
        child.painter().text(
            egui::pos2(text_right, inner.center().y),
            egui::Align2::RIGHT_CENTER,
            trail,
            theme::font_footnote(),
            theme::text_tertiary(),
        );
    }
    child.painter().text(
        egui::pos2(inner.max.x, inner.center().y),
        egui::Align2::RIGHT_CENTER,
        "›",
        egui::FontId::proportional(12.0),
        theme::text_quaternary(),
    );
    if props.show_separator {
        let sep_y = rect.max.y - 0.5;
        child
            .painter()
            .hline((inner.min.x + 12.0)..=inner.max.x, sep_y, theme::hairline());
    }
    theme::pointer(response.on_hover_text(tip))
}

/// Ellipsize `text` with `…` until it fits `max_w` (measured with the same
/// painter layout the row paints with). Char-wise pop avoids byte-index
/// slicing on non-ASCII text; empty budget still yields a bare ellipsis.
fn ellipsize(ui: &Ui, text: &str, font: &egui::FontId, max_w: f32) -> String {
    if fits(ui, text, font, max_w) {
        return text.to_owned();
    }
    let mut out = text.to_owned();
    while !out.is_empty() {
        let _ = out.pop();
        if fits(ui, &format!("{out}…"), font, max_w) {
            break;
        }
    }
    format!("{out}…")
}

fn fits(ui: &Ui, text: &str, font: &egui::FontId, max_w: f32) -> bool {
    if max_w <= 0.0 {
        return false;
    }
    let color = theme::text_primary();
    ui.painter()
        .layout_no_wrap(text.to_owned(), font.clone(), color)
        .size()
        .x
        <= max_w
}
