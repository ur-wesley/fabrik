use eframe::egui::{self, Color32, FontFamily, FontId, Stroke, Style, TextStyle, Ui, Visuals};

use super::metrics::{SPACING, radius, radius_dialog};
use super::tokens::{
    accent, border, border_hover, control_fill, control_fill_hover, control_fill_pressed,
    foreground, hairline, popup_bg, popup_shadow, set_paint_dark, surface_elevated, text_dim,
    text_muted,
};

pub fn apply(style: &mut Style) {
    let dark_mode = style.visuals.dark_mode;
    set_paint_dark(dark_mode);
    let mut visuals = if dark_mode {
        Visuals::dark()
    } else {
        Visuals::light()
    };
    visuals.dark_mode = dark_mode;
    visuals.panel_fill = Color32::TRANSPARENT;
    visuals.window_fill = popup_bg();
    visuals.window_stroke = Stroke::new(1.0_f32, border());
    visuals.window_shadow = popup_shadow();
    visuals.popup_shadow = popup_shadow();
    visuals.window_corner_radius = radius_dialog();
    visuals.faint_bg_color = surface_elevated();
    visuals.extreme_bg_color = if dark_mode {
        Color32::from_rgb(30, 30, 30)
    } else {
        Color32::WHITE
    };
    visuals.widgets.noninteractive.bg_fill = surface_elevated();
    visuals.widgets.noninteractive.weak_bg_fill = surface_elevated();
    visuals.widgets.noninteractive.bg_stroke = Stroke::new(1.0_f32, border());
    visuals.widgets.noninteractive.fg_stroke = Stroke::new(1.0_f32, text_dim());
    visuals.widgets.noninteractive.corner_radius = radius();
    visuals.widgets.inactive.bg_fill = control_fill();
    visuals.widgets.inactive.weak_bg_fill = Color32::TRANSPARENT;
    visuals.widgets.inactive.bg_stroke = Stroke::new(1.0_f32, border());
    visuals.widgets.inactive.fg_stroke = Stroke::new(1.0_f32, text_muted());
    visuals.widgets.inactive.corner_radius = radius();
    visuals.widgets.inactive.expansion = 0.0;
    visuals.widgets.hovered.bg_fill = control_fill_hover();
    visuals.widgets.hovered.weak_bg_fill = control_fill_hover();
    visuals.widgets.hovered.bg_stroke = Stroke::new(1.0_f32, border_hover());
    visuals.widgets.hovered.fg_stroke = Stroke::new(1.0_f32, foreground());
    visuals.widgets.hovered.corner_radius = radius();
    visuals.widgets.hovered.expansion = 0.0;
    visuals.widgets.active.bg_fill = control_fill_pressed();
    visuals.widgets.active.weak_bg_fill = control_fill_pressed();
    visuals.widgets.active.bg_stroke = Stroke::new(1.0_f32, accent());
    visuals.widgets.active.fg_stroke = Stroke::new(1.0_f32, foreground());
    visuals.widgets.active.corner_radius = radius();
    visuals.widgets.active.expansion = 0.0;
    visuals.widgets.open.bg_fill = popup_bg();
    visuals.widgets.open.weak_bg_fill = popup_bg();
    visuals.widgets.open.bg_stroke = hairline();
    visuals.widgets.open.fg_stroke = Stroke::new(1.0_f32, foreground());
    visuals.widgets.open.corner_radius = radius();
    let accent = accent();
    visuals.selection.bg_fill =
        Color32::from_rgba_unmultiplied(accent.r(), accent.g(), accent.b(), 45);
    visuals.selection.stroke = Stroke::new(1.0_f32, accent);
    visuals.override_text_color = Some(foreground());
    style.visuals = visuals;
    style.spacing.item_spacing = egui::vec2(SPACING, SPACING);
    style.spacing.button_padding = egui::vec2(10.0, 6.0);
    style.text_styles.insert(
        TextStyle::Heading,
        FontId::new(20.0, FontFamily::Proportional),
    );
    style
        .text_styles
        .insert(TextStyle::Body, FontId::new(13.0, FontFamily::Proportional));
    style.text_styles.insert(
        TextStyle::Small,
        FontId::new(11.0, FontFamily::Proportional),
    );
    style.text_styles.insert(
        TextStyle::Monospace,
        FontId::new(12.0, FontFamily::Monospace),
    );
}

pub fn sync_paint_from_ui(ui: &Ui) {
    set_paint_dark(ui.style().visuals.dark_mode);
}

pub fn sync_paint_from_ctx(ctx: &egui::Context) {
    set_paint_dark(ctx.global_style().visuals.dark_mode);
}
