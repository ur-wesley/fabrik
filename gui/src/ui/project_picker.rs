use std::path::{Path, PathBuf};

use eframe::egui::{self, Key, Margin, Modifiers, ScrollArea, Sense, Stroke, Ui};

use crate::discover;
use crate::model::{Action, RecentRepo, recent_from_path};

use super::{icons, theme};

const PICKER_WIDTH: f32 = 600.0;
const PICKER_LIST_H: f32 = 320.0;
const PICKER_PAD: i8 = 16;

pub struct PickerProps {
    pub tabs: Vec<RecentRepo>,
    pub recents: Vec<RecentRepo>,
    pub active: Option<PathBuf>,
    pub query: String,
    pub selected: usize,
}

struct PickerItem {
    path: PathBuf,
    name: String,
    parent: String,
    is_tab: bool,
    is_typed: bool,
    is_new: bool,
}

struct BodyOut {
    query: Option<String>,
    delta: Option<i32>,
    chosen: Option<PathBuf>,
    close: bool,
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
    let neighbors = discover::beads_neighbors(&seeds);
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

fn picker_items(props: &PickerProps, neighbors: &[PathBuf]) -> Vec<PickerItem> {
    let typed = discover::typed_beads(&props.query);
    let filtered_tabs = discover::suggest_fuzzy(&props.query, &props.tabs);
    let mut pool: Vec<RecentRepo> = Vec::new();
    for recent in &props.recents {
        if props.tabs.iter().any(|tab| tab.path == recent.path) {
            continue;
        }
        pool.push(recent.clone());
    }
    for neighbor in neighbors {
        if props.tabs.iter().any(|tab| tab.path == *neighbor) {
            continue;
        }
        if pool.iter().any(|recent| recent.path == *neighbor) {
            continue;
        }
        pool.push(recent_from_path(neighbor.clone()));
    }
    let filtered_recents = discover::suggest_fuzzy(&props.query, &pool);
    let mut items = Vec::new();
    if let Some(ref typed_path) = typed {
        if let Some(tab) = filtered_tabs.iter().find(|tab| tab.path == *typed_path) {
            items.push(typed_item(tab, true));
        } else if let Some(recent) = filtered_recents
            .iter()
            .find(|item| item.path == *typed_path)
        {
            items.push(typed_item(recent, false));
        } else {
            let fresh = recent_from_path(typed_path.clone());
            items.push(PickerItem {
                path: fresh.path,
                name: fresh.name,
                parent: fresh.parent,
                is_tab: false,
                is_typed: true,
                is_new: true,
            });
        }
    }
    for tab in &filtered_tabs {
        if typed.as_ref().is_some_and(|path| path == &tab.path) {
            continue;
        }
        items.push(PickerItem {
            path: tab.path.clone(),
            name: tab.name.clone(),
            parent: tab.parent.clone(),
            is_tab: true,
            is_typed: false,
            is_new: false,
        });
    }
    for recent in &filtered_recents {
        if typed.as_ref().is_some_and(|path| path == &recent.path) {
            continue;
        }
        items.push(PickerItem {
            path: recent.path.clone(),
            name: recent.name.clone(),
            parent: recent.parent.clone(),
            is_tab: false,
            is_typed: false,
            is_new: false,
        });
    }
    items
}

fn typed_item(recent: &RecentRepo, is_tab: bool) -> PickerItem {
    PickerItem {
        path: recent.path.clone(),
        name: recent.name.clone(),
        parent: recent.parent.clone(),
        is_tab,
        is_typed: true,
        is_new: false,
    }
}

const fn clamp_selection(selected: usize, len: usize) -> usize {
    if len == 0 {
        0
    } else {
        let max = len.saturating_sub(1);
        if selected > max { max } else { selected }
    }
}

fn step_selection(selected: usize, len: usize, delta: i32) -> usize {
    if len == 0 {
        return 0;
    }
    let clamped = clamp_selection(selected, len);
    let current = i32::try_from(clamped).unwrap_or(0);
    let max = i32::try_from(len.saturating_sub(1)).unwrap_or(0);
    let next = current.saturating_add(delta).clamp(0, max);
    usize::try_from(next).unwrap_or(0)
}

fn picker_directory_start(active: Option<&Path>, recents: &[RecentRepo]) -> PathBuf {
    if let Some(active) = active {
        if let Some(parent) = active.parent() {
            return parent.to_path_buf();
        }
    }
    if let Some(recent) = recents.first() {
        if let Some(parent) = recent.path.parent() {
            return parent.to_path_buf();
        }
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

fn picker_empty_state(ui: &mut Ui) {
    let width = ui.available_width();
    ui.allocate_ui_with_layout(
        egui::vec2(width, PICKER_LIST_H),
        egui::Layout::centered_and_justified(egui::Direction::TopDown),
        |ui| {
            ui.label(
                egui::RichText::new("no matching projects")
                    .font(theme::font_callout())
                    .color(theme::text_secondary()),
            );
        },
    );
}

fn picker_list_frame() -> egui::Frame {
    egui::Frame::new()
        .fill(theme::grouped_secondary_bg())
        .stroke(Stroke::new(1.0, theme::border_hover()))
        .corner_radius(theme::radius_card())
        .inner_margin(Margin::ZERO)
}

fn render_picker_sections(
    ui: &mut Ui,
    items: &[PickerItem],
    preview: usize,
    active: Option<&Path>,
    out: &mut BodyOut,
) {
    let mut idx = 0;
    if idx < items.len() && items[idx].is_typed {
        let typed_end = items.iter().position(|item| !item.is_typed).unwrap_or(items.len());
        picker_list_frame().show(ui, |ui| {
            theme::fill_width(ui);
            for (offset, item) in items[..typed_end].iter().enumerate() {
                let last = offset + 1 == typed_end;
                if picker_row(ui, item, offset == preview, active, offset == 0, !last) {
                    out.chosen = Some(item.path.clone());
                }
            }
        });
        idx = typed_end;
        ui.add_space(theme::SPACING);
    }
    if idx < items.len() && items[idx].is_tab {
        let tab_start = idx;
        while idx < items.len() && items[idx].is_tab {
            idx = idx.saturating_add(1);
        }
        section_label(ui, "Open tabs");
        picker_list_frame().show(ui, |ui| {
            theme::fill_width(ui);
            for (offset, item) in items[tab_start..idx].iter().enumerate() {
                let flat = tab_start + offset;
                let last = offset + 1 == idx - tab_start;
                if picker_row(ui, item, flat == preview, active, offset == 0, !last) {
                    out.chosen = Some(item.path.clone());
                }
            }
        });
        ui.add_space(theme::SPACING);
    }
    if idx < items.len() {
        let rec_start = idx;
        section_label(ui, "Recents & neighbors");
        picker_list_frame().show(ui, |ui| {
            theme::fill_width(ui);
            let count = items.len().saturating_sub(rec_start);
            for (offset, item) in items[rec_start..].iter().enumerate() {
                let flat = rec_start + offset;
                let last = offset + 1 == count;
                if picker_row(ui, item, flat == preview, active, offset == 0, !last) {
                    out.chosen = Some(item.path.clone());
                }
            }
        });
    }
}

fn section_label(ui: &mut Ui, text: &str) {
    ui.label(
        egui::RichText::new(text)
            .font(theme::font_footnote())
            .color(theme::text_secondary()),
    );
    ui.add_space(4.0);
}

fn picker_row_corners(first: bool, last: bool) -> egui::CornerRadius {
    let r = theme::radius_card();
    if first && last {
        r
    } else if first {
        egui::CornerRadius {
            nw: r.nw,
            ne: r.ne,
            sw: 0,
            se: 0,
        }
    } else if last {
        egui::CornerRadius {
            nw: 0,
            ne: 0,
            sw: r.sw,
            se: r.se,
        }
    } else {
        egui::CornerRadius::ZERO
    }
}

fn picker_row_fill(is_selected: bool, hovered: bool) -> egui::Color32 {
    if is_selected {
        theme::row_selected_fill()
    } else if hovered {
        theme::surface_hover()
    } else {
        egui::Color32::TRANSPARENT
    }
}

fn picker_row(
    ui: &mut Ui,
    item: &PickerItem,
    is_selected: bool,
    active: Option<&Path>,
    first: bool,
    show_separator: bool,
) -> bool {
    let is_active = active.is_some_and(|path| path == item.path.as_path());
    let last = !show_separator;
    let row_h = theme::ROW_HEIGHT.max(48.0);
    let width = ui.available_width();
    let (rect, response) = ui.allocate_exact_size(egui::vec2(width, row_h), Sense::click());
    let hovered = response.hovered();
    let fill = picker_row_fill(is_selected, hovered);
    if fill != egui::Color32::TRANSPARENT {
        ui.painter().rect_filled(rect, picker_row_corners(first, last), fill);
    }
    let icon = if item.is_new || item.is_typed {
        icons::PLUS
    } else {
        icons::FOLDER
    };
    let title = if item.is_new {
        format!("add {}", item.name)
    } else if item.is_typed {
        format!("open {}", item.name)
    } else {
        item.name.clone()
    };
    let plus_row = item.is_new || item.is_typed;
    let icon_color = if plus_row && !is_selected {
        theme::accent()
    } else {
        theme::text_primary()
    };
    let inner = rect.shrink2(egui::vec2(theme::PAD, 4.0));
    let icon_w = 20.0;
    let active_mark = if is_active { 14.0 } else { 0.0 };
    let text_x = inner.min.x + icon_w + 10.0 + active_mark;
    ui.painter().text(
        egui::pos2(inner.min.x + icon_w * 0.5, inner.center().y),
        egui::Align2::CENTER_CENTER,
        icon,
        egui::FontId::proportional(17.0),
        icon_color,
    );
    if is_active {
        ui.painter().text(
            egui::pos2(inner.min.x + icon_w + 10.0, inner.center().y),
            egui::Align2::LEFT_CENTER,
            icons::CHECK,
            egui::FontId::proportional(11.0),
            theme::accent(),
        );
    }
    const LINE_GAP: f32 = 2.0;
    const TITLE_H: f32 = 14.0;
    const SUB_H: f32 = 12.0;
    let block_h = TITLE_H + LINE_GAP + SUB_H;
    let mid = inner.center().y;
    let title_y = mid - block_h * 0.5 + TITLE_H * 0.5;
    let sub_y = mid + block_h * 0.5 - SUB_H * 0.5;
    ui.painter().text(
        egui::pos2(text_x, title_y),
        egui::Align2::LEFT_CENTER,
        &title,
        theme::font_callout(),
        theme::text_primary(),
    );
    ui.painter().text(
        egui::pos2(text_x, sub_y),
        egui::Align2::LEFT_CENTER,
        &item.parent,
        theme::font_footnote(),
        theme::text_secondary(),
    );
    if show_separator {
        let sep_y = rect.max.y - 0.5;
        ui.painter().hline(
            (inner.min.x + theme::PAD)..=inner.max.x,
            sep_y,
            theme::hairline(),
        );
    }
    if is_selected {
        response.scroll_to_me(None);
    }
    theme::pointer(response).clicked()
}

#[cfg(test)]
mod tests {
    use std::path::{Path, PathBuf};

    use crate::model::recent_from_path;

    use super::{PickerProps, clamp_selection, picker_items, step_selection};

    fn props_with(tabs: &[PathBuf], recents: &[PathBuf], query: &str) -> PickerProps {
        PickerProps {
            tabs: tabs.iter().cloned().map(recent_from_path).collect(),
            recents: recents.iter().cloned().map(recent_from_path).collect(),
            active: tabs.first().cloned(),
            query: query.to_owned(),
            selected: 0,
        }
    }

    fn beads_dir(tag: &str) -> PathBuf {
        let dir = std::env::temp_dir().join(format!("bd-neon-picker-{tag}"));
        let _ = std::fs::create_dir_all(dir.join(".beads"));
        dir
    }

    #[test]
    fn typed_valid_path_pins_row_zero() {
        let typed = beads_dir("typed-top");
        let props = props_with(
            &[PathBuf::from("/work/alpha"), PathBuf::from("/work/beta")],
            &[PathBuf::from("/work/gamma")],
            typed.to_str().unwrap_or("/work"),
        );
        let items = picker_items(&props, &[]);
        assert!(!items.is_empty());
        assert!(
            items
                .first()
                .is_some_and(|first| first.path == typed && first.is_typed)
        );
        let _ = std::fs::remove_dir_all(&typed);
    }

    #[test]
    fn typed_matching_tab_moves_to_top_without_dup() {
        let tab_a = beads_dir("tab-a");
        let tab_b = beads_dir("tab-b");
        let props = props_with(
            &[tab_a.clone(), tab_b.clone()],
            &[],
            tab_b.to_str().unwrap_or("/work"),
        );
        let items = picker_items(&props, &[]);
        assert!(
            items
                .first()
                .is_some_and(|first| first.path == tab_b && first.is_typed)
        );
        assert_eq!(items.iter().filter(|item| item.path == tab_b).count(), 1);
        let _ = std::fs::remove_dir_all(&tab_a);
        let _ = std::fs::remove_dir_all(&tab_b);
    }

    #[test]
    fn fuzzy_query_filters_with_tabs_first() {
        let props = props_with(
            &[PathBuf::from("/work/neon-tabs")],
            &[
                PathBuf::from("/work/neon-helper"),
                PathBuf::from("/work/unrelated-zzz"),
            ],
            "neon",
        );
        let items = picker_items(&props, &[]);
        assert!(items.iter().all(|item| item.is_typed || {
            let text = item.path.to_string_lossy().to_ascii_lowercase();
            text.contains("neon") || item.name.to_ascii_lowercase().contains("neon")
        }));
        assert!(
            items
                .iter()
                .all(|item| item.path.as_path() != Path::new("/work/unrelated-zzz"))
        );
        let first_tab = items.iter().position(|item| item.is_tab);
        let first_recent = items.iter().position(|item| !item.is_tab && !item.is_typed);
        if let (Some(tab), Some(recent)) = (first_tab, first_recent) {
            assert!(tab < recent);
        }
    }

    #[test]
    fn no_match_no_typed_yields_empty() {
        let props = props_with(
            &[PathBuf::from("/work/alpha")],
            &[PathBuf::from("/work/beta")],
            "zzz-no-such-project",
        );
        let items = picker_items(&props, &[]);
        assert!(items.is_empty());
    }

    #[test]
    fn empty_query_keeps_tabs_then_recents_order() {
        let props = props_with(
            &[PathBuf::from("/work/t1"), PathBuf::from("/work/t2")],
            &[PathBuf::from("/work/r1")],
            "",
        );
        let items = picker_items(&props, &[]);
        let paths: Vec<PathBuf> = items.iter().map(|item| item.path.clone()).collect();
        assert_eq!(
            paths,
            vec![
                PathBuf::from("/work/t1"),
                PathBuf::from("/work/t2"),
                PathBuf::from("/work/r1"),
            ]
        );
    }

    #[test]
    fn clamp_and_step_stay_in_range() {
        assert_eq!(clamp_selection(5, 3), 2);
        assert_eq!(clamp_selection(0, 0), 0);
        assert_eq!(clamp_selection(1, 2), 1);
        assert_eq!(step_selection(0, 3, -1), 0);
        assert_eq!(step_selection(2, 3, 1), 2);
        assert_eq!(step_selection(0, 3, 1), 1);
        assert_eq!(step_selection(2, 3, -1), 1);
        assert_eq!(step_selection(9, 3, 0), 2);
        assert_eq!(step_selection(0, 0, 1), 0);
    }
}
