use eframe::egui::{self, Color32, FontFamily, FontId, InnerResponse, Margin, Response, Sense, Ui};

use super::metrics::{PAD_I8, radius, radius_card, radius_dialog};
use super::tokens::{
    card_shadow, control_fill, dialog_fill, dialog_highlight_stroke, dialog_shadow, dialog_stroke,
    glass_stroke, grouped_secondary_bg, hairline, muted, toast_shadow,
};

pub fn dialog_frame(style: &egui::Style) -> egui::Frame {
    egui::Frame::popup(style)
        .fill(dialog_fill())
        .stroke(dialog_stroke())
        .shadow(dialog_shadow())
        .corner_radius(radius_dialog())
}

pub fn paint_dialog_top_highlight(ui: &Ui) {
    let rect = ui.max_rect();
    ui.painter()
        .hline(rect.x_range(), rect.top() + 1.0, dialog_highlight_stroke());
}
pub fn toast_frame() -> egui::Frame {
    egui::Frame::new()
        .fill(dialog_fill())
        .stroke(dialog_stroke())
        .shadow(toast_shadow())
        .inner_margin(Margin::symmetric(10, 6))
        .corner_radius(radius())
}

#[allow(dead_code)]
pub fn font_large_title() -> FontId {
    FontId::new(26.0, FontFamily::Proportional)
}

pub fn font_title2() -> FontId {
    FontId::new(20.0, FontFamily::Proportional)
}

pub fn font_headline() -> FontId {
    FontId::new(15.0, FontFamily::Proportional)
}

pub fn font_callout() -> FontId {
    FontId::new(12.0, FontFamily::Proportional)
}

pub fn font_footnote() -> FontId {
    FontId::new(11.0, FontFamily::Proportional)
}

pub fn font_caption2() -> FontId {
    FontId::new(10.0, FontFamily::Proportional)
}
pub fn card_frame() -> egui::Frame {
    egui::Frame::new()
        .fill(grouped_secondary_bg())
        .stroke(hairline())
        .shadow(card_shadow())
        .inner_margin(Margin::same(PAD_I8))
        .corner_radius(radius_card())
}

pub fn grouped_inset_frame() -> egui::Frame {
    egui::Frame::new()
        .fill(grouped_secondary_bg())
        .stroke(hairline())
        .corner_radius(radius_card())
        .inner_margin(Margin::ZERO)
}

pub fn muted_frame() -> egui::Frame {
    egui::Frame::new()
        .fill(muted())
        .stroke(hairline())
        .inner_margin(Margin::same(PAD_I8))
        .corner_radius(radius_card())
}

pub fn remaining_body<R>(ui: &mut Ui, add_contents: impl FnOnce(&mut Ui) -> R) -> R {
    let size = ui.available_size();
    let (rect, _) = ui.allocate_exact_size(size, egui::Sense::hover());
    let mut child = ui.new_child(
        egui::UiBuilder::new()
            .max_rect(rect)
            .layout(egui::Layout::top_down(egui::Align::Min)),
    );
    child.set_clip_rect(rect);
    add_contents(&mut child)
}

pub fn pointer(response: Response) -> Response {
    response.on_hover_cursor(egui::CursorIcon::PointingHand)
}

pub fn fill_width(ui: &mut Ui) {
    let w = ui.available_width();
    ui.set_min_width(w);
    ui.set_max_width(w);
    ui.set_width(w);
}

pub fn exact_card<R>(
    ui: &mut Ui,
    size: egui::Vec2,
    fill: Color32,
    pad: f32,
    add_contents: impl FnOnce(&mut Ui) -> R,
) -> InnerResponse<R> {
    let (rect, response) = ui.allocate_exact_size(size, egui::Sense::click());
    ui.painter()
        .rect(rect, radius(), fill, hairline(), egui::StrokeKind::Inside);
    let inner = rect.shrink(pad);
    let mut child = ui.new_child(
        egui::UiBuilder::new()
            .max_rect(inner)
            .layout(egui::Layout::top_down(egui::Align::Min)),
    );
    child.set_clip_rect(inner);
    let inner_res = add_contents(&mut child);
    InnerResponse::new(inner_res, response)
}

pub fn glass_slot<R>(
    ui: &mut Ui,
    size: egui::Vec2,
    sense: Sense,
    add_contents: impl FnOnce(&mut Ui) -> R,
) -> InnerResponse<R> {
    let (rect, response) = ui.allocate_exact_size(size, sense);
    ui.painter().rect(
        rect,
        radius(),
        control_fill(),
        glass_stroke(),
        egui::StrokeKind::Inside,
    );
    let inner = rect.shrink2(egui::vec2(8.0, 3.0));
    let mut child = ui.new_child(
        egui::UiBuilder::new()
            .max_rect(inner)
            .layout(egui::Layout::left_to_right(egui::Align::Center)),
    );
    child.set_clip_rect(rect);
    let inner_res = add_contents(&mut child);
    InnerResponse::new(inner_res, response)
}

pub fn glass_slot_compact<R>(
    ui: &mut Ui,
    size: egui::Vec2,
    sense: Sense,
    add_contents: impl FnOnce(&mut Ui) -> R,
) -> InnerResponse<R> {
    glass_slot(ui, size, sense, add_contents)
}
