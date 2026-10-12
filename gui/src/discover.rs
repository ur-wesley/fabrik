use std::collections::HashSet;
use std::path::{Path, PathBuf};

use crate::model::RecentRepo;

/// Fuzzy-score weights for [`score_fuzzy`] (values match the doc comment).
const SCORE_PER_CHAR: i32 = 100;
const BONUS_START: i32 = 50;
const BONUS_BOUNDARY: i32 = 40;
const BONUS_CAMEL: i32 = 40;
const BONUS_CONSECUTIVE: i32 = 40;
const PENALTY_LEADING_PER_CHAR: i32 = 12;
const PENALTY_GAP_PER_CHAR: i32 = 8;
const PENALTY_LEN_PER_CHAR: i32 = 1;

pub fn beads_neighbors(seeds: &[PathBuf]) -> Vec<PathBuf> {
    // Sequential: typically 1-2 parent dirs; Rayon pool hops cost more than
    // the single `read_dir` per parent on the picker hot path.
    let parents: HashSet<PathBuf> = seeds
        .iter()
        .filter_map(|seed| seed.parent().map(Path::to_path_buf))
        .collect();
    let mut found: Vec<PathBuf> = parents
        .into_iter()
        .flat_map(|parent| read_beads_dirs(&parent))
        .collect();
    found.sort();
    found.dedup();
    found
}

fn read_beads_dirs(parent: &Path) -> Vec<PathBuf> {
    let Ok(entries) = std::fs::read_dir(parent) else {
        return Vec::new();
    };
    entries
        .filter_map(|entry| {
            let path = entry.ok()?.path();
            is_fabrik_repo(&path).then_some(path)
        })
        .collect()
}

/// Case-insensitive subsequence fuzzy score.
///
/// Returns `None` when `needle` is not a subsequence of `haystack`.
/// Empty needle matches everything with score 0 so callers can keep
/// original order. Bonuses: consecutive run (+40), match at start (+50),
/// match after a path boundary (`/ \ _ - .` space, +40), `camelCase`
/// lower->Upper transition (+40). Penalties: gap between matches
/// (8 per char, 12 per leading char) and overall haystack length (1 per
/// char). Base score is 100 per matched char.
pub fn score_fuzzy(needle: &str, haystack: &str) -> Option<i32> {
    if needle.is_empty() {
        return Some(0);
    }
    // Single lowercase pass per side; matching lowercases on the fly from
    // the original hay chars so only two `Vec<char>` are built instead of
    // three, with an early length bail before any allocation.
    let needle_lower = needle.to_ascii_lowercase();
    let needle_chars: Vec<char> = needle_lower.chars().collect();
    let needle_len = needle_chars.len();
    if needle_len == 0 {
        return Some(0);
    }
    let hay_orig_chars: Vec<char> = haystack.chars().collect();
    let hay_len = hay_orig_chars.len();
    if needle_len > hay_len {
        return None;
    }
    let mut score: i32 = 0;
    let mut needle_idx: usize = 0;
    let mut last_match: Option<usize> = None;
    for (hay_idx, hay_orig) in hay_orig_chars.iter().enumerate() {
        let hay_ch = hay_orig.to_ascii_lowercase();
        let Some(want) = needle_chars.get(needle_idx).copied() else {
            break;
        };
        if hay_ch != want {
            continue;
        }
        score = score.saturating_add(SCORE_PER_CHAR);
        if hay_idx == 0 {
            score = score.saturating_add(BONUS_START);
        } else if let Some(prev_orig) = hay_orig_chars.get(hay_idx.saturating_sub(1)).copied() {
            // Boundary chars are ASCII symbols, so lower == orig here.
            if is_fuzzy_boundary(prev_orig.to_ascii_lowercase()) {
                score = score.saturating_add(BONUS_BOUNDARY);
            } else if is_camel_transition(prev_orig, *hay_orig) {
                score = score.saturating_add(BONUS_CAMEL);
            }
        }
        if last_match.is_some_and(|last| last.saturating_add(1) == hay_idx) {
            score = score.saturating_add(BONUS_CONSECUTIVE);
        }
        let gap_penalty = last_match.map_or_else(
            || {
                i32::try_from(hay_idx)
                    .unwrap_or(i32::MAX)
                    .saturating_mul(PENALTY_LEADING_PER_CHAR)
            },
            |last| {
                let gap = hay_idx.saturating_sub(last.saturating_add(1));
                i32::try_from(gap)
                    .unwrap_or(i32::MAX)
                    .saturating_mul(PENALTY_GAP_PER_CHAR)
            },
        );
        score = score.saturating_sub(gap_penalty);
        needle_idx = needle_idx.saturating_add(1);
        last_match = Some(hay_idx);
        if needle_idx >= needle_len {
            break;
        }
    }
    if needle_idx != needle_len {
        return None;
    }
    score = score.saturating_sub(
        i32::try_from(hay_len)
            .unwrap_or(i32::MAX)
            .saturating_mul(PENALTY_LEN_PER_CHAR),
    );
    Some(score)
}

const fn is_fuzzy_boundary(value: char) -> bool {
    matches!(value, '/' | '\\' | '_' | '-' | '.' | ' ')
}

const fn is_camel_transition(prev: char, current: char) -> bool {
    prev.is_ascii_lowercase() && current.is_ascii_uppercase()
}

/// Rank `items` for `query`, scoring the project name with 2x weight plus
/// the full path score. Empty query returns items in original order.
/// Non-matching items (neither name nor path is a subsequence) are
/// filtered out. Sort is stable-descending, so ties keep input order.
///
/// Picker hot path (per keystroke): the needle is lowercased once here for
/// the length pre-check; per-item `to_string_lossy` runs once and items
/// are capped so a huge neighbor list cannot stall the UI.
///
/// Contract: this function never synthesizes the typed-input row. Callers
/// (see [`suggest`]) must pin `typed_beads(query)` on top afterwards; the
/// scorer must not filter that row out once the caller adds it.
pub fn suggest_fuzzy(query: &str, items: &[RecentRepo]) -> Vec<RecentRepo> {
    const MAX_FUZZY_ITEMS: usize = 500;
    let needle = query.trim();
    if needle.is_empty() {
        return items.to_vec();
    }
    let needle_len = needle.chars().count();
    let mut scored: Vec<(i32, RecentRepo)> = Vec::new();
    for item in items.iter().take(MAX_FUZZY_ITEMS) {
        // Cheap length pre-check before the lowercase + subsequence scan.
        // `to_string_lossy` allocates once per item per keystroke; the cap
        // above bounds the total work.
        let path_text = item.path.to_string_lossy();
        if needle_len > item.name.chars().count() && needle_len > path_text.chars().count() {
            continue;
        }
        let name_score = score_fuzzy(needle, &item.name);
        let path_score = score_fuzzy(needle, &path_text);
        if name_score.is_none() && path_score.is_none() {
            continue;
        }
        let combined = name_score
            .unwrap_or(0)
            .saturating_mul(2)
            .saturating_add(path_score.unwrap_or(0));
        scored.push((combined, item.clone()));
    }
    scored.sort_by_key(|entry| std::cmp::Reverse(entry.0));
    scored.into_iter().map(|(_, item)| item).collect()
}

pub fn is_fabrik_repo(path: &Path) -> bool {
    path.join(".fabrik/config.yaml").is_file() && path.join(".fabrik/dolt").is_dir()
}

pub fn is_git_repo(path: &Path) -> bool {
    path.join(".git").exists()
}

/// Any directory the user may open as a workspace (Fabrik repo or not yet inited).
pub fn workspace_from_picked(path: PathBuf) -> Option<PathBuf> {
    if let Some(repo) = repo_from_picked(path.clone()) {
        return Some(repo);
    }
    let dir = if path.file_name().is_some_and(|name| name == ".fabrik") {
        path.parent().map(Path::to_path_buf)?
    } else {
        path
    };
    dir.is_dir().then_some(dir)
}

pub fn typed_beads(query: &str) -> Option<PathBuf> {
    let trimmed = query.trim();
    if trimmed.is_empty() {
        return None;
    }
    repo_from_picked(PathBuf::from(trimmed))
}

/// Resolve a user-picked or typed path to the nearest Fabrik repo root.
pub fn repo_from_picked(path: PathBuf) -> Option<PathBuf> {
    let start = if path.file_name().is_some_and(|name| name == ".fabrik") {
        path.parent().map(Path::to_path_buf)?
    } else {
        path
    };
    let mut current = start;
    for _ in 0..32 {
        if is_fabrik_repo(&current) {
            return Some(current);
        }
        let parent = current.parent()?;
        if parent == current {
            break;
        }
        current = parent.to_path_buf();
    }
    None
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::model::recent_from_path;

    #[test]
    fn typed_beads_requires_db() {
        assert!(typed_beads("").is_none());
        let fabrik = PathBuf::from(r"C:\Arbeit\fabrik");
        if is_fabrik_repo(&fabrik) {
            assert_eq!(typed_beads(r"C:\Arbeit\fabrik"), Some(fabrik));
        }
    }

    #[test]
    fn repo_from_picked_finds_manifest_dir() {
        let root = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
        assert_eq!(repo_from_picked(root.clone()), Some(root.clone()));
        if is_fabrik_repo(&root) {
            assert_eq!(repo_from_picked(root.join(".fabrik")), Some(root));
        }
    }

    #[test]
    fn fuzzy_miss_returns_none() {
        assert!(score_fuzzy("xyz", "bd-neon").is_none());
        assert!(score_fuzzy("bn", "abc").is_none());
        assert!(score_fuzzy("longneedle", "short").is_none());
    }

    #[test]
    fn fuzzy_bn_matches_bd_neon() {
        assert!(score_fuzzy("bn", "bd-neon").is_some());
    }

    #[test]
    fn fuzzy_fabrik_matches_full_path() {
        assert!(score_fuzzy("fabrik", r"C:\Arbeit\fabrik").is_some());
    }

    #[test]
    fn fuzzy_consecutive_beats_scattered() {
        let tight = score_fuzzy("ab", "ab").unwrap_or(i32::MIN);
        let loose = score_fuzzy("ab", "a--b").unwrap_or(i32::MAX);
        assert!(tight > loose);
    }

    #[test]
    fn fuzzy_start_bonus() {
        let at_start = score_fuzzy("bd", "bd-neon").unwrap_or(i32::MIN);
        let later = score_fuzzy("bd", "xxbd-neon").unwrap_or(i32::MAX);
        assert!(at_start > later);
    }

    #[test]
    fn fuzzy_boundary_bonus() {
        let boundary = score_fuzzy("n", "a-n").unwrap_or(i32::MIN);
        let plain = score_fuzzy("n", "axn").unwrap_or(i32::MAX);
        assert!(boundary > plain);
    }

    #[test]
    fn fuzzy_camel_bonus() {
        let camel = score_fuzzy("fb", "fooBar").unwrap_or(i32::MIN);
        let flat = score_fuzzy("fb", "foobar").unwrap_or(i32::MAX);
        assert!(camel > flat);
    }

    #[test]
    fn fuzzy_empty_needle() {
        assert_eq!(score_fuzzy("", "anything"), Some(0));
        let items = vec![recent_from_path(PathBuf::from(r"C:\Arbeit\fabrik"))];
        let ranked = suggest_fuzzy("", &items);
        assert_eq!(ranked.len(), 1);
    }

    #[test]
    fn fuzzy_name_weighting_and_ordering() {
        let named = RecentRepo {
            path: PathBuf::from("/work/neon-helper"),
            name: "neon-helper".to_owned(),
            parent: "/work".to_owned(),
        };
        let path_only = RecentRepo {
            path: PathBuf::from("/neon/other-tool"),
            name: "other-tool".to_owned(),
            parent: "/neon".to_owned(),
        };
        let ranked = suggest_fuzzy("neon", &[path_only.clone(), named.clone()]);
        let first = ranked.first().cloned().unwrap_or(path_only);
        assert_eq!(first, named);

        let exact = RecentRepo {
            path: PathBuf::from("/work/ab"),
            name: "ab".to_owned(),
            parent: "/work".to_owned(),
        };
        let scattered = RecentRepo {
            path: PathBuf::from("/work/a--b"),
            name: "a--b".to_owned(),
            parent: "/work".to_owned(),
        };
        let ordered = suggest_fuzzy("ab", &[scattered.clone(), exact.clone()]);
        let top = ordered.first().cloned().unwrap_or(scattered);
        assert_eq!(top, exact);
    }
}
