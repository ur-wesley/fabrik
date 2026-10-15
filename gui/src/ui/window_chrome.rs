use eframe::egui::{self, CornerRadius, PointerButton, Sense, Ui, ViewportCommand};

use super::theme;
use crate::platform::glass_debug_enabled;

pub const TITLE_H: f32 = 40.0;
const TRAFFIC_SIZE: f32 = 14.0;
const TRAFFIC_PITCH: f32 = 23.0;
const TRAFFIC_LEFT: f32 = 20.0;

pub fn shell<R>(
    ui: &mut Ui,
    add_contents: impl FnOnce(&mut Ui, &mut Ui) -> (Vec<crate::model::Action>, R),
) -> (Vec<crate::model::Action>, R) {
    theme::sync_paint_from_ui(ui);
    let rect = ui.max_rect();
    ui.painter()
        .rect_filled(rect, CornerRadius::ZERO, egui::Color32::TRANSPARENT);
    let tint = if glass_debug_enabled() {
        theme::glass_debug_tint()
    } else {
        theme::glass_tint()
    };
    ui.painter().rect_filled(rect, CornerRadius::ZERO, tint);
    ui.painter().rect(
        rect,
        CornerRadius::ZERO,
        egui::Color32::TRANSPARENT,
        theme::glass_stroke(),
        egui::StrokeKind::Inside,
    );

    let title_rect =
        egui::Rect::from_min_max(rect.min, egui::pos2(rect.max.x, rect.min.y + TITLE_H));
    title_bar_chrome(ui, title_rect);

    let nav_left = if cfg!(target_os = "macos") {
        title_rect.min.x + TRAFFIC_LEFT + 3.0 * TRAFFIC_PITCH + theme::SPACING
    } else {
        title_rect.min.x + theme::PAD_CONTENT
    };
    let nav_right = if cfg!(target_os = "macos") {
        title_rect.max.x - theme::PAD_CONTENT
    } else {
        title_rect.max.x - 3.0 * TITLE_H
    };
    let nav_rect = egui::Rect::from_min_max(
        egui::pos2(nav_left, title_rect.min.y),
        egui::pos2(nav_right, title_rect.max.y),
    );
    let mut title_ui = ui.new_child(
        egui::UiBuilder::new()
            .max_rect(nav_rect)
            .layout(egui::Layout::left_to_right(egui::Align::Center)),
    );
    title_ui.set_height(TITLE_H);
    title_ui.spacing_mut().item_spacing.x = 6.0;

    let body_rect = egui::Rect::from_min_max(egui::pos2(rect.min.x, title_rect.max.y), rect.max);
    let mut body_ui = ui.new_child(
        egui::UiBuilder::new()
            .max_rect(body_rect)
            .layout(egui::Layout::top_down(egui::Align::Min)),
    );
    body_ui.set_clip_rect(body_rect);
    add_contents(&mut title_ui, &mut body_ui)
}

fn title_bar_chrome(ui: &mut Ui, rect: egui::Rect) {
    ui.painter()
        .rect_filled(rect, CornerRadius::ZERO, theme::title_bar_fill());
    ui.painter().hline(
        rect.left_bottom().x..=rect.right_bottom().x,
        rect.bottom(),
        theme::title_bar_separator(),
    );

    let focused = ui.input(|input| input.viewport().focused) != Some(false);
    let controls_reserve = if cfg!(target_os = "macos") {
        TRAFFIC_LEFT + 3.0 * TRAFFIC_PITCH + theme::SPACING
    } else {
        3.0 * TITLE_H
    };
    let drag_rect = egui::Rect::from_min_max(
        rect.min,
        egui::pos2((rect.max.x - controls_reserve).max(rect.min.x), rect.max.y),
    );
    let drag = ui.interact(
        drag_rect,
        ui.id().with("title_drag"),
        Sense::click_and_drag(),
    );
    if drag.double_clicked() {
        let maximized = ui.input(|input| input.viewport().maximized.unwrap_or(false));
        ui.ctx()
            .send_viewport_cmd(ViewportCommand::Maximized(!maximized));
    }
    if drag.drag_started_by(PointerButton::Primary) {
        ui.ctx().send_viewport_cmd(ViewportCommand::StartDrag);
    }
    if drag.dragged_by(PointerButton::Primary) {
        ui.ctx().request_repaint();
    }

    if cfg!(target_os = "macos") {
        traffic_lights(ui, rect, focused);
    } else {
        ui.scope_builder(
            egui::UiBuilder::new()
                .max_rect(rect)
                .layout(egui::Layout::right_to_left(egui::Align::Center)),
            |ui| {
                ui.spacing_mut().item_spacing.x = 0.0;
                window_controls(ui);
            },
        );
    }
}

fn traffic_lights(ui: &mut Ui, bar: egui::Rect, focused: bool) {
    let colors = if focused {
        [
            theme::traffic_close(),
            theme::traffic_minimize(),
            theme::traffic_zoom(),
        ]
    } else {
        [
            theme::traffic_inactive(),
            theme::traffic_inactive(),
            theme::traffic_inactive(),
        ]
    };
    let cy = bar.center().y;
    for (index, color) in colors.into_iter().enumerate() {
        let cx = bar.min.x + TRAFFIC_LEFT + TRAFFIC_SIZE * 0.5 + index as f32 * TRAFFIC_PITCH;
        let center = egui::pos2(cx, cy);
        let hit = egui::Rect::from_center_size(center, egui::vec2(TRAFFIC_SIZE, TRAFFIC_SIZE));
        let response = ui.interact(hit, ui.id().with(("traffic", index)), Sense::click());
        ui.painter()
            .circle_filled(center, TRAFFIC_SIZE * 0.5, color);
        if response.clicked() {
            match index {
                0 => ui.ctx().send_viewport_cmd(ViewportCommand::Close),
                1 => ui.ctx().send_viewport_cmd(ViewportCommand::Minimized(true)),
                2 => {
                    let maximized = ui.input(|input| input.viewport().maximized.unwrap_or(false));
                    ui.ctx()
                        .send_viewport_cmd(ViewportCommand::Maximized(!maximized));
                }
                _ => {}
            }
        }
    }
}

fn window_control_btn(ui: &mut Ui, icon: &str, is_close: bool) -> egui::Response {
    let size = egui::vec2(TITLE_H, TITLE_H);
    let (rect, response) = ui.allocate_exact_size(size, Sense::click());

    let is_hovered = response.hovered();
    let (bg_color, fg_color) = if is_hovered {
        if is_close {
            (egui::Color32::from_rgb(196, 43, 28), egui::Color32::WHITE)
        } else {
            (theme::control_fill_hover(), theme::foreground())
        }
    } else {
        (egui::Color32::TRANSPARENT, theme::dim_text())
    };

    if is_hovered {
        ui.painter().rect_filled(rect, CornerRadius::ZERO, bg_color);
    }

    ui.painter().text(
        rect.center(),
        egui::Align2::CENTER_CENTER,
        icon,
        egui::FontId::proportional(13.0),
        fg_color,
    );

    theme::pointer(response)
}

fn window_controls(ui: &mut Ui) {
    use super::icons;
    if window_control_btn(ui, icons::X, true).clicked() {
        ui.ctx().send_viewport_cmd(ViewportCommand::Close);
    }
    let maximized = ui.input(|input| input.viewport().maximized.unwrap_or(false));
    let max_icon = if maximized {
        icons::CORNERS_IN
    } else {
        icons::CORNERS_OUT
    };
    if window_control_btn(ui, max_icon, false).clicked() {
        ui.ctx()
            .send_viewport_cmd(ViewportCommand::Maximized(!maximized));
    }
    if window_control_btn(ui, icons::MINUS, false).clicked() {
        ui.ctx().send_viewport_cmd(ViewportCommand::Minimized(true));
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn titlebar_shell_renders_without_panic() {
        let ctx = egui::Context::default();
        let mut output = ctx.run_ui(egui::RawInput::default(), |ui| {
            egui::CentralPanel::default().show(ui, |ui| {
                shell(ui, |title, body| {
                    title.label("nav");
                    body.label("child contents");
                    (Vec::new(), ())
                });
            });
        });
        output.textures_delta.clear();
    }
}
