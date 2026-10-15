use eframe::egui::{RichText, Ui};

use crate::fabrik::{DetectChangesResult, ExploreResult, GraphContextResult, GraphImpactResult};
use crate::model::Action;

use super::lookup::{cap, edge_row, foot, head, node_row, section};
use super::theme;

pub(super) fn context_card(
    ui: &mut Ui,
    symbol: &str,
    ctx: &GraphContextResult,
    anchor: Option<&str>,
    actions: &mut Vec<Action>,
) {
    theme::card_frame().show(ui, |ui| {
        head(ui, format!("context {symbol}"));
        foot(
            ui,
            format!(
                "{} · {} callers · {} callees · {} notes · {} tests",
                ctx.symbol.id,
                ctx.callers.len(),
                ctx.callees.len(),
                ctx.observations.len(),
                ctx.tests.len(),
            ),
        );
        if !ctx.callers.is_empty() {
            ui.add_space(theme::SPACING);
            section(ui, "callers");
            for (index, caller) in ctx.callers.iter().enumerate() {
                let loc = format!("{}:{}", caller.path, caller.line);
                node_row(
                    ui,
                    caller.name.clone(),
                    loc.clone(),
                    loc,
                    &caller.id,
                    anchor,
                    index + 1 == ctx.callers.len(),
                    actions,
                );
            }
        }
        if !ctx.callees.is_empty() {
            ui.add_space(theme::SPACING);
            section(ui, "callees");
            for edge in &ctx.callees {
                edge_row(ui, edge);
            }
        }
        if !ctx.observations.is_empty() {
            ui.add_space(theme::SPACING);
            section(ui, "observations");
            for obs in &ctx.observations {
                cap(ui, format!("{} · {} · {}", obs.title, obs.kind, obs.id));
            }
        }
        if !ctx.tests.is_empty() {
            ui.add_space(theme::SPACING);
            section(ui, "tests");
            for test in &ctx.tests {
                cap(ui, format!("{} · {}", test.symbol, test.path));
            }
        }
    });
}

pub(super) fn impact_card(
    ui: &mut Ui,
    symbol: &str,
    impact: &GraphImpactResult,
    anchor: Option<&str>,
    actions: &mut Vec<Action>,
) {
    theme::card_frame().show(ui, |ui| {
        head(
            ui,
            if impact.root.is_empty() {
                format!("impact {symbol} (total {})", impact.total)
            } else {
                format!(
                    "impact {symbol} (root {} · total {})",
                    impact.root, impact.total
                )
            },
        );
        let mut depths: Vec<&String> = impact.by_depth.keys().collect();
        depths.sort();
        for depth in depths {
            let Some(nodes) = impact.by_depth.get(depth) else {
                continue;
            };
            ui.add_space(theme::SPACING);
            section(ui, &format!("depth {depth}: {}", nodes.len()));
            for (index, node) in nodes.iter().enumerate() {
                let loc = format!("{}:{}", node.path, node.line);
                node_row(
                    ui,
                    node.name.clone(),
                    loc.clone(),
                    loc,
                    &node.id,
                    anchor,
                    index + 1 == nodes.len(),
                    actions,
                );
            }
        }
    });
}

pub(super) fn detect_card(ui: &mut Ui, base: &str, detect: &DetectChangesResult) {
    theme::card_frame().show(ui, |ui| {
        head(
            ui,
            format!("changes vs {}", if base.is_empty() { "HEAD" } else { base }),
        );
        foot(
            ui,
            format!(
                "{} files · {} touched · {} impacted",
                detect.changed_files.len(),
                detect.touched_symbols.len(),
                detect.impacted_symbols.len(),
            ),
        );
        list_section(ui, "changed files", &detect.changed_files);
        list_section(ui, "touched", &detect.touched_symbols);
        list_section(ui, "impacted", &detect.impacted_symbols);
        list_section(ui, "suggested checks", &detect.suggested_checks);
    });
}

fn list_section(ui: &mut Ui, title: &str, items: &[String]) {
    if items.is_empty() {
        return;
    }
    ui.add_space(theme::SPACING);
    section(ui, title);
    for item in items {
        cap(ui, item.clone());
    }
}

pub(super) fn explore_section(ui: &mut Ui, explore: &ExploreResult) {
    theme::card_frame().show(ui, |ui| {
        head(ui, "explore".to_owned());
        if !explore.hint.is_empty() {
            cap(ui, explore.hint.clone());
        }
        if !explore.files.is_empty() {
            ui.add_space(theme::SPACING);
            section(ui, "files");
            for file in &explore.files {
                foot(ui, file.path.clone());
                for snippet in &file.snippets {
                    let header = if snippet.symbol.is_empty() {
                        format!("lines {}–{}", snippet.line_start, snippet.line_end)
                    } else {
                        format!(
                            "{} · lines {}–{}",
                            snippet.symbol, snippet.line_start, snippet.line_end
                        )
                    };
                    cap(ui, header);
                    if !snippet.text.is_empty() {
                        ui.label(
                            RichText::new(snippet.text.clone())
                                .font(theme::font_caption2())
                                .color(theme::text_tertiary()),
                        );
                    }
                }
            }
        }
        if !explore.paths.is_empty() {
            ui.add_space(theme::SPACING);
            section(ui, "paths");
            for path in &explore.paths {
                let mut label = format!("{} → {}", path.from, path.to);
                if !path.hops.is_empty() {
                    label.push_str(&format!(" (via {})", path.hops.join(" → ")));
                }
                cap(ui, label);
            }
        }
        if !explore.candidates.is_empty() {
            ui.add_space(theme::SPACING);
            section(ui, "candidates");
            for candidate in &explore.candidates {
                cap(ui, candidate.clone());
            }
        }
        if let Some(blast) = explore.blast_radius.as_ref() {
            ui.add_space(theme::SPACING);
            section(
                ui,
                &format!("blast radius: {} dependents", blast.dependents_count),
            );
            if let Some(callers) = blast.callers.as_ref() {
                for caller in callers {
                    cap(ui, caller.clone());
                }
            }
        }
    });
}
