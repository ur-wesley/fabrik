use std::sync::atomic::{AtomicBool, Ordering};

use eframe::egui::{self, Color32, Stroke};

static PAINT_DARK: AtomicBool = AtomicBool::new(true);

pub fn set_paint_dark(dark: bool) {
    PAINT_DARK.store(dark, Ordering::Relaxed);
}

pub(crate) fn dark() -> bool {
    PAINT_DARK.load(Ordering::Relaxed)
}

pub fn glass_tint() -> Color32 {
    if cfg!(target_os = "macos") {
        if dark() {
            Color32::from_rgba_unmultiplied(18, 18, 22, 56)
        } else {
            Color32::from_rgba_unmultiplied(246, 246, 246, 100)
        }
    } else if dark() {
        Color32::from_rgba_unmultiplied(0, 0, 0, 26)
    } else {
        Color32::from_rgba_unmultiplied(246, 246, 246, 92)
    }
}

pub fn glass_debug_tint() -> Color32 {
    Color32::from_rgba_unmultiplied(255, 0, 255, 40)
}

pub fn title_bar_fill() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(31, 31, 31, 140)
    } else {
        Color32::from_rgba_unmultiplied(255, 255, 255, 217)
    }
}

pub fn title_bar_separator() -> Stroke {
    if dark() {
        Stroke::new(1.0_f32, Color32::from_rgba_unmultiplied(255, 255, 255, 26))
    } else {
        Stroke::new(1.0_f32, Color32::from_rgba_unmultiplied(0, 0, 0, 13))
    }
}

pub fn surface_elevated() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(26, 26, 26, 245)
    } else {
        Color32::from_rgba_unmultiplied(245, 245, 245, 184)
    }
}

pub fn surface_hover() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 28)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 28)
    }
}

pub fn surface_active() -> Color32 {
    accent()
}

pub fn muted() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 23)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 26)
    }
}

pub fn control_fill() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 18)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 20)
    }
}

pub fn control_fill_hover() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 28)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 28)
    }
}

pub fn control_fill_pressed() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 41)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 41)
    }
}

pub fn accent_secondary_fill() -> Color32 {
    let a = accent();
    Color32::from_rgba_unmultiplied(a.r(), a.g(), a.b(), if dark() { 56 } else { 51 })
}

pub fn segmented_track() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 23)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 26)
    }
}

pub fn on_accent_text() -> Color32 {
    foreground()
}

pub fn popup_bg() -> Color32 {
    dialog_fill()
}

pub fn popup_shadow() -> egui::Shadow {
    dialog_shadow()
}

pub fn dialog_backdrop() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(0, 0, 0, 120)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 80)
    }
}

pub fn dialog_fill() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(26, 26, 26, 245)
    } else {
        Color32::from_rgba_unmultiplied(245, 245, 245, 230)
    }
}

pub fn dialog_stroke() -> Stroke {
    if dark() {
        Stroke::new(1.0_f32, Color32::from_rgba_unmultiplied(255, 255, 255, 51))
    } else {
        Stroke::new(1.0_f32, Color32::from_rgba_unmultiplied(0, 0, 0, 31))
    }
}

pub fn dialog_shadow() -> egui::Shadow {
    egui::Shadow {
        offset: [0, 18],
        blur: 54,
        spread: 0,
        color: Color32::from_rgba_unmultiplied(0, 0, 0, if dark() { 145 } else { 77 }),
    }
}

pub fn dialog_top_highlight() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 51)
    } else {
        Color32::from_rgba_unmultiplied(255, 255, 255, 179)
    }
}

pub fn dialog_highlight_stroke() -> Stroke {
    Stroke::new(1.0_f32, dialog_top_highlight())
}

pub fn search_fill() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(118, 118, 128, 61)
    } else {
        Color32::from_rgba_unmultiplied(118, 118, 128, 31)
    }
}

pub fn foreground() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 217)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 217)
    }
}

pub fn foreground_dimmed() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 128)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 128)
    }
}

pub fn border() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 43)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 31)
    }
}

pub fn border_hover() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 64)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 51)
    }
}

pub fn hairline() -> Stroke {
    Stroke::new(1.0_f32, border())
}

pub fn glass_stroke() -> Stroke {
    hairline()
}

pub fn accent() -> Color32 {
    if dark() {
        Color32::from_rgb(10, 132, 255)
    } else {
        Color32::from_rgb(0, 122, 255)
    }
}

pub fn grouped_bg() -> Color32 {
    if dark() {
        Color32::from_rgb(28, 28, 30)
    } else {
        Color32::from_rgb(242, 242, 247)
    }
}

pub fn grouped_secondary_bg() -> Color32 {
    if dark() {
        Color32::from_rgb(44, 44, 46)
    } else {
        Color32::from_rgb(255, 255, 255)
    }
}

pub fn text_primary() -> Color32 {
    foreground()
}

pub fn text_secondary() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(235, 235, 245, 153)
    } else {
        Color32::from_rgba_unmultiplied(60, 60, 67, 153)
    }
}

pub fn text_tertiary() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(235, 235, 245, 77)
    } else {
        Color32::from_rgba_unmultiplied(60, 60, 67, 77)
    }
}

pub fn text_quaternary() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(235, 235, 245, 46)
    } else {
        Color32::from_rgba_unmultiplied(60, 60, 67, 46)
    }
}

pub fn text_placeholder() -> Color32 {
    text_tertiary()
}

pub fn segment_selected_fill() -> Color32 {
    if dark() {
        Color32::from_rgb(99, 99, 102)
    } else {
        Color32::WHITE
    }
}

pub fn segment_selected_stroke() -> Stroke {
    if dark() {
        Stroke::new(0.5, Color32::from_rgba_unmultiplied(255, 255, 255, 31))
    } else {
        Stroke::new(0.5, Color32::from_rgba_unmultiplied(0, 0, 0, 20))
    }
}

pub fn card_shadow() -> egui::Shadow {
    egui::Shadow {
        offset: [0, 2],
        blur: 8,
        spread: 0,
        color: Color32::from_rgba_unmultiplied(0, 0, 0, if dark() { 64 } else { 26 }),
    }
}

pub fn toast_shadow() -> egui::Shadow {
    egui::Shadow {
        offset: [0, 6],
        blur: 20,
        spread: 0,
        color: Color32::from_rgba_unmultiplied(0, 0, 0, if dark() { 72 } else { 36 }),
    }
}

pub const fn neon_cyan() -> Color32 {
    Color32::from_rgb(0, 136, 255)
}

pub fn neon_magenta() -> Color32 {
    sys_red()
}

pub fn neon_amber() -> Color32 {
    sys_orange()
}

pub const fn neon_blue() -> Color32 {
    Color32::from_rgb(0, 136, 255)
}

pub fn neon_green() -> Color32 {
    sys_green()
}

pub fn neon_violet() -> Color32 {
    sys_purple()
}

pub fn sys_red() -> Color32 {
    if dark() {
        Color32::from_rgb(255, 97, 97)
    } else {
        Color32::from_rgb(255, 59, 48)
    }
}

pub fn sys_orange() -> Color32 {
    if dark() {
        Color32::from_rgb(255, 184, 77)
    } else {
        Color32::from_rgb(255, 149, 0)
    }
}

pub fn sys_green() -> Color32 {
    if dark() {
        Color32::from_rgb(85, 245, 125)
    } else {
        Color32::from_rgb(52, 199, 89)
    }
}

pub fn sys_purple() -> Color32 {
    if dark() {
        Color32::from_rgb(255, 97, 255)
    } else {
        Color32::from_rgb(175, 82, 222)
    }
}

/// Distinct node tint per symbol kind for the lookup graph viewer.
///
/// Covers the node kinds the CLI graph emits (`file`, `symbol`), the edge
/// kinds reused as node kinds on derived columns (`calls`, `contains`,
/// `imports`), and common symbol kinds (`fn`, `struct`, `enum`, `trait`,
/// …) so future CLI output already renders distinctly. Returns `None` for
/// unknown kinds so callers can fall back to the column-role tint.
pub fn kind_color(kind: &str) -> Option<Color32> {
    match kind.to_ascii_lowercase().as_str() {
        "fn" | "function" | "method" | "closure" | "calls" | "caller" | "callee" => {
            Some(neon_cyan())
        }
        "struct" | "class" | "type" | "symbol" | "record" | "object" => Some(neon_violet()),
        "enum" | "union" => Some(neon_magenta()),
        "trait" | "interface" | "protocol" => Some(neon_green()),
        "file" | "module" | "package" | "contains" => Some(neon_amber()),
        "import" | "imports" | "export" => Some(traffic_minimize()),
        _ => None,
    }
}

pub fn text_dim() -> Color32 {
    if dark() {
        Color32::from_rgb(152, 152, 157)
    } else {
        Color32::from_rgb(99, 99, 102)
    }
}

pub fn dim_text() -> Color32 {
    text_dim()
}

pub fn text_muted() -> Color32 {
    text_dim()
}

pub fn flash_bg() -> Color32 {
    let a = accent();
    Color32::from_rgba_unmultiplied(a.r(), a.g(), a.b(), 38)
}

pub fn row_selected_fill() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 56)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 52)
    }
}

pub fn traffic_close() -> Color32 {
    Color32::from_rgb(255, 92, 96)
}

pub fn traffic_minimize() -> Color32 {
    Color32::from_rgb(250, 200, 0)
}

pub fn traffic_zoom() -> Color32 {
    Color32::from_rgb(53, 199, 89)
}

pub fn traffic_inactive() -> Color32 {
    if dark() {
        Color32::from_rgba_unmultiplied(255, 255, 255, 43)
    } else {
        Color32::from_rgba_unmultiplied(0, 0, 0, 38)
    }
}
