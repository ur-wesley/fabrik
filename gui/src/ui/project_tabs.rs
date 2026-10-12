use std::path::PathBuf;

use eframe::egui::{self, Align2, Color32, CornerRadius, Id, PointerButton, ScrollArea, Sense, Ui};

use crate::model::{Action, RecentRepo};

use super::{icons, theme};

pub struct ProjectTabsProps {
    pub tabs: Vec<RecentRepo>,
    pub active: Option<PathBuf>,
    pub loading: bool,
}

pub fn project_tabs(ui: &mut Ui, props: &ProjectTabsProps) -> Vec<Action> {
    let mut actions = Vec::new();
    let width = ui.available_width();
    let (rect, _) = ui.allocate_exact_size(egui::vec2(width, theme::TAB_H), egui::Sense::hover());
    ui.painter().rect(
        rect,
        CornerRadius::ZERO,
        theme::muted(),
        theme::glass_stroke(),
        egui::StrokeKind::Inside,
    );
    let mut child = ui.new_child(
        egui::UiBuilder::new()
            .max_rect(rect)
            .layout(egui::Layout::left_to_right(egui::Align::Center)),
    );
    child.set_clip_rect(rect);
    child.set_height(theme::TAB_H);
    child.spacing_mut().item_spacing.x = 0.0;
    ScrollArea::horizontal()
        .id_salt("project_tabs")
        .auto_shrink([false, false])
        .show(&mut child, |ui| {
            ui.set_height(theme::TAB_H);
            ui.spacing_mut().item_spacing.x = 0.0;
            if tab_add_project(ui).clicked() {
                actions.push(Action::TogglePicker);
            }
            if props.tabs.is_empty() {
                ui.allocate_ui_with_layout(
                    egui::vec2(theme::TAB_MIN_W + theme::PAD, theme::TAB_H),
                    egui::Layout::left_to_right(egui::Align::Center),
                    |ui| {
                        ui.add_space(theme::PAD);
                        ui.label(
                            egui::RichText::new("no open projects")
                                .small()
                                .color(theme::dim_text()),
                        );
                    },
                );
            }
            for tab in &props.tabs {
                if let Some(action) = tab_chip(ui, tab, props) {
                    actions.push(action);
                }
            }
        });
    actions
}

fn tab_add_project(ui: &mut Ui) -> egui::Response {
    let size = egui::vec2(theme::TAB_H, theme::TAB_H);
    let (rect, response) = ui.allocate_exact_size(size, Sense::click());
    if ui.is_rect_visible(rect) {
        let hovered = response.hovered();
        if hovered {
            ui.painter()
                .rect_filled(rect, CornerRadius::ZERO, theme::surface_hover());
        }
        let divider = egui::Rect::from_min_max(egui::pos2(rect.max.x - 1.0, rect.min.y), rect.max);
        ui.painter()
            .rect_filled(divider, CornerRadius::ZERO, theme::border());
        let fg = if hovered {
            theme::foreground()
        } else {
            theme::dim_text()
        };
        ui.painter().text(
            rect.center(),
            Align2::CENTER_CENTER,
            icons::PLUS,
            egui::TextStyle::Body.resolve(ui.style()),
            fg,
        );
    }
    theme::pointer(response).on_hover_text("Open project")
}

fn tab_chip(ui: &mut Ui, tab: &RecentRepo, props: &ProjectTabsProps) -> Option<Action> {
    let active = props.active.as_ref().is_some_and(|path| path == &tab.path);
    let label = tab_label(props.tabs.as_slice(), tab);
    let tab_w = tab_width(ui, &label, active && props.loading);
    let (rect, response) =
        ui.allocate_exact_size(egui::vec2(tab_w, theme::TAB_H), egui::Sense::click());
    let tab_hovered = response.hovered();
    let fill = if active {
        theme::accent_secondary_fill()
    } else if tab_hovered {
        theme::surface_hover()
    } else {
        Color32::TRANSPARENT
    };
    if ui.is_rect_visible(rect) {
        ui.painter().rect_filled(rect, CornerRadius::ZERO, fill);
        let divider = egui::Rect::from_min_max(egui::pos2(rect.max.x - 1.0, rect.min.y), rect.max);
        ui.painter()
            .rect_filled(divider, CornerRadius::ZERO, theme::border());
    }
    let inner = egui::Rect::from_min_max(
        egui::pos2(rect.min.x + theme::TAB_PAD_LEFT, rect.min.y),
        egui::pos2(rect.max.x - theme::TAB_PAD_RIGHT, rect.max.y),
    );
    let inner_h = inner.height();
    let mut child = ui.new_child(
        egui::UiBuilder::new()
            .max_rect(inner)
            .layout(egui::Layout::top_down(egui::Align::Center)),
    );
    child.set_clip_rect(inner);
    let close_col_w = theme::TAB_LABEL_GAP + theme::TAB_CLOSE_W;
    let label_col_w = (inner.width() - close_col_w).max(0.0);
    let text_color = if active || tab_hovered {
        theme::foreground()
    } else {
        theme::dim_text()
    };
    let mut close = false;
    child.vertical_centered(|ui| {
        ui.horizontal(|ui| {
            ui.set_width(inner.width());
            ui.spacing_mut().item_spacing.x = 0.0;
            ui.allocate_ui_with_layout(
                egui::vec2(label_col_w, inner_h),
                egui::Layout::left_to_right(egui::Align::Center),
                |ui| {
                    ui.spacing_mut().item_spacing.x = theme::TAB_PAD_X;
                    if active && props.loading {
                        ui.add(egui::Spinner::new().size(10.0));
                    }
                    let label_w = ui.available_width().max(0.0);
                    ui.set_max_width(label_w);
                    ui.add(
                        egui::Label::new(egui::RichText::new(label).small().color(text_color))
                            .truncate()
                            .selectable(false),
                    );
                },
            );
            ui.add_space(theme::TAB_LABEL_GAP);
            let close_response = tab_close(ui, rect, &tab.path);
            close = close_response.clicked();
        });
    });
    let response = theme::pointer(response);
    if close || response.clicked_by(PointerButton::Middle) {
        Some(Action::CloseTab(tab.path.clone()))
    } else if response.clicked() {
        Some(Action::PickRecent(tab.path.clone()))
    } else {
        None
    }
}

fn tab_close(ui: &mut Ui, tab_rect: egui::Rect, path: &PathBuf) -> egui::Response {
    let size = egui::vec2(theme::TAB_CLOSE_W, theme::TAB_CLOSE_W);
    let content_right = tab_rect.max.x - theme::TAB_PAD_RIGHT;
    let center = egui::pos2(content_right - size.x * 0.5, tab_rect.center().y);
    let hit = egui::Rect::from_center_size(center, size);
    let response = ui.interact(hit, Id::new(path).with("tab_close"), Sense::click());
    if ui.is_rect_visible(hit) {
        if response.hovered() {
            ui.painter().rect_filled(hit, theme::radius_sm(), theme::control_fill_hover());
        }
        let fg = if response.hovered() {
            theme::foreground()
        } else {
            theme::dim_text()
        };
        let font_id = egui::TextStyle::Small.resolve(ui.style());
        ui.painter()
            .text(hit.center(), Align2::CENTER_CENTER, icons::X, font_id, fg);
    }
    theme::pointer(response).on_hover_text("close")
}

fn tab_width(ui: &Ui, label: &str, spinning: bool) -> f32 {
    let font_id = egui::TextStyle::Small.resolve(ui.style());
    let galley = ui
        .painter()
        .layout_no_wrap(label.to_owned(), font_id, theme::foreground());
    let spinner = if spinning {
        10.0 + theme::TAB_PAD_X
    } else {
        0.0
    };
    let text_w = galley.size().x;
    let raw = theme::TAB_PAD_LEFT
        + spinner
        + text_w
        + theme::TAB_LABEL_GAP
        + theme::TAB_CLOSE_W
        + theme::TAB_PAD_RIGHT;
    raw.clamp(theme::TAB_MIN_W, theme::TAB_MAX_W)
}

fn tab_label(tabs: &[RecentRepo], tab: &RecentRepo) -> String {
    let duplicate = tabs.iter().filter(|item| item.name == tab.name).count() > 1;
    if duplicate {
        format!("{} — {}", tab.name, tab.parent)
    } else {
        tab.name.clone()
    }
}
