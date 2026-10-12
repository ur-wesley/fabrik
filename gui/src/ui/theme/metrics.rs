use eframe::egui;

pub const SPACING: f32 = 8.0;
pub const PAD: f32 = 12.0;
pub const PAD_CONTENT: f32 = 12.0;
pub const PAD_I8: i8 = 12;
pub const SPACING_I8: i8 = 8;
pub const HEADER_H: f32 = 38.0;
pub const CONTROL_H: f32 = 32.0;
pub const SEARCH_H: f32 = 36.0;
pub const PICKER_SEARCH_H: f32 = 36.0;
pub const TEXT_INPUT_MARGIN_V: i8 = 7;
pub const ROW_HEIGHT: f32 = 44.0;
pub const TAB_H: f32 = 28.0;
pub const TAB_MIN_W: f32 = 120.0;
pub const TAB_MAX_W: f32 = 220.0;
pub const TAB_PAD_X: f32 = 4.0;
pub const TAB_PAD_LEFT: f32 = 4.0;
pub const TAB_PAD_RIGHT: f32 = 2.0;
pub const TAB_LABEL_GAP: f32 = 8.0;
pub const TAB_CLOSE_W: f32 = 16.0;
pub const SEARCH_MAX: f32 = 280.0;
pub const LANE_COLLAPSED_W: f32 = 38.0;
pub const LANE_COLLAPSED_PAD: f32 = 8.0;
pub const LANE_PAD: f32 = 12.0;
pub const LANE_GAP: f32 = 8.0;
pub const LANE_MAX_W: f32 = 380.0;
pub const FOOTER_H: f32 = 28.0;
pub const SEGMENT_H: f32 = 28.0;

pub const fn radius() -> egui::CornerRadius {
    egui::CornerRadius::same(8)
}

pub const fn radius_segment() -> egui::CornerRadius {
    egui::CornerRadius::same(7)
}

pub const fn radius_card() -> egui::CornerRadius {
    egui::CornerRadius::same(14)
}

pub const fn radius_dialog() -> egui::CornerRadius {
    egui::CornerRadius::same(16)
}

pub const fn radius_sm() -> egui::CornerRadius {
    egui::CornerRadius::same(4)
}

pub const fn radius_picker_search() -> egui::CornerRadius {
    egui::CornerRadius::same(10)
}
