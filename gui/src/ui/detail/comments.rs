use eframe::egui::{self, Ui};
use egui_commonmark::{CommonMarkCache, CommonMarkViewer};

use crate::model::{Action, Comment, IssueDetail, format_local};
use crate::toast::error_line;

use super::super::{badge, icons, theme};
use super::{ComposerState, can_post_comment};

pub(super) fn comments_section(
    ui: &mut Ui,
    detail: &IssueDetail,
    cache: &mut CommonMarkCache,
    composer: ComposerState<'_>,
) -> Option<Action> {
    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing.x = theme::SPACING;
        ui.label(
            egui::RichText::new(icons::FILE_TEXT)
                .small()
                .color(theme::dim_text()),
        );
        ui.label(
            egui::RichText::new("COMMENTS")
                .small()
                .strong()
                .color(theme::dim_text()),
        );
        let count = detail.comments.len().max(detail.comment_count);
        badge::badge(ui, &count.to_string());
    });
    ui.add_space(theme::SPACING);

    let _ = theme::muted_frame()
        .inner_margin(egui::Margin::same(theme::PAD_I8))
        .show(ui, |ui| {
            theme::fill_width(ui);
            if detail.comments.is_empty() {
                ui.label(
                    egui::RichText::new("No comments yet.")
                        .small()
                        .italics()
                        .color(theme::dim_text()),
                );
            } else {
                for comment in &detail.comments {
                    comment_row(ui, cache, comment);
                    ui.add_space(theme::SPACING);
                }
            }
        });

    ui.add_space(theme::SPACING);
    composer_card(ui, composer)
}

fn comment_row(ui: &mut Ui, cache: &mut CommonMarkCache, comment: &Comment) {
    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing = egui::vec2(theme::SPACING, 0.0);
        badge::badge_color(
            ui,
            &format!(
                "{} {}",
                icons::USER,
                comment.author.as_deref().unwrap_or("unknown")
            ),
            theme::neon_cyan(),
        );
        if let Some(created) = comment.created_at {
            ui.label(
                egui::RichText::new(format_local(created))
                    .small()
                    .color(theme::dim_text()),
            );
        }
    });
    ui.add_space(2.0);
    if comment.body.trim().is_empty() {
        ui.label(
            egui::RichText::new("(empty comment)")
                .small()
                .italics()
                .color(theme::dim_text()),
        );
    } else {
        CommonMarkViewer::new().show(ui, cache, &comment.body);
    }
}

fn composer_card(ui: &mut Ui, composer: ComposerState<'_>) -> Option<Action> {
    let ComposerState {
        draft,
        posting,
        error,
    } = composer;
    let mut action = None;

    if let Some(message) = &error {
        let line = error_line(message, 72);
        let label = egui::Label::new(
            egui::RichText::new(line)
                .small()
                .color(theme::sys_red()),
        )
        .truncate();
        let resp = ui.add(label);
        resp.on_hover_text(message);
        ui.add_space(theme::SPACING);
    }

    let composer_id = egui::Id::new("comment_composer");
    let ctrl_enter = ui.ctx().memory(|mem| mem.has_focus(composer_id))
        && ui
            .ctx()
            .input(|input| input.modifiers.ctrl && input.key_pressed(egui::Key::Enter));
    let can_post = can_post_comment(draft, posting);

    let _ = theme::muted_frame()
        .inner_margin(egui::Margin::same(theme::PAD_I8))
        .show(ui, |ui| {
            theme::fill_width(ui);
            let response = ui.add(
                egui::TextEdit::multiline(draft)
                    .id(composer_id)
                    .hint_text("Write a comment… (Ctrl+Enter to post)")
                    .desired_rows(3),
            );
            if response.has_focus() {
                response.scroll_to_me(None);
            }
            ui.add_space(theme::SPACING);
            ui.horizontal(|ui| {
                if posting {
                    ui.label(
                        egui::RichText::new("Posting…")
                            .small()
                            .italics()
                            .color(theme::dim_text()),
                    );
                } else {
                    let post = ui.add_enabled(
                        can_post,
                        egui::Button::new(egui::RichText::new("Post comment").small().strong()),
                    );
                    if post.clicked() {
                        action = Some(Action::PostComment);
                    }
                    ui.label(
                        egui::RichText::new("Ctrl+Enter to post")
                            .small()
                            .color(theme::dim_text()),
                    );
                }
            });
        });

    if ctrl_enter && can_post {
        action = Some(Action::PostComment);
    }
    action
}
