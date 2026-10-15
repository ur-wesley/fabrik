use std::time::{Duration, Instant};

use eframe::egui;
use egui_commonmark::CommonMarkCache;

use crate::effects::Effects;
use crate::model::{Action, Screen};
use crate::platform;
use crate::store::MAX_TAB_SLOT;
use crate::store::Store;
use crate::ui::{
    board, chrome, detail, lookup, mem_form, memory, project_picker, settings, theme, toast,
    window_chrome,
};
use crate::views::{
    apply_route_toolbar, board_view, lookup_pane_view, memory_pane_view, route_toolbar_view,
    settings_pane_view, window_nav_screen,
};
use crate::watch::RepoWorker;

const TICK_MS: u64 = 200;
const REPAINT_MS: u64 = 250;
const GLASS_REAPPLY_EVERY_N_FRAMES: u64 = 30;

const TAB_KEYS: [egui::Key; 9] = [
    egui::Key::Num1,
    egui::Key::Num2,
    egui::Key::Num3,
    egui::Key::Num4,
    egui::Key::Num5,
    egui::Key::Num6,
    egui::Key::Num7,
    egui::Key::Num8,
    egui::Key::Num9,
];

fn tab_slot(index: usize) -> Option<u8> {
    u8::try_from(index)
        .ok()
        .map(|base| base.saturating_add(1).min(MAX_TAB_SLOT))
}

pub struct NeonApp {
    pub(crate) state: Store,
    pub(crate) worker: RepoWorker,
    pub(crate) subscribe: crate::subscribe::SubscribeService,
    pub(crate) last_tick: Instant,
    pub(crate) markdown: CommonMarkCache,
    pub(crate) last_focus: Option<bool>,
    pub(crate) last_maximized: Option<bool>,
    pub(crate) glass_applied: bool,
    pub(crate) glass_tick: u64,
    pub(crate) last_egui_theme: Option<egui::Theme>,
    pub(crate) mem_query_pending: Option<(String, Instant)>,
    pub(crate) lookup_query_pending: Option<(String, Instant)>,
}

impl NeonApp {
    pub fn new(cc: &eframe::CreationContext<'_>) -> Self {
        let mut fonts = egui::FontDefinitions::default();
        egui_phosphor::add_to_fonts(&mut fonts, egui_phosphor::Variant::Regular);
        const PHOSPHOR: &str = "phosphor";
        if let Some(keys) = fonts.families.get_mut(&egui::FontFamily::Proportional) {
            keys.retain(|family| family != PHOSPHOR);
            keys.push(PHOSPHOR.into());
        }
        cc.egui_ctx.set_fonts(fonts);

        cc.egui_ctx.set_theme(egui::ThemePreference::System);
        cc.egui_ctx.all_styles_mut(theme::apply);
        let worker = RepoWorker::new();
        let subscribe = crate::subscribe::SubscribeService::new();
        let state = Store::new();
        let mut app = Self {
            state,
            worker,
            subscribe,
            last_tick: Instant::now(),
            markdown: CommonMarkCache::default(),
            last_focus: None,
            last_maximized: None,
            glass_applied: false,
            glass_tick: 0,
            last_egui_theme: None,
            mem_query_pending: None,
            lookup_query_pending: None,
        };
        if let Some(arg) = std::env::args().skip(1).find(|a| !a.starts_with('-')) {
            app.effects()
                .apply(vec![Action::RepoPicked(std::path::PathBuf::from(arg))]);
        } else if let Some(repo) = app.state.workspace.repo.clone() {
            app.worker.set_repo(&repo);
            app.subscribe.set_repo(Some(&repo));
        }
        app.worker
            .refresh_memory(app.state.workspace.repo.clone().as_deref());
        app
    }

    pub(crate) fn effects(&mut self) -> Effects<'_> {
        Effects {
            state: &mut self.state,
            worker: &mut self.worker,
            subscribe: &mut self.subscribe,
            mem_query_pending: &mut self.mem_query_pending,
            lookup_query_pending: &mut self.lookup_query_pending,
        }
    }

    fn handle_keyboard_ctx(&self, ctx: &egui::Context) -> Vec<Action> {
        let mut actions = Vec::new();
        ctx.input_mut(|input| {
            if input.consume_key(
                egui::Modifiers::COMMAND.plus(egui::Modifiers::SHIFT),
                egui::Key::Tab,
            ) {
                actions.push(Action::PrevTab);
            } else if input.consume_key(egui::Modifiers::COMMAND, egui::Key::Tab) {
                actions.push(Action::NextTab);
            }
            if input.consume_key(egui::Modifiers::COMMAND, egui::Key::PageDown) {
                actions.push(Action::NextTab);
            }
            if input.consume_key(egui::Modifiers::COMMAND, egui::Key::PageUp) {
                actions.push(Action::PrevTab);
            }
            if input.consume_key(egui::Modifiers::COMMAND, egui::Key::P) {
                actions.push(Action::TogglePicker);
            }
            if input.consume_key(
                egui::Modifiers::COMMAND.plus(egui::Modifiers::SHIFT),
                egui::Key::T,
            ) {
                actions.push(Action::ReopenTab);
            }
            if input.consume_key(egui::Modifiers::COMMAND, egui::Key::W)
                && let Some(path) = self.state.workspace.repo.clone()
            {
                actions.push(Action::CloseTab(path));
            }
            for (index, key) in TAB_KEYS.iter().enumerate() {
                if input.consume_key(egui::Modifiers::COMMAND, *key)
                    && let Some(slot) = tab_slot(index)
                {
                    actions.push(Action::SelectTab(slot));
                }
            }
        });
        ctx.input(|input| {
            if input.key_pressed(egui::Key::Escape) {
                if self.state.workspace.picker_open {
                    actions.push(Action::ClosePicker);
                } else if self.state.memory.mem_delete_confirm.is_some() {
                    actions.push(Action::CancelMemDelete);
                } else if self.state.memory.mem_form_open {
                    actions.push(Action::CloseMemForm);
                } else if self.state.memory.mem_prompt_open {
                    actions.push(Action::CloseMemPrompt);
                } else if self.state.workspace.screen == Screen::Detail {
                    actions.push(Action::Back);
                } else if self.state.memory.mem_detail_id.is_some() {
                    actions.push(Action::MemClearDetail);
                } else if self.state.workspace.screen == Screen::Memory {
                    if !self.state.memory.mem_search.is_empty() {
                        actions.push(Action::ClearPaneSearch);
                    }
                } else if self.state.workspace.screen == Screen::Lookup {
                    if self.state.lookup.lookup_path.is_some()
                        || self.state.lookup.lookup_explain.is_some()
                    {
                        actions.push(Action::LookupClearPath);
                    } else if !self.state.lookup.lookup_query.is_empty() {
                        actions.push(Action::ClearPaneSearch);
                    }
                } else if self.state.workspace.screen == Screen::Settings {
                    actions.push(Action::SetScreen(Screen::Board));
                } else {
                    actions.push(Action::ClearPaneSearch);
                }
            }
        });
        if self.state.workspace.picker_open {
            return actions;
        }
        if self.state.workspace.screen == Screen::Detail {
            return actions;
        }
        ctx.input(|input| {
            if input.key_pressed(egui::Key::Slash) {
                match self.state.workspace.screen {
                    Screen::Memory => actions.push(Action::FocusMemSearch),
                    Screen::Lookup => actions.push(Action::FocusLookupSearch),
                    Screen::Board => actions.push(Action::FocusSearch),
                    Screen::Detail | Screen::Settings => {}
                }
            }
        });
        if self.state.workspace.screen != Screen::Board {
            return actions;
        }
        ctx.input(|input| {
            if input.key_pressed(egui::Key::ArrowDown) {
                actions.push(Action::MoveSelection(1));
            }
            if input.key_pressed(egui::Key::ArrowUp) {
                actions.push(Action::MoveSelection(-1));
            }
            if input.key_pressed(egui::Key::Enter) {
                actions.push(Action::OpenSelected);
            }
        });
        actions
    }

    fn show_popup(&mut self, ctx: &egui::Context) {
        theme::sync_paint_from_ctx(ctx);
        if self.state.workspace.picker_open {
            let props = project_picker::PickerProps {
                tabs: self.state.tab_entries(),
                recents: self.state.recent_entries(),
                active: self.state.workspace.repo.clone(),
                query: self.state.workspace.picker_query.clone(),
                selected: self.state.workspace.picker_selected,
            };
            let actions = project_picker::project_picker(ctx, &props);
            self.effects().apply(actions);
        }
        if self.state.workspace.screen != Screen::Detail {
            self.show_create_popup(ctx);
            self.show_memory_popups(ctx);
            return;
        }
        let view = detail::DetailView {
            detail: self.state.board.detail.clone(),
            loading_id: self.state.board.detail_id.clone(),
        };
        let composer = detail::ComposerState {
            draft: &mut self.state.board.comment_draft,
            posting: self.state.board.comment_posting,
            error: self.state.board.comment_error.clone(),
        };
        let mut ops = detail::OpsForm {
            active: view.detail.as_ref().is_some_and(|d| d.status == "active"),
            by: &mut self.state.board.op_by,
            close_reason: &mut self.state.board.close_reason,
            force: &mut self.state.board.write_force,
            dep_target: &mut self.state.board.dep_target,
            dep_kind: &mut self.state.board.dep_kind,
            busy: self.state.board.write_busy,
            error: self.state.board.write_error.clone(),
        };
        let frame = theme::dialog_frame(ctx.global_style().as_ref());
        let modal = egui::Modal::new(egui::Id::new("issue_popup"))
            .frame(frame)
            .backdrop_color(theme::dialog_backdrop())
            .show(ctx, |ui| {
                detail::popup(ui, &view, &mut self.markdown, composer, &mut ops)
            });
        let should_close = modal.should_close();
        let mut actions = Vec::new();
        if let Some(action) = modal.inner {
            actions.push(action);
        }
        if should_close {
            actions.push(Action::Back);
        }
        self.effects().apply(actions);
        self.show_create_popup(ctx);
        self.show_memory_popups(ctx);
    }

    fn show_create_popup(&mut self, ctx: &egui::Context) {
        if !self.state.board.create_open {
            return;
        }
        let mut form = crate::ui::board::CreateForm {
            title: &mut self.state.board.create_title,
            body: &mut self.state.board.create_body,
            acceptance: &mut self.state.board.create_acceptance,
            priority: &mut self.state.board.create_priority,
            kind: &mut self.state.board.create_kind,
            busy: self.state.board.write_busy,
            error: self.state.board.write_error.clone(),
        };
        let frame = theme::dialog_frame(ctx.global_style().as_ref());
        let modal = egui::Modal::new(egui::Id::new("create_issue"))
            .frame(frame)
            .backdrop_color(theme::dialog_backdrop())
            .show(ctx, |ui| crate::ui::board::create_dialog(ui, &mut form));
        let should_close = modal.should_close();
        let mut actions = modal.inner;
        if should_close {
            actions.push(Action::CloseCreate);
        }
        self.effects().apply(actions);
    }
}

impl eframe::App for NeonApp {
    fn clear_color(&self, _visuals: &egui::Visuals) -> [f32; 4] {
        egui::Rgba::TRANSPARENT.to_array()
    }

    fn ui(&mut self, ui: &mut egui::Ui, frame: &mut eframe::Frame) {
        let ctx = ui.ctx().clone();
        let egui_theme = ctx.theme();
        if self.last_egui_theme != Some(egui_theme) {
            ctx.all_styles_mut(theme::apply);
            self.last_egui_theme = Some(egui_theme);
        }
        theme::sync_paint_from_ui(ui);
        let focused = ctx.input(|i| i.viewport().focused) != Some(false);
        let maximized = ctx.input(|i| i.viewport().maximized).unwrap_or(false);
        let focus_changed = self.last_focus != Some(focused);
        let maximized_changed = self.last_maximized != Some(maximized);
        let periodic = self.glass_tick.is_multiple_of(GLASS_REAPPLY_EVERY_N_FRAMES);
        if (!self.glass_applied || focus_changed || maximized_changed || periodic)
            && platform::refresh_glass(Some(frame), focused)
        {
            self.glass_applied = true;
            self.last_focus = Some(focused);
            self.last_maximized = Some(maximized);
        }
        self.glass_tick = self.glass_tick.wrapping_add(1);

        if self.last_tick.elapsed() >= Duration::from_millis(TICK_MS) {
            self.last_tick = Instant::now();
            self.effects().apply(vec![Action::Tick]);
        }

        let worker_actions = self.worker.poll(&ctx);
        let subscribe_actions = self.subscribe.poll();
        self.effects().apply(worker_actions);
        self.effects().apply(subscribe_actions);

        let keyboard_actions = self.handle_keyboard_ctx(&ctx);
        self.effects().apply(keyboard_actions);

        let mut actions = Vec::new();

        egui::CentralPanel::default()
            .frame(egui::Frame::NONE)
            .show(ui, |ui| {
                let nav_screen = window_nav_screen(&self.state);
                let screen = self.state.workspace.screen;
                let mut route_toolbar = route_toolbar_view(&self.state);
                let (title_actions, body_actions) =
                    window_chrome::shell(ui, |title_ui, body_ui| {
                        let title_actions = chrome::window_nav(title_ui, nav_screen);

                        let body_actions = match screen {
                            Screen::Memory => {
                                let view = memory_pane_view(&self.state);
                                let mut form = mem_form::MemForm::from_store(&self.state.memory);
                                let mut actions = memory::memory_pane(
                                    body_ui,
                                    &view,
                                    &mut route_toolbar,
                                    &mut self.markdown,
                                    &mut form,
                                );
                                form.flush(&mut actions, &self.state.memory);
                                actions
                            }
                            Screen::Lookup => {
                                let view = lookup_pane_view(&self.state);
                                lookup::lookup_pane(body_ui, &view, &mut route_toolbar)
                            }
                            Screen::Settings => {
                                let view = settings_pane_view(&self.state);
                                settings::settings_pane(body_ui, &view)
                            }
                            Screen::Board | Screen::Detail => {
                                let view = board_view(&self.state);
                                board::board(body_ui, &view, &mut route_toolbar)
                            }
                        };
                        (title_actions, body_actions)
                    });
                if screen != Screen::Settings {
                    apply_route_toolbar(&mut self.state, &route_toolbar);
                }
                actions.extend(title_actions);
                actions.extend(body_actions);
            });

        self.show_popup(&ctx);
        self.effects().apply(actions);

        toast::toast_overlay(&ctx, &mut self.state.toasts);

        ctx.request_repaint_after(Duration::from_millis(REPAINT_MS));
    }
}

#[cfg(test)]
mod tests {
    use super::tab_slot;

    #[test]
    fn tab_slot_is_one_based_with_num9_clamped_to_8() {
        assert_eq!(tab_slot(0), Some(1));
        assert_eq!(tab_slot(1), Some(2));
        assert_eq!(tab_slot(6), Some(7));
        assert_eq!(tab_slot(7), Some(8));
        assert_eq!(tab_slot(8), Some(8));
    }
}
