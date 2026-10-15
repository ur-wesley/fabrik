use eframe::egui::{self, Align, Id, Layout, Order, Stroke, Ui};

use crate::toast::{ToastKind, ToastStore};

use super::{icons, theme};

const TOAST_W: f32 = 320.0;
const CLOSE_SIZE: f32 = 20.0;
const ICON_SIZE: f32 = 14.0;

pub fn toast_overlay(ctx: &egui::Context, store: &mut ToastStore) {
    let items = store.items();
    if items.is_empty() {
        return;
    }

    let mut dismiss: Vec<u64> = Vec::new();
    let edge = theme::PAD_CONTENT + theme::SPACING;
    let bottom = theme::FOOTER_H + edge;
    let content = ctx.content_rect();
    let toast_w = TOAST_W.min(content.width() - 2.0 * edge);

    egui::Area::new(Id::new("neon_toast_stack"))
        .order(Order::Foreground)
        .anchor(egui::Align2::RIGHT_BOTTOM, egui::vec2(-edge, -bottom))
        .show(ctx, |ui| {
            ui.set_width(toast_w);
            ui.spacing_mut().item_spacing.y = theme::SPACING;
            for toast in items.iter().rev() {
                if toast_row(ui, toast, toast_w) {
                    dismiss.push(toast.id);
                }
            }
        });

    for id in dismiss {
        store.dismiss(id);
    }
}

fn toast_row(ui: &mut Ui, toast: &crate::toast::Toast, toast_w: f32) -> bool {
    let mut dismiss = false;

    let card = theme::toast_frame().show(ui, |ui| {
        ui.set_max_width(toast_w);
        ui.set_min_width(toast_w);
        ui.with_layout(Layout::left_to_right(Align::TOP), |ui| {
            ui.spacing_mut().item_spacing.x = 6.0;
            let (icon, icon_color) = match toast.kind {
                ToastKind::Info => (icons::CHECK_CIRCLE, theme::foreground()),
                ToastKind::Error => (icons::WARNING, theme::sys_red()),
            };
            ui.label(
                egui::RichText::new(icon)
                    .font(theme::font_callout())
                    .size(ICON_SIZE)
                    .color(icon_color),
            );
            let text_w = (ui.available_width() - CLOSE_SIZE - 2.0).max(0.0);
            ui.allocate_ui(egui::vec2(text_w, 0.0), |ui| {
                ui.set_width(text_w);
                ui.add(
                    egui::Label::new(
                        egui::RichText::new(&toast.message)
                            .font(theme::font_callout())
                            .color(theme::foreground()),
                    )
                    .wrap(),
                );
            });
            if toast_close(ui) {
                dismiss = true;
            }
        });
    });

    if card.response.clicked() && !dismiss {
        dismiss = true;
    }
    dismiss
}

fn toast_close(ui: &mut Ui) -> bool {
    theme::pointer(
        ui.add(
            egui::Button::new(
                egui::RichText::new(icons::X)
                    .font(theme::font_footnote())
                    .color(theme::dim_text()),
            )
            .fill(egui::Color32::TRANSPARENT)
            .stroke(Stroke::NONE)
            .min_size(egui::vec2(CLOSE_SIZE, CLOSE_SIZE)),
        ),
    )
    .clicked()
}
