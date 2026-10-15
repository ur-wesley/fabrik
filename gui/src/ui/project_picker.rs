use std::path::{Path, PathBuf};

use eframe::egui::{self, Key, Modifiers, ScrollArea, Sense, Stroke, Ui};

use crate::discover;
use crate::model::{Action, RecentRepo};

use super::picker_rows::{
    PickerItem, clamp_selection, picker_empty_state, picker_items, render_picker_sections,
    step_selection,
};
use super::{icons, theme};

const PICKER_WIDTH: f32 = 600.0;
pub const PICKER_LIST_H: f32 = 320.0;
const PICKER_PAD: i8 = 16;

pub struct PickerProps {
    pub tabs: Vec<RecentRepo>,
    pub recents: Vec<RecentRepo>,
    pub active: Option<PathBuf>,
    pub query: String,
    pub selected: usize,
}

pub struct BodyOut {
    pub query: Option<String>,
    pub delta: Option<i32>,
    pub chosen: Option<PathBuf>,
    pub close: bool,
}

/// VSCode-style `Ctrl+P` project picker: centered modal, autofocus filter
/// input, typed-path escape hatch pinned at row 0, then open tabs, then
/// recents + neighbor dirs ranked by [`discover::suggest_fuzzy`].
#[allow(clippy::too_many_lines)]
pub fn project_picker(ctx: &egui::Context, props: &PickerProps) -> Vec<Action> {
    let picker_id = egui::Id::new("project_picker");
    let focus_id = picker_id.with("autofocus");
    let mut query = props.query.clone();

    let seeds: Vec<PathBuf> = props
        .recents
        .iter()
        .map(|recent| recent.path.clone())
        .chain(props.active.clone())
        .collect();
    let neighbors = discover::fabrik_neighbors(&seeds);
    let items = picker_items(props, &neighbors);
    let selected = clamp_selection(props.selected, items.len());

    let frame = theme::dialog_frame(ctx.global_style().as_ref());
    let modal = egui::Modal::new(picker_id)
        .frame(frame)
        .backdrop_color(theme::dialog_backdrop())
        .show(ctx, |ui| {
            ui.set_min_width(PICKER_WIDTH);
            ui.set_max_width(PICKER_WIDTH);
            egui::Frame::new()
                .fill(theme::grouped_bg())
                .inner_margin(egui::Margin {
                    left: 0,
                    right: 0,
                    top: PICKER_PAD,
                    bottom: PICKER_PAD,
                })
                .show(ui, |ui| {
                    picker_body(
                        ui,
                        picker_id,
                        &mut query,
                        &items,
                        selected,
                        props.active.as_deref(),
                        &props.recents,
                    )
                })
                .inner
        });
    let should_close = modal.should_close();
    let out = modal.inner;
    let mut actions = Vec::new();
    if let Some(next) = out.query
        && next != props.query
    {
        actions.push(Action::SetPickerQuery(next));
    }
    if let Some(delta) = out.delta {
        actions.push(Action::MovePicker(delta));
    }
    if let Some(path) = out.chosen {
        actions.push(Action::PickRecent(path));
        actions.push(Action::ClosePicker);
        ctx.data_mut(|data| data.insert_temp(focus_id, true));
    } else if out.close || should_close {
        actions.push(Action::ClosePicker);
        ctx.data_mut(|data| data.insert_temp(focus_id, true));
    }
    actions
}

fn picker_directory_start(active: Option<&Path>, recents: &[RecentRepo]) -> PathBuf {
    if let Some(parent) = active.and_then(|active| active.parent()) {
        return parent.to_path_buf();
    }
    if let Some(parent) = recents.first().and_then(|recent| recent.path.parent()) {
        return parent.to_path_buf();
    }
    std::env::current_dir().unwrap_or_else(|_| PathBuf::from("."))
}

fn pick_project_directory(start: &Path) -> Option<PathBuf> {
    let picked = rfd::FileDialog::new()
        .set_title("Open project")
        .set_directory(start)
        .pick_folder();
    picked.and_then(discover::workspace_from_picked)
}

fn picker_body(
    ui: &mut Ui,
    picker_id: egui::Id,
    query: &mut String,
    items: &[PickerItem],
    selected: usize,
    active: Option<&Path>,
    recents: &[RecentRepo],
) -> BodyOut {
    let query_id = picker_id.with("query");
    let focus_id = picker_id.with("autofocus");
    let mut out = BodyOut {
        query: None,
        delta: None,
        chosen: None,
        close: false,
    };

    let h_pad = f32::from(PICKER_PAD);
    let mut search_out = SearchBarOut {
        changed: false,
        picked: None,
    };
    ui.horizontal(|ui| {
        ui.add_space(h_pad);
        let search_w = ui.available_width() - h_pad;
        search_out = ui
            .allocate_ui(egui::vec2(search_w, theme::PICKER_SEARCH_H), |ui| {
                picker_search_bar(ui, query_id, focus_id, query, active, recents)
            })
            .inner;
        ui.add_space(h_pad);
    });
    if search_out.changed {
        out.query = Some(query.clone());
    }
    if let Some(path) = search_out.picked {
        out.chosen = Some(path);
    }

    ui.add_space(theme::SPACING);

    let down = ui.input_mut(|input| input.consume_key(Modifiers::NONE, Key::ArrowDown));
    let up = ui.input_mut(|input| input.consume_key(Modifiers::NONE, Key::ArrowUp));
    if down {
        out.delta = Some(1);
        ui.memory_mut(|mem| mem.request_focus(query_id));
    }
    if up {
        out.delta = Some(-1);
        ui.memory_mut(|mem| mem.request_focus(query_id));
    }
    let esc = ui.input_mut(|input| input.consume_key(Modifiers::NONE, Key::Escape));
    if esc {
        out.close = true;
    }

    let preview = step_selection(selected, items.len(), out.delta.unwrap_or(0));

    theme::fill_width(ui);
    ScrollArea::vertical()
        .id_salt("project_picker_list")
        .max_height(PICKER_LIST_H)
        .auto_shrink([false, false])
        .show(ui, |ui| {
            egui::Frame::new()
                .inner_margin(egui::Margin::symmetric(PICKER_PAD, 0))
                .show(ui, |ui| {
                    theme::fill_width(ui);
                    if items.is_empty() {
                        picker_empty_state(ui);
                        return;
                    }
                    render_picker_sections(ui, items, preview, active, &mut out);
                });
        });

    let enter = ui.input_mut(|input| input.consume_key(Modifiers::NONE, Key::Enter));
    if enter && out.chosen.is_none() {
        out.chosen = items.get(selected).map(|item| item.path.clone());
        ui.memory_mut(|mem| mem.request_focus(query_id));
    }
    out
}

struct SearchBarOut {
    changed: bool,
    picked: Option<PathBuf>,
}

fn picker_search_bar(
    ui: &mut Ui,
    query_id: egui::Id,
    focus_id: egui::Id,
    query: &mut String,
    active: Option<&Path>,
    recents: &[RecentRepo],
) -> SearchBarOut {
    const FIELD_INSET_X: f32 = 10.0;
    const FIELD_INSET_Y: f32 = 5.0;
    const BETWEEN: f32 = theme::SPACING;

    let bar_h = theme::PICKER_SEARCH_H;
    let folder_size = egui::vec2(bar_h, bar_h);
    let row_w = ui.available_width();
    let search_w = row_w - folder_size.x - BETWEEN;

    let mut changed = false;
    let mut picked = None;

    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = 0.0;

        let (rect, _) = ui.allocate_exact_size(egui::vec2(search_w, bar_h), Sense::hover());
        ui.painter().rect(
            rect,
            theme::radius_picker_search(),
            theme::search_fill(),
            Stroke::NONE,
            egui::StrokeKind::Inside,
        );
        let inner = rect.shrink2(egui::vec2(FIELD_INSET_X, FIELD_INSET_Y));
        let mut child = ui.new_child(
            egui::UiBuilder::new()
                .max_rect(inner)
                .layout(egui::Layout::left_to_right(egui::Align::Center)),
        );
        child.spacing_mut().item_spacing.x = 6.0;
        child.label(
            egui::RichText::new(icons::MAGNIFYING_GLASS)
                .size(14.0)
                .color(theme::text_secondary()),
        );
        let response = child.add(
            egui::TextEdit::singleline(query)
                .id(query_id)
                .hint_text(
                    egui::RichText::new("type to filter projects, Enter to open")
                        .color(theme::text_placeholder()),
                )
                .text_color(theme::foreground())
                .frame(egui::Frame::NONE)
                .lock_focus(true)
                .desired_width(f32::INFINITY)
                .margin(egui::Margin::symmetric(2, 5)),
        );
        if response.changed() {
            changed = true;
        }
        let steal = child.data_mut(|data| {
            let slot = data.get_temp_mut_or_insert_with(focus_id, || true);
            let first = *slot;
            *slot = false;
            first
        });
        if steal {
            response.request_focus();
        }

        ui.add_space(BETWEEN);
        let folder_btn = picker_folder_button(ui, folder_size);
        if folder_btn.clicked() {
            let start = picker_directory_start(active, recents);
            picked = pick_project_directory(&start);
        }
    });

    SearchBarOut { changed, picked }
}

fn picker_folder_button(ui: &mut Ui, size: egui::Vec2) -> egui::Response {
    let (rect, response) = ui.allocate_exact_size(size, Sense::click());
    let fill = if response.is_pointer_button_down_on() {
        theme::control_fill_pressed()
    } else if response.hovered() {
        theme::control_fill_hover()
    } else {
        theme::search_fill()
    };
    ui.painter().rect(
        rect,
        theme::radius_picker_search(),
        fill,
        Stroke::NONE,
        egui::StrokeKind::Inside,
    );
    ui.painter().text(
        rect.center(),
        egui::Align2::CENTER_CENTER,
        icons::FOLDER,
        egui::FontId::proportional(14.0),
        theme::accent(),
    );
    theme::pointer(response)
}
