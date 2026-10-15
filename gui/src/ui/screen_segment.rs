use eframe::egui::Ui;

use crate::model::Screen;

use super::segment;

const SCREEN_W: [f32; 4] = [64.0, 72.0, 72.0, 76.0];

pub fn screen_segment(ui: &mut Ui, screen: Screen) -> Option<Screen> {
    let labels = ["Board", "Memory", "Lookup", "Settings"];
    let selected = match screen {
        Screen::Board | Screen::Detail => 0,
        Screen::Memory => 1,
        Screen::Lookup => 2,
        Screen::Settings => 3,
    };
    let picked = segment::segmented_control(
        ui,
        ui.id().with("screen_tabs"),
        &labels,
        &SCREEN_W,
        selected,
    )?;
    let next = match picked {
        0 => Screen::Board,
        1 => Screen::Memory,
        2 => Screen::Lookup,
        3 => Screen::Settings,
        _ => return None,
    };
    if next == screen || (screen == Screen::Detail && next == Screen::Board) {
        return None;
    }
    Some(next)
}
