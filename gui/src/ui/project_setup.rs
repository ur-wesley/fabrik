use eframe::egui::{self, Ui};

use crate::model::Action;

use super::{button, theme};

pub struct ProjectSetupProps {
    pub path: String,
    pub git: bool,
    pub busy: bool,
}

pub fn project_setup_banner(ui: &mut Ui, props: &ProjectSetupProps) -> Vec<Action> {
    let mut actions = Vec::new();
    theme::card_frame().show(ui, |ui| {
        ui.label(
            egui::RichText::new("Set up this project")
                .font(theme::font_headline())
                .color(theme::text_primary()),
        );
        ui.label(
            egui::RichText::new(format!("{} is not initialized for Fabrik yet.", props.path))
                .font(theme::font_callout())
                .color(theme::foreground_dimmed()),
        );
        if !props.git {
            ui.label(
                egui::RichText::new("No .git folder — run git init first for best results.")
                    .font(theme::font_footnote())
                    .color(theme::sys_orange()),
            );
        }
        ui.add_space(theme::SPACING);
        ui.horizontal(|ui| {
            let label = if props.busy {
                "Initializing…"
            } else {
                "Initialize with fabrik"
            };
            let btn = button::button(
                ui,
                &button::ButtonProps::new(label)
                    .variant(button::ButtonVariant::Secondary)
                    .min_size(egui::vec2(180.0, theme::CONTROL_H)),
            );
            if !props.busy && btn.clicked() {
                actions.push(Action::RunProjectInit);
            }
            if props.busy {
                ui.add(egui::Spinner::new());
            }
        });
    });
    actions
}
