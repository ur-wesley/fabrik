use eframe::egui;

use crate::model::Action;
use crate::ui::{mem_form, memory, theme};
use crate::views::memory_pane_view;

use super::app::NeonApp;

#[allow(clippy::missing_const_for_fn)]
fn empty_observation() -> crate::fabrik::Observation {
    crate::fabrik::Observation {
        id: String::new(),
        session_id: None,
        project: String::new(),
        kind: String::new(),
        title: String::new(),
        body: String::new(),
        topic_key: None,
        symbol_id: None,
        created_at: String::new(),
    }
}

impl NeonApp {
    #[allow(clippy::too_many_lines)]
    pub(crate) fn show_memory_popups(&mut self, ctx: &egui::Context) {
        if self.state.memory.mem_form_open {
            let view = memory_pane_view(&self.state);
            let mut form = mem_form::MemForm::from_store(&self.state.memory);
            let frame = theme::dialog_frame(ctx.global_style().as_ref());
            let modal = egui::Modal::new(egui::Id::new("mem_form"))
                .frame(frame)
                .backdrop_color(theme::dialog_backdrop())
                .show(ctx, |ui| mem_form::mem_form_card(ui, &view, &mut form));
            let should_close = modal.should_close();
            let mut actions = modal.inner;
            form.flush(&mut actions, &self.state.memory);
            if should_close {
                actions.push(Action::CloseMemForm);
            }
            self.effects().apply(actions);
        }
        if self.state.memory.mem_prompt_open {
            let view = memory_pane_view(&self.state);
            let mut form = mem_form::MemForm::from_store(&self.state.memory);
            let frame = theme::dialog_frame(ctx.global_style().as_ref());
            let modal = egui::Modal::new(egui::Id::new("mem_prompt"))
                .frame(frame)
                .backdrop_color(theme::dialog_backdrop())
                .show(ctx, |ui| mem_form::mem_prompt_card(ui, &view, &mut form));
            let should_close = modal.should_close();
            let mut actions = modal.inner;
            form.flush(&mut actions, &self.state.memory);
            if should_close {
                actions.push(Action::CloseMemPrompt);
            }
            self.effects().apply(actions);
        }
        if let Some(id) = self.state.memory.mem_delete_confirm.clone() {
            let busy = self.state.memory.mem_write_busy;
            let frame = theme::dialog_frame(ctx.global_style().as_ref());
            let modal = egui::Modal::new(egui::Id::new("mem_delete"))
                .frame(frame)
                .backdrop_color(theme::dialog_backdrop())
                .show(ctx, |ui| mem_form::delete_confirm_card(ui, &id, busy));
            let should_close = modal.should_close();
            let mut actions = modal.inner;
            if should_close {
                actions.push(Action::CancelMemDelete);
            }
            self.effects().apply(actions);
        }
        if self.state.memory.mem_detail_id.is_some() {
            let frame = theme::dialog_frame(ctx.global_style().as_ref());
            if self.state.memory.mem_detail.is_some() {
                let detail = self
                    .state
                    .memory
                    .mem_detail
                    .clone()
                    .unwrap_or_else(empty_observation);
                let timeline = self.state.memory.mem_timeline.clone();
                let timeline_id = self.state.memory.mem_timeline_id.clone();
                let delete_confirm = self.state.memory.mem_delete_confirm.clone();
                let write_busy = self.state.memory.mem_write_busy;
                let modal = egui::Modal::new(egui::Id::new("memory_detail"))
                    .frame(frame)
                    .backdrop_color(theme::dialog_backdrop())
                    .show(ctx, |ui| {
                        memory::detail_sheet(
                            ui,
                            &detail,
                            &mut self.markdown,
                            timeline.as_ref(),
                            timeline_id.as_deref(),
                            delete_confirm.as_deref(),
                            write_busy,
                        )
                    });
                let should_close = modal.should_close();
                let mut actions = modal.inner;
                if should_close {
                    actions.push(Action::MemClearDetail);
                }
                self.effects().apply(actions);
            } else {
                let loading_id = self.state.memory.mem_detail_id.clone().unwrap_or_default();
                let modal = egui::Modal::new(egui::Id::new("memory_detail"))
                    .frame(frame)
                    .backdrop_color(theme::dialog_backdrop())
                    .show(ctx, |ui| {
                        ui.set_min_width(720.0);
                        ui.set_min_height(520.0);
                        theme::paint_dialog_top_highlight(ui);
                        ui.vertical_centered(|ui| {
                            ui.add_space(40.0);
                            ui.add(egui::Spinner::new());
                            ui.add_space(theme::SPACING);
                            ui.label(
                                egui::RichText::new(format!("loading {loading_id}…"))
                                    .italics()
                                    .color(theme::dim_text()),
                            );
                        });
                        Vec::<Action>::new()
                    });
                if modal.should_close() {
                    self.effects().apply(vec![Action::MemClearDetail]);
                } else {
                    self.effects().apply(modal.inner);
                }
            }
        }
    }
}
