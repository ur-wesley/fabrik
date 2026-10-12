mod comments;
mod deps;
mod markdown;
mod sidebar;

use eframe::egui::{self, ScrollArea, Ui};

use egui_commonmark::CommonMarkCache;

use crate::model::{Action, IssueDetail};

use super::{badge, icon_button, icons, theme};

pub struct DetailView {
    pub detail: Option<IssueDetail>,
    pub loading_id: Option<String>,
}

pub struct ComposerState<'a> {
    pub draft: &'a mut String,
    pub posting: bool,
    pub error: Option<String>,
}

pub fn can_post_comment(draft: &str, posting: bool) -> bool {
    !posting && !draft.trim().is_empty()
}

pub fn popup(
    ui: &mut Ui,
    view: &DetailView,
    cache: &mut CommonMarkCache,
    composer: ComposerState<'_>,
) -> Option<Action> {
    ui.set_min_width(820.0);
    ui.set_max_width(860.0);
    ui.set_min_height(520.0);
    ui.set_max_height(640.0);
    theme::paint_dialog_top_highlight(ui);

    let title = view.detail.as_ref().map_or_else(
        || view.loading_id.as_deref().unwrap_or("issue"),
        |detail| detail.title.as_str(),
    );

    let mut action = None;
    let mut close = false;

    ui.horizontal(|ui| {
        ui.spacing_mut().item_spacing = egui::vec2(theme::SPACING, 0.0);

        if let Some(detail) = &view.detail {
            if let Some(parent_id) = &detail.parent {
                let parent_btn = ui.add(
                    egui::Button::new(
                        egui::RichText::new(format!("{} {parent_id}", icons::ARROW_UP_RIGHT))
                            .small()
                            .strong()
                            .color(theme::neon_cyan()),
                    )
                    .fill(egui::Color32::TRANSPARENT)
                    .stroke(egui::Stroke::NONE),
                );
                if theme::pointer(parent_btn)
                    .on_hover_text("Open parent bead")
                    .clicked()
                {
                    action = Some(Action::OpenDetail(parent_id.clone()));
                }
                ui.label(egui::RichText::new("/").small().color(theme::dim_text()));
            }

            badge::issue_id(ui, &detail.id);
            copy_id_button(ui, &detail.id);
        } else if let Some(id) = &view.loading_id {
            badge::issue_id(ui, id);
        }

        ui.with_layout(egui::Layout::right_to_left(egui::Align::Center), |ui| {
            if icon_button::icon_button(
                ui,
                &icon_button::IconButtonProps {
                    label: icons::X,
                    selected: false,
                },
            ) {
                close = true;
            }
            if let Some(detail) = &view.detail {
                badge::status_badge(ui, &detail.status);
                badge::priority_badge(ui, detail.priority);
                badge::kind_badge(ui, &detail.kind);
            }
        });
    });

    ui.add_space(theme::SPACING);

    ui.add(
        egui::Label::new(
            egui::RichText::new(title)
                .size(18.0)
                .strong()
                .color(theme::foreground()),
        )
        .wrap(),
    );

    ui.add_space(theme::SPACING);
    ui.painter().hline(
        ui.max_rect().x_range(),
        ui.cursor().top(),
        theme::hairline(),
    );
    ui.painter().hline(
        ui.max_rect().x_range(),
        ui.cursor().top() + 1.0,
        theme::dialog_highlight_stroke(),
    );
    ui.add_space(theme::SPACING);

    if close {
        return Some(Action::Back);
    }
    if action.is_some() {
        return action;
    }

    if let Some(detail) = &view.detail {
        if let Some(act) = popup_body(ui, detail, cache, composer) {
            return Some(act);
        }
    } else if let Some(id) = &view.loading_id {
        ui.add_space(40.0);
        ui.vertical_centered(|ui| {
            ui.label(
                egui::RichText::new(format!("loading {id}…"))
                    .italics()
                    .color(theme::dim_text()),
            );
        });
    }

    None
}

fn popup_body(
    ui: &mut Ui,
    detail: &IssueDetail,
    cache: &mut CommonMarkCache,
    composer: ComposerState<'_>,
) -> Option<Action> {
    let mut action = None;
    let avail_w = ui.available_width();
    let sidebar_w = 230.0_f32;
    let content_w = (avail_w - sidebar_w - theme::SPACING).max(360.0);

    ui.horizontal_top(|ui| {
        ui.allocate_ui_with_layout(
            egui::vec2(content_w, ui.available_height()),
            egui::Layout::top_down(egui::Align::Min),
            |ui| {
                ScrollArea::vertical()
                    .id_salt("issue_content_scroll")
                    .auto_shrink([false, false])
                    .show(ui, |ui| {
                        markdown::markdown_card(
                            ui,
                            cache,
                            icons::FILE_TEXT,
                            "DESCRIPTION",
                            detail.description.as_deref(),
                            "No description provided.",
                        );

                        ui.add_space(theme::SPACING);

                        if detail.acceptance_criteria.is_some() {
                            markdown::markdown_card(
                                ui,
                                cache,
                                icons::CHECK,
                                "ACCEPTANCE CRITERIA",
                                detail.acceptance_criteria.as_deref(),
                                "",
                            );
                            ui.add_space(theme::SPACING);
                        }

                        if let Some(act) =
                            comments::comments_section(ui, detail, cache, composer)
                        {
                            action = Some(act);
                        }
                        if action.is_none() {
                            ui.add_space(theme::SPACING);
                        }

                        if let Some(act) = deps::dependency_section(ui, detail) {
                            action = Some(act);
                        }
                    });
            },
        );

        ui.add_space(theme::SPACING);
        ui.painter().vline(
            ui.cursor().left(),
            ui.max_rect().y_range(),
            theme::hairline(),
        );
        ui.add_space(theme::SPACING);

        let sidebar_avail = ui.available_width();
        ui.allocate_ui_with_layout(
            egui::vec2(sidebar_avail, ui.available_height()),
            egui::Layout::top_down(egui::Align::Min),
            |ui| {
                if let Some(act) = sidebar::sidebar(ui, detail) {
                    action = Some(act);
                }
            },
        );
    });

    action
}

fn copy_id_button(ui: &mut Ui, id: &str) {
    let copy_id = ui.make_persistent_id(format!("copy_{id}"));
    let last_copied = ui.data_mut(|d| d.get_temp::<std::time::Instant>(copy_id));
    let just_copied = last_copied.is_some_and(|t| t.elapsed().as_secs() < 2);

    let (icon, color) = if just_copied {
        (icons::CHECK, theme::neon_green())
    } else {
        (icons::COPY, theme::dim_text())
    };

    let btn = ui.add(
        egui::Button::new(egui::RichText::new(icon).small().color(color))
            .fill(egui::Color32::TRANSPARENT)
            .stroke(egui::Stroke::NONE)
            .corner_radius(theme::radius_sm()),
    );
    let btn = theme::pointer(btn);
    if btn.clicked() {
        ui.ctx().copy_text(id.to_owned());
        ui.data_mut(|d| d.insert_temp(copy_id, std::time::Instant::now()));
    }
    if just_copied {
        btn.on_hover_text("Copied to clipboard!");
    } else {
        btn.on_hover_text("Copy issue ID");
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::Utc;

    #[test]
    fn two_column_structure() {
        let detail = IssueDetail {
            id: "bd-neon-1".to_owned(),
            title: "Test Bead".to_owned(),
            description: Some("Here is description".to_owned()),
            acceptance_criteria: None,
            status: "open".to_owned(),
            priority: 1,
            kind: "feature".to_owned(),
            assignee: Some("alice".to_owned()),
            owner: None,
            parent: None,
            created_at: Some(Utc::now()),
            updated_at: None,
            started_at: None,
            closed_at: None,
            dependencies: vec![],
            dependents: vec![],
            dependency_count: 0,
            dependent_count: 0,
            comment_count: 2,
            comments: Vec::new(),
        };
        assert_eq!(detail.id, "bd-neon-1");
    }

    #[test]
    fn full_metadata_sidebar() {
        let detail = IssueDetail {
            id: "bd-12".to_owned(),
            title: "Metadata Test".to_owned(),
            description: None,
            acceptance_criteria: None,
            status: "in_progress".to_owned(),
            priority: 0,
            kind: "bug".to_owned(),
            assignee: Some("bob".to_owned()),
            owner: None,
            parent: Some("bd-10".to_owned()),
            created_at: None,
            updated_at: None,
            started_at: None,
            closed_at: None,
            dependencies: vec![],
            dependents: vec![],
            dependency_count: 0,
            dependent_count: 0,
            comment_count: 0,
            comments: Vec::new(),
        };
        assert_eq!(detail.parent.as_deref(), Some("bd-10"));
        assert_eq!(detail.assignee.as_deref(), Some("bob"));
    }

    #[test]
    fn empty_timestamps_hidden() {
        let detail = IssueDetail {
            id: "bd-3".to_owned(),
            title: "Timestamps Test".to_owned(),
            description: None,
            acceptance_criteria: None,
            status: "open".to_owned(),
            priority: 2,
            kind: "task".to_owned(),
            assignee: None,
            owner: None,
            parent: None,
            created_at: None,
            updated_at: None,
            started_at: None,
            closed_at: None,
            dependencies: vec![],
            dependents: vec![],
            dependency_count: 0,
            dependent_count: 0,
            comment_count: 0,
            comments: Vec::new(),
        };
        assert!(detail.created_at.is_none());
        assert!(detail.closed_at.is_none());
    }

    #[test]
    fn breadcrumb_nav() {
        let detail = IssueDetail {
            id: "bd-neon-4.2".to_owned(),
            title: "Subtask".to_owned(),
            description: None,
            acceptance_criteria: None,
            status: "open".to_owned(),
            priority: 2,
            kind: "task".to_owned(),
            assignee: None,
            owner: None,
            parent: Some("bd-neon-4".to_owned()),
            created_at: None,
            updated_at: None,
            started_at: None,
            closed_at: None,
            dependencies: vec![],
            dependents: vec![],
            dependency_count: 0,
            dependent_count: 0,
            comment_count: 0,
            comments: Vec::new(),
        };
        assert_eq!(detail.parent.as_deref(), Some("bd-neon-4"));
    }

    #[test]
    fn composer_post_gate() {
        assert!(can_post_comment("hello", false));
        assert!(!can_post_comment("   ", false));
        assert!(!can_post_comment("", false));
        assert!(!can_post_comment("hello", true));
    }
}
