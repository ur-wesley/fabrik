use eframe::egui::{self, Sense, Ui};

use super::theme;

pub struct ToggleProps {
    pub on: bool,
    pub label: &'static str,
    pub enabled: bool,
}

pub fn toggle(ui: &mut Ui, props: &ToggleProps) -> Option<bool> {
    let mut flipped = None;
    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = theme::SPACING;
        let label = egui::RichText::new(props.label)
            .font(theme::font_callout())
            .color(if props.enabled {
                theme::text_primary()
            } else {
                theme::text_tertiary()
            });
        ui.add(egui::Label::new(label).selectable(false));
        let track_w = 44.0;
        let track_h = 26.0;
        let (rect, response) = ui.allocate_exact_size(
            egui::vec2(track_w, track_h),
            if props.enabled {
                Sense::click()
            } else {
                Sense::hover()
            },
        );
        let track_radius = egui::CornerRadius::same(13);
        let track_fill = if props.on && props.enabled {
            theme::accent()
        } else {
            theme::segmented_track()
        };
        ui.painter().rect_filled(rect, track_radius, track_fill);
        let knob = 22.0;
        let pad = 2.0;
        let knob_x = if props.on {
            rect.max.x - knob - pad
        } else {
            rect.min.x + pad
        };
        let knob_rect = egui::Rect::from_min_size(
            egui::pos2(knob_x, rect.center().y - knob / 2.0),
            egui::vec2(knob, knob),
        );
        ui.painter().circle_filled(
            knob_rect.center(),
            knob / 2.0,
            theme::grouped_secondary_bg(),
        );
        ui.painter().circle_stroke(
            knob_rect.center(),
            knob / 2.0,
            egui::Stroke::new(0.5, theme::border()),
        );
        if props.enabled && theme::pointer(response).clicked() {
            flipped = Some(!props.on);
        }
    });
    flipped
}
