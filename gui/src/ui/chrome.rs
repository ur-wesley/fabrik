use eframe::egui::{self, Ui};

use crate::model::{Action, Screen, ViewMode};

use super::{button, screen_segment, search_field, segment, theme};

pub struct RouteToolbarView {
    pub screen: Screen,
    pub view_mode: ViewMode,
    pub search: String,
    pub search_focused: bool,
    pub search_hint: &'static str,
    pub show_list_kanban: bool,
    pub show_rebuild: bool,
    pub rebuild_label: &'static str,
    pub rebuild_busy: bool,
}

pub fn window_nav(ui: &mut Ui, screen: Screen) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.allocate_ui_with_layout(
        egui::vec2(ui.available_width(), theme::HEADER_H),
        egui::Layout::left_to_right(egui::Align::Center),
        |ui| {
            ui.spacing_mut().item_spacing.x = 8.0;
            ui.spacing_mut().interact_size.y = theme::CONTROL_H;
            if let Some(next) = screen_segment::screen_segment(ui, screen) {
                actions.push(Action::SetScreen(next));
            }
        },
    );
    actions
}

pub fn route_toolbar(ui: &mut Ui, view: &mut RouteToolbarView) -> Vec<Action> {
    let mut actions = Vec::new();
    let focus_request = view.search_focused;
    ui.allocate_ui_with_layout(
        egui::vec2(ui.available_width(), theme::HEADER_H),
        egui::Layout::left_to_right(egui::Align::Center),
        |ui| {
            ui.spacing_mut().item_spacing.x = 6.0;
            ui.spacing_mut().interact_size.y = theme::CONTROL_H;
            ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
                ui.spacing_mut().item_spacing.x = 6.0;
                if view.show_rebuild {
                    ui.add_enabled_ui(!view.rebuild_busy, |ui| {
                        let btn = button::button(
                            ui,
                            &button::ButtonProps::new(view.rebuild_label)
                                .variant(button::ButtonVariant::Outline)
                                .min_size(egui::vec2(96.0, theme::CONTROL_H)),
                        );
                        if btn.clicked() {
                            actions.push(Action::RunGraphBuild);
                        }
                    });
                    if view.rebuild_busy {
                        ui.add(egui::Spinner::new());
                    }
                }
                if view.show_list_kanban {
                    if let Some(mode) = segment::segment(
                        ui,
                        &segment::SegmentProps {
                            view: view.view_mode,
                        },
                    ) {
                        actions.push(Action::SetView(mode));
                    }
                }
                ui.with_layout(egui::Layout::left_to_right(egui::Align::Center), |ui| {
                    ui.spacing_mut().item_spacing.x = 6.0;
                    let mut search_props = search_field::SearchFieldProps {
                        value: view.search.clone(),
                        focused: view.search_focused,
                        hint: view.search_hint,
                    };
                    let (_, changed) = search_field::search_field(ui, &mut search_props);
                    view.search = search_props.value;
                    view.search_focused = search_props.focused;
                    if changed {
                        actions.push(match view.screen {
                            Screen::Memory => Action::SetMemSearch(view.search.clone()),
                            Screen::Lookup => Action::SetLookupQuery(view.search.clone()),
                            _ => Action::SetSearch(view.search.clone()),
                        });
                    }
                });
            });
        },
    );
    if focus_request && !view.search_focused {
        actions.push(match view.screen {
            Screen::Memory => Action::SetMemSearchFocus(false),
            Screen::Lookup => Action::SetLookupSearchFocus(false),
            _ => Action::SetBoardSearchFocus(false),
        });
    }
    actions
}

pub struct PaneFooterProps {
    pub live: bool,
    pub loading: bool,
    pub stale: bool,
    pub updated: String,
    pub project: String,
    pub issues: usize,
}

pub fn pane_shell<R: Default>(
    ui: &mut Ui,
    footer: &PaneFooterProps,
    tabs: impl FnOnce(&mut Ui) -> Vec<Action>,
    toolbar: impl FnOnce(&mut Ui) -> Vec<Action>,
    body: impl FnOnce(&mut Ui) -> (Vec<Action>, R),
) -> (Vec<Action>, R) {
    let mut actions = Vec::new();
    let mut out = R::default();
    ui.with_layout(egui::Layout::bottom_up(egui::Align::Min), |ui| {
        ui.spacing_mut().item_spacing.y = 0.0;
        super::footer::footer(
            ui,
            &super::footer::FooterProps {
                live: footer.live,
                loading: footer.loading,
                stale: footer.stale,
                updated: footer.updated.clone(),
                project: footer.project.clone(),
                issues: footer.issues,
            },
        );
        ui.with_layout(egui::Layout::top_down(egui::Align::Min), |ui| {
            ui.spacing_mut().item_spacing.y = 0.0;
            actions.extend(tabs(ui));
            egui::Frame::new()
                .fill(egui::Color32::TRANSPARENT)
                .inner_margin(egui::Margin {
                    left: theme::PAD_I8,
                    right: theme::PAD_I8,
                    top: theme::SPACING_I8,
                    bottom: 0,
                })
                .show(ui, |ui| {
                    actions.extend(toolbar(ui));
                });
            egui::Frame::new()
                .fill(egui::Color32::TRANSPARENT)
                .inner_margin(egui::Margin {
                    left: theme::PAD_I8,
                    right: theme::PAD_I8,
                    top: theme::SPACING_I8,
                    bottom: theme::PAD_I8,
                })
                .show(ui, |ui| {
                    let (body_actions, r) = body(ui);
                    actions.extend(body_actions);
                    out = r;
                });
        });
    });
    (actions, out)
}
