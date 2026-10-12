use eframe::egui::{self, Sense, Ui};

use crate::model::ViewMode;

use super::{icons, theme};

pub struct SegmentProps {
    pub view: ViewMode,
}

const LIST_KANBAN_W: [f32; 2] = [72.0, 84.0];

pub fn segment(ui: &mut Ui, props: &SegmentProps) -> Option<ViewMode> {
    let list_label = icons::with_icon(icons::LIST, "List");
    let board_label = icons::with_icon(icons::COLUMNS, "Board");
    let labels = [list_label.as_str(), board_label.as_str()];
    let selected = match props.view {
        ViewMode::List => 0,
        ViewMode::Kanban => 1,
    };
    let picked = segmented_control(ui, ui.id().with("list_kanban"), &labels, &LIST_KANBAN_W, selected)?;
    Some(if picked == 0 { ViewMode::List } else { ViewMode::Kanban })
}

pub fn segmented_control(
    ui: &mut Ui,
    id: egui::Id,
    labels: &[&str],
    widths: &[f32],
    selected: usize,
) -> Option<usize> {
    let count = labels.len().min(widths.len());
    if count == 0 {
        return None;
    }
    let total_w = widths[..count].iter().sum::<f32>();
    let (rect, _) =
        ui.allocate_exact_size(egui::vec2(total_w, theme::SEGMENT_H), Sense::hover());
    ui.painter()
        .rect_filled(rect, theme::radius_segment(), theme::segmented_track());

    let mut picked = None;
    for index in 0..count {
        let seg_min =
            egui::pos2(rect.min.x + widths[..index].iter().sum::<f32>(), rect.min.y);
        let seg_rect =
            egui::Rect::from_min_size(seg_min, egui::vec2(widths[index], theme::SEGMENT_H));
        let is_selected = index == selected;
        if is_selected {
            let inset = seg_rect.shrink2(egui::vec2(2.0, 2.0));
            ui.painter().rect(
                inset,
                theme::radius_segment(),
                theme::segment_selected_fill(),
                theme::segment_selected_stroke(),
                egui::StrokeKind::Inside,
            );
        }
        let response = ui.interact(seg_rect, id.with(index), Sense::click());
        let text_color = if is_selected {
            theme::text_primary()
        } else {
            theme::text_secondary()
        };
        ui.painter().text(
            seg_rect.center(),
            egui::Align2::CENTER_CENTER,
            labels[index],
            theme::font_callout(),
            text_color,
        );
        if theme::pointer(response).clicked() {
            picked = Some(index);
        }
    }
    picked
}
