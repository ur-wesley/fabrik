use std::path::{Path, PathBuf};

use eframe::egui::{self, Margin, Stroke, Ui};

use crate::discover;
use crate::model::{RecentRepo, recent_from_path};

use super::project_picker::{BodyOut, PICKER_LIST_H, PickerProps};
use super::{icons, theme};

pub struct PickerItem {
    pub path: PathBuf,
    pub name: String,
    pub parent: String,
    pub is_tab: bool,
    pub is_typed: bool,
    pub is_new: bool,
}

pub fn picker_items(props: &PickerProps, neighbors: &[PathBuf]) -> Vec<PickerItem> {
    let typed = discover::typed_fabrik(&props.query);
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

pub const fn clamp_selection(selected: usize, len: usize) -> usize {
    if len == 0 {
        0
    } else {
        let max = len.saturating_sub(1);
        if selected > max { max } else { selected }
    }
}

pub fn step_selection(selected: usize, len: usize, delta: i32) -> usize {
    if len == 0 {
        return 0;
    }
    let clamped = clamp_selection(selected, len);
    let current = i32::try_from(clamped).unwrap_or(0);
    let max = i32::try_from(len.saturating_sub(1)).unwrap_or(0);
    let next = current.saturating_add(delta).clamp(0, max);
    usize::try_from(next).unwrap_or(0)
}

pub fn picker_empty_state(ui: &mut Ui) {
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

pub fn render_picker_sections(
    ui: &mut Ui,
    items: &[PickerItem],
    preview: usize,
    active: Option<&Path>,
    out: &mut BodyOut,
) {
    let mut idx = 0;
    if items.get(idx).is_some_and(|item| item.is_typed) {
        let typed_end = items
            .iter()
            .position(|item| !item.is_typed)
            .unwrap_or(items.len());
        if let Some(typed) = items.get(..typed_end) {
            let count = typed.len();
            picker_list_frame().show(ui, |ui| {
                theme::fill_width(ui);
                for (offset, item) in typed.iter().enumerate() {
                    let last = offset.saturating_add(1) == count;
                    if picker_row(ui, item, offset == preview, active, offset == 0, !last) {
                        out.chosen = Some(item.path.clone());
                    }
                }
            });
        }
        idx = typed_end;
        ui.add_space(theme::SPACING);
    }
    if items.get(idx).is_some_and(|item| item.is_tab) {
        let tab_start = idx;
        while items.get(idx).is_some_and(|item| item.is_tab) {
            idx = idx.saturating_add(1);
        }
        section_label(ui, "Open tabs");
        if let Some(tabs) = items.get(tab_start..idx) {
            picker_list_frame().show(ui, |ui| {
                theme::fill_width(ui);
                for (offset, item) in tabs.iter().enumerate() {
                    let flat = tab_start.saturating_add(offset);
                    let last = offset.saturating_add(1) == tabs.len();
                    if picker_row(ui, item, flat == preview, active, offset == 0, !last) {
                        out.chosen = Some(item.path.clone());
                    }
                }
            });
        }
        ui.add_space(theme::SPACING);
    }
    if idx < items.len() {
        let rec_start = idx;
        section_label(ui, "Recents & neighbors");
        if let Some(recents) = items.get(rec_start..) {
            picker_list_frame().show(ui, |ui| {
                theme::fill_width(ui);
                for (offset, item) in recents.iter().enumerate() {
                    let flat = rec_start.saturating_add(offset);
                    let last = offset.saturating_add(1) == recents.len();
                    if picker_row(ui, item, flat == preview, active, offset == 0, !last) {
                        out.chosen = Some(item.path.clone());
                    }
                }
            });
        }
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

const fn picker_row_corners(first: bool, last: bool) -> egui::CornerRadius {
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
    const LINE_GAP: f32 = 2.0;
    const TITLE_H: f32 = 14.0;
    const SUB_H: f32 = 12.0;
    let is_active = active.is_some_and(|path| path == item.path.as_path());
    let last = !show_separator;
    let row_h = theme::ROW_HEIGHT.max(48.0);
    let width = ui.available_width();
    let (rect, response) = ui.allocate_exact_size(egui::vec2(width, row_h), egui::Sense::click());
    let hovered = response.hovered();
    let fill = picker_row_fill(is_selected, hovered);
    if fill != egui::Color32::TRANSPARENT {
        ui.painter()
            .rect_filled(rect, picker_row_corners(first, last), fill);
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
        egui::pos2(icon_w.mul_add(0.5, inner.min.x), inner.center().y),
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
    let block_h = TITLE_H + LINE_GAP + SUB_H;
    let mid = inner.center().y;
    let title_y = TITLE_H.mul_add(0.5, block_h.mul_add(-0.5, mid));
    let sub_y = SUB_H.mul_add(-0.5, block_h.mul_add(0.5, mid));
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
    use crate::ui::project_picker::PickerProps;

    use super::{clamp_selection, picker_items, step_selection};

    fn props_with(tabs: &[PathBuf], recents: &[PathBuf], query: &str) -> PickerProps {
        PickerProps {
            tabs: tabs.iter().cloned().map(recent_from_path).collect(),
            recents: recents.iter().cloned().map(recent_from_path).collect(),
            active: tabs.first().cloned(),
            query: query.to_owned(),
            selected: 0,
        }
    }

    fn fabrik_dir(tag: &str) -> PathBuf {
        let dir = std::env::temp_dir().join(format!("fabrik-picker-{tag}"));
        let _ = std::fs::create_dir_all(dir.join(".fabrik").join("dolt"));
        let _ = std::fs::write(dir.join(".fabrik").join("config.yaml"), "apps: []\n");
        dir
    }

    #[test]
    fn typed_valid_path_pins_row_zero() {
        let typed = fabrik_dir("typed-top");
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
        let tab_a = fabrik_dir("tab-a");
        let tab_b = fabrik_dir("tab-b");
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
        // Absolute query: cannot resolve to a repo by walking up, so no
        // typed row is pinned and nothing fuzzy-matches.
        let props = props_with(
            &[PathBuf::from("/work/alpha")],
            &[PathBuf::from("/work/beta")],
            "/zzz-no-such-project",
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
