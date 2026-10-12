use eframe::egui::{self, Ui};

use crate::model::Action;

use super::{button, theme};

pub struct GraphSetupProps {
    pub rebuild: bool,
    pub busy: bool,
    pub body: String,
}

pub fn graph_setup_banner(ui: &mut Ui, props: &GraphSetupProps) -> Vec<Action> {
    let mut actions = Vec::new();
    theme::card_frame().show(ui, |ui| {
        ui.label(
            egui::RichText::new("Build the code graph")
                .font(theme::font_headline())
                .color(theme::text_primary()),
        );
        ui.label(
            egui::RichText::new(props.body.clone())
                .font(theme::font_callout())
                .color(theme::foreground_dimmed()),
        );
        ui.add_space(theme::SPACING);
        ui.horizontal(|ui| {
            let label = if props.busy {
                if props.rebuild {
                    "Rebuilding…"
                } else {
                    "Building…"
                }
            } else if props.rebuild {
                "Rebuild graph"
            } else {
                "Build graph"
            };
            let btn = button::button(
                ui,
                &button::ButtonProps::new(label)
                    .variant(button::ButtonVariant::Secondary)
                    .min_size(egui::vec2(180.0, theme::CONTROL_H)),
            );
            if !props.busy && btn.clicked() {
                actions.push(Action::RunGraphBuild);
            }
            if props.busy {
                ui.add(egui::Spinner::new());
            }
        });
    });
    actions
}
