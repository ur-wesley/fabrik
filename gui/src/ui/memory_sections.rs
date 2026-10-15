use super::icons;

/// One visual section split out of a structured memory body.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct MemSection {
    /// Display label (trimmed raw label, e.g. `What`, `Next Steps`).
    pub label: String,
    /// Normalized lowercase key (e.g. `what`, `next steps`, `files`).
    pub key: String,
    /// Section body text (inline value + following lines, trimmed).
    pub body: String,
}

/// Normalize a raw header label to a known key.
///
/// Returns `None` for unknown labels.
#[must_use]
pub fn normalize_key(raw: &str) -> Option<String> {
    let lowered = raw.trim().to_lowercase();
    // Collapse internal whitespace runs so `Next   Steps` still matches.
    let collapsed = lowered.split_whitespace().collect::<Vec<_>>().join(" ");
    let mapped = match collapsed.as_str() {
        "nextsteps" => "next steps".to_owned(),
        "relevant files" => "files".to_owned(),
        "discovered" => "discoveries".to_owned(),
        other => other.to_owned(),
    };
    match mapped.as_str() {
        "what" | "why" | "where" | "learned" | "goal" | "instructions" | "discoveries"
        | "accomplished" | "next steps" | "files" => Some(mapped),
        _ => None,
    }
}

fn strip_bold_markers(text: &str) -> &str {
    let mut out = text.trim();
    for marker in ["**", "__"] {
        if out.len() >= marker.len() * 2
            && out.starts_with(marker)
            && out.ends_with(marker)
            && out.len() > marker.len() * 2
        {
            out = out[marker.len()..out.len() - marker.len()].trim();
            break;
        }
    }
    // Single-sided emphasis leftovers (e.g. `*What*` without full strip).
    out = out
        .trim_start_matches(['*', '_', '`'])
        .trim_end_matches(['*', '_', '`'])
        .trim();
    out
}

/// Try to parse one line as a section header.
///
/// Returns `(raw_label, inline_value)` on success.
fn parse_header_line(line: &str) -> Option<(String, String)> {
    let trimmed = line.trim();
    if trimmed.is_empty() {
        return None;
    }

    // 1. Markdown headings: `#` … `######` + label (+ optional `: value`).
    if trimmed.starts_with('#') {
        let hashes = trimmed.chars().take_while(|c| *c == '#').count();
        if (1..=6).contains(&hashes) {
            let rest = trimmed[hashes..].trim();
            // Require a space or direct label after hashes (`##Goal` is
            // tolerated, `###` alone is not a header).
            if !rest.is_empty() {
                let (label_part, value_part) = match rest.split_once(':') {
                    Some((left, right)) => (left.trim(), right.trim().to_owned()),
                    None => (rest, String::new()),
                };
                let label = strip_bold_markers(label_part.trim().trim_end_matches(':').trim());
                if !label.is_empty() && normalize_key(label).is_some() {
                    return Some((label.to_owned(), value_part));
                }
            }
        }
    }

    // 2. Bold-colon forms: `**Label**: value`, `**Label:** value`,
    //    `**Label** : value` (and `__` equivalents).
    for marker in ["**", "__"] {
        if trimmed.starts_with(marker) {
            let after_open = &trimmed[marker.len()..];
            if let Some(close) = after_open.find(marker) {
                let inner = after_open[..close].trim();
                let rest = after_open[close + marker.len()..].trim_start();
                let inner_has_colon = inner.ends_with(':');
                let rest_has_colon = rest.starts_with(':');
                if inner_has_colon || rest_has_colon {
                    let label = strip_bold_markers(inner.trim_end_matches(':').trim());
                    if !label.is_empty() && normalize_key(label).is_some() {
                        let value = if rest_has_colon {
                            rest[1..].trim().to_owned()
                        } else {
                            rest.to_owned()
                        };
                        return Some((label.to_owned(), value));
                    }
                    // Bold prefix with a known label but no usable split is
                    // still not a header; fall through to plain handling.
                    return None;
                }
                // `**bold** text` without a colon is emphatic prose, not a
                // section header.
                return None;
            }
        }
    }

    // 3. Plain `Label: value`, only for known labels.
    if let Some((left, right)) = trimmed.split_once(':') {
        let label = left.trim();
        // Reject list/quote/callout prefixes and decorated labels so
        // `- Note: x` or `> Why: y` stay prose.
        if label.is_empty()
            || label.contains(['*', '#', '`', '|', '[', ']', '>', '-', '+'])
            || label.len() > 24
        {
            return None;
        }
        if normalize_key(label).is_some() {
            return Some((label.to_owned(), right.trim().to_owned()));
        }
    }

    None
}

/// Split a raw memory body into preamble + visual sections.
///
/// Supports Engram `**What**: …` roles and session-summary `## Goal`
/// headings. Returns an empty section list when fewer than two sections
/// are found so single-`Note: …` bodies fall back to plain rendering.
#[must_use]
pub fn parse_sections(raw: &str) -> (String, Vec<MemSection>) {
    let normalized = raw.replace("\r\n", "\n").replace('\r', "\n");
    let mut preamble_lines: Vec<&str> = Vec::new();
    let mut sections: Vec<MemSection> = Vec::new();
    let mut bodies: Vec<Vec<String>> = Vec::new();
    let mut seen_header = false;

    for line in normalized.lines() {
        if let Some((label, inline)) = parse_header_line(line) {
            let key = normalize_key(&label).unwrap_or_else(|| label.to_lowercase());
            sections.push(MemSection {
                label: label.clone(),
                key,
                body: String::new(),
            });
            bodies.push(Vec::new());
            if !inline.is_empty() {
                if let Some(last) = bodies.last_mut() {
                    last.push(inline);
                }
            }
            seen_header = true;
        } else if seen_header {
            if let Some(last) = bodies.last_mut() {
                last.push(line.to_owned());
            }
        } else {
            preamble_lines.push(line);
        }
    }

    let preamble = preamble_lines.join("\n").trim().to_owned();
    for (section, lines) in sections.iter_mut().zip(bodies.iter()) {
        // Drop leading/trailing blank lines; keep interior breaks.
        let mut start = 0;
        let mut end = lines.len();
        while start < end && lines[start].trim().is_empty() {
            start += 1;
        }
        while end > start && lines[end - 1].trim().is_empty() {
            end -= 1;
        }
        section.body = lines[start..end].join("\n").trim().to_owned();
    }

    if sections.len() < 2 {
        // Avoid false splits: a lone `**What**: …` or `Note: …` stays a
        // single unstructured body.
        return (normalized.trim().to_owned(), Vec::new());
    }
    (preamble, sections)
}

/// Whether `raw` should render as split section cards.
#[must_use]
pub fn is_structured(raw: &str) -> bool {
    parse_sections(raw).1.len() >= 2
}

/// Phosphor icon for a normalized section key.
#[must_use]
pub fn section_icon(key: &str) -> &'static str {
    match key {
        "what" => icons::FILE_TEXT,
        "why" => icons::MAGNIFYING_GLASS,
        "where" => icons::FOLDER,
        "learned" => icons::CHECK_CIRCLE,
        "goal" => icons::CHECK_SQUARE,
        "instructions" => icons::GEAR,
        "discoveries" => icons::MAGNIFYING_GLASS,
        "accomplished" => icons::CHECK,
        "next steps" => icons::HOURGLASS,
        "files" => icons::FOLDER,
        _ => icons::LIST,
    }
}

#[cfg(test)]
mod tests {
    use super::{is_structured, normalize_key, parse_sections, section_icon};

    #[test]
    fn parses_what_block() {
        let raw = "**What**: one sentence\n**Why**: reason\n**Where**: files";
        let (preamble, sections) = parse_sections(raw);
        assert!(preamble.is_empty());
        assert_eq!(sections.len(), 3);
        assert_eq!(sections[0].key, "what");
        assert_eq!(sections[0].label, "What");
        assert_eq!(sections[0].body, "one sentence");
        assert_eq!(sections[1].key, "why");
        assert_eq!(sections[2].key, "where");
        assert!(is_structured(raw));
    }

    #[test]
    fn parses_goal_block() {
        let raw = "## Goal\nship it\n## Next Steps\n- a\n- b\n## Relevant Files\nfoo.rs";
        let (preamble, sections) = parse_sections(raw);
        assert!(preamble.is_empty());
        assert_eq!(sections.len(), 3);
        assert_eq!(sections[0].key, "goal");
        assert_eq!(sections[0].body, "ship it");
        assert_eq!(sections[1].key, "next steps");
        assert_eq!(sections[2].key, "files");
        assert_eq!(sections[2].label, "Relevant Files");
    }

    #[test]
    fn single_colon_no_false_positive() {
        let raw = "Note: just a plain note";
        let (preamble, sections) = parse_sections(raw);
        assert!(sections.is_empty());
        assert_eq!(preamble, raw);
        assert!(!is_structured(raw));
    }

    #[test]
    fn single_known_header_falls_back() {
        let raw = "**What**: only one role here";
        let (_, sections) = parse_sections(raw);
        assert!(sections.is_empty());
        assert!(!is_structured(raw));
    }

    #[test]
    fn keeps_preamble() {
        let raw = "Intro line here.\n**What**: a\n**Why**: b";
        let (preamble, sections) = parse_sections(raw);
        assert_eq!(preamble, "Intro line here.");
        assert_eq!(sections.len(), 2);
    }

    #[test]
    fn learned_multiline() {
        let raw = "**What**: a\n**Learned**: line one\nline two\n\nline three\n**Why**: b";
        let (_, sections) = parse_sections(raw);
        assert_eq!(sections.len(), 3);
        assert_eq!(sections[1].key, "learned");
        assert_eq!(sections[1].body, "line one\nline two\n\nline three");
    }

    #[test]
    fn header_variants() {
        let raw = "**What:** a\n**Why** : b\n### Where\nthere";
        let (_, sections) = parse_sections(raw);
        assert_eq!(sections.len(), 3);
        assert_eq!(sections[0].body, "a");
        assert_eq!(sections[1].body, "b");
        assert_eq!(sections[2].key, "where");
    }

    #[test]
    fn plain_labels_case_insensitive() {
        let raw = "WHAT: a\nwhy: b";
        let (_, sections) = parse_sections(raw);
        assert_eq!(sections.len(), 2);
        assert_eq!(sections[0].key, "what");
        assert_eq!(sections[1].key, "why");
    }

    #[test]
    fn discovered_normalizes() {
        assert_eq!(normalize_key("discovered"), Some("discoveries".to_owned()));
        assert_eq!(normalize_key("NextSteps"), Some("next steps".to_owned()));
        assert_eq!(normalize_key("Relevant Files"), Some("files".to_owned()));
        assert_eq!(normalize_key("Note"), None);
    }

    #[test]
    fn icons_cover_known_keys() {
        assert!(!section_icon("what").is_empty());
        assert!(!section_icon("bogus").is_empty());
    }
}
