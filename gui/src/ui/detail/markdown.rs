use eframe::egui::{self, Ui};
use egui_commonmark::{CommonMarkCache, CommonMarkViewer};

use super::super::theme;

pub(super) fn markdown_card(
    ui: &mut Ui,
    cache: &mut CommonMarkCache,
    icon: &str,
    title: &str,
    body: Option<&str>,
    placeholder: &str,
) {
    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = theme::SPACING;
        ui.label(egui::RichText::new(icon).small().color(theme::dim_text()));
        ui.label(
            egui::RichText::new(title)
                .small()
                .strong()
                .color(theme::dim_text()),
        );
    });
    ui.add_space(theme::SPACING);

    let _ = theme::muted_frame()
        .inner_margin(egui::Margin::same(theme::PAD_I8))
        .show(ui, |ui| {
            theme::fill_width(ui);
            if let Some(content) = body {
                if content.trim().is_empty() {
                    ui.label(
                        egui::RichText::new(placeholder)
                            .small()
                            .italics()
                            .color(theme::dim_text()),
                    );
                } else {
                    CommonMarkViewer::new().show(ui, cache, content);
                }
            } else {
                ui.label(
                    egui::RichText::new(placeholder)
                        .small()
                        .italics()
                        .color(theme::dim_text()),
                );
            }
        });
}
