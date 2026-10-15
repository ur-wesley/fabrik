use eframe::egui::{Id, Ui};
use egui_graphs::MetadataFrame;

pub(super) const GRAPH_ID: &str = "lookup-graph";
pub(super) const CACHE_KEY: &str = "lookup_graph_cache";
pub(super) const FIT_KEY: &str = "lookup_graph_fit_request";
pub(super) const RESET_KEY: &str = "lookup_graph_reset_request";
pub const HINT_TEXT: &str = "Drag nodes to rearrange · Scroll to zoom · Click a node to select · Right-click copies path:line";
const ZOOM_IN_FACTOR: f32 = 1.25;
const ZOOM_OUT_FACTOR: f32 = 0.8;
const ZOOM_MIN: f32 = 0.2;
const ZOOM_MAX: f32 = 4.0;
/// Multiply the persisted view zoom, clamped to a sane range. Runs before
/// the canvas is drawn so the same frame already reflects the new zoom.
fn adjust_zoom(ui: &mut Ui, factor: f32) {
    let frame = MetadataFrame::new(Some(GRAPH_ID.to_owned())).load(ui);
    let mut next = frame;
    next.zoom = (next.zoom * factor).clamp(ZOOM_MIN, ZOOM_MAX);
    next.save(ui);
}

/// Manual wheel-zoom for the canvas response: the widget's own
/// zoom-and-pan stays enabled, and this applies the same
/// factor/clamp as the +/- buttons per wheel notch so a wheel that
/// never reaches the widget still zooms. Kept as the default; if a
/// runtime check shows double-zoom (widget + manual both firing),
/// drop this handler and rely on the widget zoom instead.
pub(super) fn apply_wheel_zoom(ui: &mut Ui, delta_y: f32) {
    let factor = if delta_y > 0.0 {
        ZOOM_IN_FACTOR
    } else {
        ZOOM_OUT_FACTOR
    };
    adjust_zoom(ui, factor);
}

fn request_flag(ui: &Ui, key: &str) {
    ui.ctx()
        .data_mut(|data| data.insert_temp(Id::new(key), true));
}

pub(super) fn take_flag(ui: &Ui, key: &str) -> bool {
    let id = Id::new(key);
    let hit: bool = ui
        .ctx()
        .data_mut(|data| data.get_temp::<bool>(id))
        .unwrap_or_default();
    if hit {
        ui.ctx().data_mut(|data| data.insert_temp(id, false));
    }
    hit
}

/// Zoom buttons for the compact header strip: zoom in/out, fit-to-screen,
/// reset layout. Renders bare buttons with no wrapping row so the caller
/// can place them inline in the single header strip. Fit arms a one-frame
/// `fit_to_screen`; reset drops the frame cache so the fingerprint
/// rebuilds, re-settles, and refits.
pub fn zoom_buttons(ui: &mut Ui) {
    if ui.small_button("+").on_hover_text("Zoom in").clicked() {
        adjust_zoom(ui, ZOOM_IN_FACTOR);
    }
    if ui.small_button("-").on_hover_text("Zoom out").clicked() {
        adjust_zoom(ui, ZOOM_OUT_FACTOR);
    }
    if ui
        .small_button("Fit")
        .on_hover_text("Fit graph to screen")
        .clicked()
    {
        request_flag(ui, FIT_KEY);
    }
    if ui
        .small_button("Reset")
        .on_hover_text("Reset layout")
        .clicked()
    {
        request_flag(ui, RESET_KEY);
    }
}
