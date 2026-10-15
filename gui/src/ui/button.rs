use eframe::egui::{self, Color32, Response, Ui};

use super::theme;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ButtonVariant {
    Outline,
    Ghost,
    Secondary,
}

#[derive(Debug, Clone, Copy)]
pub struct ButtonProps<'a> {
    pub label: &'a str,
    pub variant: ButtonVariant,
    pub selected: bool,
    pub min_size: egui::Vec2,
}

impl<'a> ButtonProps<'a> {
    pub const fn new(label: &'a str) -> Self {
        Self {
            label,
            variant: ButtonVariant::Outline,
            selected: false,
            min_size: egui::Vec2::new(theme::CONTROL_H, theme::CONTROL_H),
        }
    }

    pub const fn variant(mut self, variant: ButtonVariant) -> Self {
        self.variant = variant;
        self
    }

    pub const fn selected(mut self, selected: bool) -> Self {
        self.selected = selected;
        self
    }

    pub const fn min_size(mut self, min_size: egui::Vec2) -> Self {
        self.min_size = min_size;
        self
    }
}

pub fn button(ui: &mut Ui, props: &ButtonProps<'_>) -> Response {
    let focused = ui.input(|input| input.viewport().focused) != Some(false);
    let (fill, stroke, text) = if props.selected {
        if focused {
            (
                theme::surface_active(),
                egui::Stroke::NONE,
                theme::on_accent_text(),
            )
        } else {
            (
                theme::control_fill(),
                theme::hairline(),
                theme::foreground(),
            )
        }
    } else {
        match props.variant {
            ButtonVariant::Secondary => (
                theme::accent_secondary_fill(),
                egui::Stroke::NONE,
                theme::accent(),
            ),
            ButtonVariant::Ghost => (Color32::TRANSPARENT, egui::Stroke::NONE, theme::accent()),
            ButtonVariant::Outline => (
                theme::control_fill(),
                theme::hairline(),
                theme::foreground(),
            ),
        }
    };
    theme::pointer(
        ui.add(
            egui::Button::new(egui::RichText::new(props.label).color(text))
                .fill(fill)
                .stroke(stroke)
                .corner_radius(theme::radius())
                .min_size(props.min_size),
        ),
    )
}
