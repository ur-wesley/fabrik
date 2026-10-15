use eframe::egui::{self, Ui};

use crate::model::Action;

use super::button;
use super::memory::MemoryView;
use super::theme;

pub struct MemForm {
    pub title: String,
    pub body: String,
    pub kind: String,
    pub topic: String,
    pub symbol: String,
    pub prompt_title: String,
    pub prompt_capture: String,
}

impl MemForm {
    pub fn from_store(store: &crate::memory::MemoryStore) -> Self {
        Self {
            title: store.mem_form_title.clone(),
            body: store.mem_form_body.clone(),
            kind: store.mem_form_kind.clone(),
            topic: store.mem_form_topic.clone(),
            symbol: store.mem_form_symbol.clone(),
            prompt_title: store.mem_prompt_title.clone(),
            prompt_capture: store.mem_prompt_capture.clone(),
        }
    }

    pub fn flush(&self, actions: &mut Vec<Action>, store: &crate::memory::MemoryStore) {
        // Owned-form write-back (mirrors the route toolbar): the pane
        // edits its copy, diffs become setter actions.
        if self.title != store.mem_form_title {
            actions.push(Action::SetMemTitle(self.title.clone()));
        }
        if self.body != store.mem_form_body {
            actions.push(Action::SetMemBody(self.body.clone()));
        }
        if self.kind != store.mem_form_kind {
            actions.push(Action::SetMemKind(self.kind.clone()));
        }
        if self.topic != store.mem_form_topic {
            actions.push(Action::SetMemTopicKey(self.topic.clone()));
        }
        if self.symbol != store.mem_form_symbol {
            actions.push(Action::SetMemSymbolId(self.symbol.clone()));
        }
        if self.prompt_title != store.mem_prompt_title {
            actions.push(Action::SetMemPromptTitle(self.prompt_title.clone()));
        }
        if self.prompt_capture != store.mem_prompt_capture {
            actions.push(Action::SetMemPromptCapture(self.prompt_capture.clone()));
        }
    }
}

pub fn can_submit_mem(title: &str, body: &str, busy: bool) -> bool {
    !busy && !title.trim().is_empty() && !body.trim().is_empty()
}

pub fn write_row(ui: &mut Ui, _view: &MemoryView<'_>) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = theme::SPACING;
        if button::button(
            ui,
            &button::ButtonProps::new("+ New")
                .variant(button::ButtonVariant::Secondary)
                .min_size(egui::vec2(84.0, theme::CONTROL_H)),
        )
        .clicked()
        {
            actions.push(Action::OpenMemForm);
        }
        if button::button(
            ui,
            &button::ButtonProps::new("Save prompt")
                .variant(button::ButtonVariant::Outline)
                .min_size(egui::vec2(96.0, theme::CONTROL_H)),
        )
        .clicked()
        {
            actions.push(Action::OpenMemPrompt);
        }
    });
    actions
}

#[allow(clippy::too_many_lines)]
pub fn mem_form_card(ui: &mut Ui, view: &MemoryView<'_>, form: &mut MemForm) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.set_min_width(560.0);
    ui.set_max_width(600.0);
    let heading = if view.editing.is_some() {
        "Edit observation"
    } else {
        "New observation"
    };
    ui.label(
        egui::RichText::new(heading)
            .size(18.0)
            .strong()
            .color(theme::text_primary()),
    );
    ui.add_space(theme::SPACING);
    let title_resp = ui.add(
        egui::TextEdit::singleline(&mut form.title)
            .hint_text("Title (required)")
            .desired_width(f32::INFINITY),
    );
    let _ = title_resp;
    ui.add_space(theme::SPACING);
    let body_resp = ui.add(
        egui::TextEdit::multiline(&mut form.body)
            .hint_text("Body — `fabrik mem save --body …`")
            .desired_rows(3)
            .desired_width(f32::INFINITY),
    );
    let _ = body_resp;
    ui.add_space(theme::SPACING);
    ui.horizontal(|ui| {
        ui.label(egui::RichText::new("kind").small().color(theme::dim_text()));
        for kind in crate::fabrik::MEM_KINDS {
            ui.selectable_value(&mut form.kind, kind.to_owned(), kind);
        }
    });
    ui.add_space(4.0);
    ui.add(
        egui::TextEdit::singleline(&mut form.topic)
            .hint_text("topic key (optional) — `--topic-key …`")
            .desired_width(f32::INFINITY),
    );
    ui.add_space(4.0);
    ui.add(
        egui::TextEdit::singleline(&mut form.symbol)
            .hint_text("symbol id (optional) — `--symbol-id sym:…`")
            .desired_width(f32::INFINITY),
    );
    ui.add_space(theme::SPACING);
    if let Some(message) = view.write_error.clone() {
        ui.label(egui::RichText::new(message).small().color(theme::sys_red()));
        ui.add_space(theme::SPACING);
    }
    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = theme::SPACING;
        if view.write_busy {
            ui.add(egui::Spinner::new());
            ui.label(
                egui::RichText::new("Saving…")
                    .small()
                    .italics()
                    .color(theme::dim_text()),
            );
        } else {
            if button::button(
                ui,
                &button::ButtonProps::new("Cancel")
                    .variant(button::ButtonVariant::Outline)
                    .min_size(egui::vec2(84.0, theme::CONTROL_H)),
            )
            .clicked()
            {
                actions.push(Action::CloseMemForm);
            }
            let ready = can_submit_mem(&form.title, &form.body, false);
            let label = if view.editing.is_some() {
                "Update"
            } else {
                "Save"
            };
            if ui
                .add_enabled(
                    ready,
                    egui::Button::new(
                        egui::RichText::new(label)
                            .small()
                            .strong()
                            .color(theme::accent()),
                    )
                    .fill(theme::accent_secondary_fill())
                    .stroke(egui::Stroke::NONE)
                    .corner_radius(theme::radius())
                    .min_size(egui::vec2(84.0, theme::CONTROL_H)),
                )
                .clicked()
            {
                actions.push(if view.editing.is_some() {
                    Action::SubmitMemUpdate
                } else {
                    Action::SubmitMemSave
                });
            }
        }
    });
    actions
}

pub fn mem_prompt_card(ui: &mut Ui, view: &MemoryView<'_>, form: &mut MemForm) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.set_min_width(560.0);
    ui.set_max_width(600.0);
    ui.label(
        egui::RichText::new("Save prompt")
            .size(18.0)
            .strong()
            .color(theme::text_primary()),
    );
    ui.label(
        egui::RichText::new("`fabrik mem save-prompt --capture-prompt …`")
            .font(theme::font_caption2())
            .color(theme::text_tertiary()),
    );
    ui.add_space(theme::SPACING);
    ui.add(
        egui::TextEdit::singleline(&mut form.prompt_title)
            .hint_text("Prompt title")
            .desired_width(f32::INFINITY),
    );
    ui.add_space(theme::SPACING);
    ui.add(
        egui::TextEdit::multiline(&mut form.prompt_capture)
            .hint_text("Prompt text (required)")
            .desired_rows(3)
            .desired_width(f32::INFINITY),
    );
    ui.add_space(theme::SPACING);
    if let Some(message) = view.write_error.clone() {
        ui.label(egui::RichText::new(message).small().color(theme::sys_red()));
        ui.add_space(theme::SPACING);
    }
    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = theme::SPACING;
        if view.write_busy {
            ui.add(egui::Spinner::new());
            ui.label(
                egui::RichText::new("Saving…")
                    .small()
                    .italics()
                    .color(theme::dim_text()),
            );
        } else {
            if button::button(
                ui,
                &button::ButtonProps::new("Cancel")
                    .variant(button::ButtonVariant::Outline)
                    .min_size(egui::vec2(84.0, theme::CONTROL_H)),
            )
            .clicked()
            {
                actions.push(Action::CloseMemPrompt);
            }
            let ready = !form.prompt_capture.trim().is_empty();
            if ui
                .add_enabled(
                    ready,
                    egui::Button::new(
                        egui::RichText::new("Save prompt")
                            .small()
                            .strong()
                            .color(theme::accent()),
                    )
                    .fill(theme::accent_secondary_fill())
                    .stroke(egui::Stroke::NONE)
                    .corner_radius(theme::radius())
                    .min_size(egui::vec2(84.0, theme::CONTROL_H)),
                )
                .clicked()
            {
                actions.push(Action::SubmitMemPrompt);
            }
        }
    });
    actions
}

pub fn delete_confirm_card(ui: &mut Ui, id: &str, busy: bool) -> Vec<Action> {
    let mut actions = Vec::new();
    ui.set_min_width(560.0);
    ui.set_max_width(600.0);
    ui.label(
        egui::RichText::new(format!("Delete {id}?"))
            .size(18.0)
            .strong()
            .color(theme::text_primary()),
    );
    ui.label(
        egui::RichText::new("Soft-deletes the observation (`fabrik mem delete --id …`).")
            .font(theme::font_footnote())
            .color(theme::text_secondary()),
    );
    ui.add_space(theme::SPACING);
    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = theme::SPACING;
        if busy {
            ui.add(egui::Spinner::new());
            ui.label(
                egui::RichText::new("Deleting…")
                    .small()
                    .italics()
                    .color(theme::dim_text()),
            );
        } else {
            if button::button(
                ui,
                &button::ButtonProps::new("Cancel")
                    .variant(button::ButtonVariant::Outline)
                    .min_size(egui::vec2(84.0, theme::CONTROL_H)),
            )
            .clicked()
            {
                actions.push(Action::CancelMemDelete);
            }
            if button::button(
                ui,
                &button::ButtonProps::new("Delete")
                    .variant(button::ButtonVariant::Secondary)
                    .min_size(egui::vec2(84.0, theme::CONTROL_H)),
            )
            .clicked()
            {
                actions.push(Action::ConfirmMemDelete);
            }
        }
    });
    actions
}

#[cfg(test)]
mod tests {
    use super::can_submit_mem;

    #[test]
    fn mem_submit_gate() {
        assert!(can_submit_mem("t", "b", false));
        assert!(!can_submit_mem("  ", "b", false));
        assert!(!can_submit_mem("t", "", false));
        assert!(!can_submit_mem("t", "b", true));
    }
}
