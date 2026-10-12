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
    let inner = rect.shrink2(egui::vec2(theme::PAD, 6.0));
    let chevron_w = 14.0;
    let text_right = inner.max.x - chevron_w - 4.0;
    let title_pos = egui::pos2(inner.min.x, inner.min.y);
    ui.painter().text(
        title_pos,
        egui::Align2::LEFT_TOP,
        &props.title,
        theme::font_callout(),
        theme::text_primary(),
    );
    if let Some(sub) = &props.subtitle {
        ui.painter().text(
            egui::pos2(inner.min.x, inner.min.y + 18.0),
            egui::Align2::LEFT_TOP,
            sub,
            theme::font_footnote(),
            theme::text_secondary(),
        );
    }
    if let Some(trail) = &props.trailing {
        ui.painter().text(
            egui::pos2(text_right, inner.center().y),
            egui::Align2::RIGHT_CENTER,
            trail,
            theme::font_footnote(),
            theme::text_tertiary(),
        );
    }
    ui.painter().text(
        egui::pos2(inner.max.x, inner.center().y),
        egui::Align2::RIGHT_CENTER,
        "›",
        egui::FontId::proportional(12.0),
        theme::text_quaternary(),
    );
    if props.show_separator {
        let sep_y = rect.max.y - 0.5;
        ui.painter().hline(
            (inner.min.x + 12.0)..=inner.max.x,
            sep_y,
            theme::hairline(),
        );
    }
    theme::pointer(response)
}
