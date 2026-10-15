use eframe::egui::{self, ScrollArea, Ui};

use crate::model::Action;

use super::{chrome, theme};

const INIT_APPS: [(&str, &str); 4] = [
    ("cursor", "Cursor"),
    ("opencode", "OpenCode"),
    ("pi", "Pi"),
    ("antigravity", "Antigravity"),
];

pub struct SettingsView {
    pub active_path: Option<std::path::PathBuf>,
    pub tabs: Vec<crate::model::RecentRepo>,
    pub last_ok_label: String,
    pub live: bool,
    pub issue_count: usize,
    pub repo_path: String,
    pub loading: bool,
    pub stale: bool,
    pub init_apps: Vec<String>,
}

pub fn settings_pane(ui: &mut Ui, view: &SettingsView) -> Vec<Action> {
    let (actions, ()) = chrome::pane_shell(
        ui,
        &chrome::PaneFooterProps {
            live: view.live,
            loading: view.loading,
            stale: view.stale,
            updated: view.last_ok_label.clone(),
            project: view.repo_path.clone(),
            issues: view.issue_count,
        },
        |ui| {
            super::project_tabs::project_tabs(
                ui,
                &super::project_tabs::ProjectTabsProps {
                    tabs: view.tabs.clone(),
                    active: view.active_path.clone(),
                    loading: view.loading,
                },
            )
        },
        |_| Vec::new(),
        |ui| {
            let mut actions = Vec::new();
            ScrollArea::vertical()
                .id_salt("settings_scroll")
                .auto_shrink([false, false])
                .show(ui, |ui| {
                    theme::fill_width(ui);
                    ui.label(
                        egui::RichText::new("Settings")
                            .font(theme::font_title2())
                            .color(theme::text_primary()),
                    );
                    ui.add_space(theme::SPACING);
                    ui.label(
                        egui::RichText::new("New project init")
                            .font(theme::font_headline())
                            .color(theme::text_primary()),
                    );
                    ui.label(
                        egui::RichText::new(
                            "Apps configured when you initialize a folder from Neon.",
                        )
                        .font(theme::font_footnote())
                        .color(theme::text_secondary()),
                    );
                    ui.add_space(6.0);
                    actions.extend(init_app_toggles(ui, view));
                    ui.add_space(theme::SPACING);
                    ui.label(
                        egui::RichText::new("Fabrik CLI")
                            .font(theme::font_headline())
                            .color(theme::text_primary()),
                    );
                    ui.label(
                        egui::RichText::new(format!("Binary: {}", crate::fabrik::fabrik_bin()))
                            .font(theme::font_footnote())
                            .color(theme::text_secondary()),
                    );
                    ui.label(
                        egui::RichText::new("Override with FABRIK_BIN in the environment.")
                            .font(theme::font_caption2())
                            .color(theme::text_tertiary()),
                    );
                });
            (actions, ())
        },
    );
    actions
}

fn init_app_toggles(ui: &mut Ui, view: &SettingsView) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.vertical(|ui| {
        for (id, label) in INIT_APPS {
            let on = view.init_apps.iter().any(|a| a == id);
            let mut checked = on;
            if ui.checkbox(&mut checked, label).changed() {
                actions.push(Action::ToggleInitApp(id.to_owned()));
            }
        }
    });
    actions
}
