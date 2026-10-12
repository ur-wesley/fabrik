use std::fs;
use std::path::{Path, PathBuf};
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::UNIX_EPOCH;

static TMP_COUNTER: AtomicU64 = AtomicU64::new(0);

#[cfg(test)]
pub fn cache_root() -> Option<PathBuf> {
    std::env::var("NEON_CACHE_DIR")
        .ok()
        .filter(|s| !s.trim().is_empty())
        .map(PathBuf::from)
}

#[cfg(not(test))]
pub fn cache_root() -> Option<PathBuf> {
    if let Ok(dir) = std::env::var("NEON_CACHE_DIR")
        && !dir.trim().is_empty()
    {
        return Some(PathBuf::from(dir));
    }
    #[cfg(windows)]
    let base = std::env::var_os("APPDATA");
    #[cfg(not(windows))]
    let base = std::env::var_os("XDG_CONFIG_HOME").or_else(|| {
        std::env::var_os("HOME").map(|h| PathBuf::from(h).join(".config").into_os_string())
    });
    base.map(|b| PathBuf::from(b).join("fabrik/neon").join("cache"))
}

fn fnv1a64(bytes: &[u8]) -> u64 {
    let mut hash: u64 = 0xcbf2_9ce4_8422_2325;
    for byte in bytes {
        hash ^= u64::from(*byte);
        hash = hash.wrapping_mul(0x0100_0000_01b3);
    }
    hash
}

fn canonical_string(repo: &Path) -> String {
    fs::canonicalize(repo).map_or_else(
        |_| repo.to_string_lossy().into_owned(),
        |canonical| canonical.to_string_lossy().into_owned(),
    )
}

fn slug_for(repo: &Path) -> String {
    let raw = repo
        .file_name()
        .map_or_else(|| "repo".to_owned(), |n| n.to_string_lossy().into_owned());
    let mut slug = String::new();
    for ch in raw.chars() {
        if ch.is_ascii_alphanumeric() {
            slug.push(ch.to_ascii_lowercase());
        } else if !slug.ends_with('-') && !slug.is_empty() {
            slug.push('-');
        }
        if slug.len() >= 32 {
            break;
        }
    }
    let trimmed = slug.trim_matches('-').to_owned();
    if trimmed.is_empty() {
        "repo".to_owned()
    } else {
        trimmed
    }
}

pub fn dirname_for(repo: &Path) -> String {
    let canonical = canonical_string(repo);
    let hash = fnv1a64(canonical.as_bytes());
    format!("{}-{hash:016x}", slug_for(repo))
}

fn repo_dir(repo: &Path) -> Option<PathBuf> {
    cache_root().map(|root| root.join(dirname_for(repo)))
}

pub(crate) fn snapshot_path(repo: &Path) -> Option<PathBuf> {
    repo_dir(repo).map(|d| d.join("snapshot.json"))
}

pub(crate) fn detail_path(repo: &Path, id: &str) -> Option<PathBuf> {
    repo_dir(repo).map(|d| d.join("details").join(format!("{}.json", sanitize_id(id))))
}

fn sanitize_id(id: &str) -> String {
    let mut out = String::with_capacity(id.len());
    for ch in id.chars() {
        if ch.is_ascii_alphanumeric() || ch == '-' || ch == '_' || ch == '.' {
            out.push(ch);
        } else {
            out.push('_');
        }
    }
    if out.is_empty() { "_".to_owned() } else { out }
}

pub(crate) fn atomic_write(dest: &Path, bytes: &[u8]) -> bool {
    let Some(parent) = dest.parent() else {
        return false;
    };
    if fs::create_dir_all(parent).is_err() {
        return false;
    }
    let count = TMP_COUNTER.fetch_add(1, Ordering::Relaxed);
    let pid = std::process::id();
    let tmp = dest.with_extension(format!("json.tmp.{pid}.{count}"));
    if fs::write(&tmp, bytes).is_err() {
        return false;
    }
    fs::rename(&tmp, dest).is_ok()
}

pub fn beads_mtime(repo: &Path) -> Option<u64> {
    let dolt = repo.join(".fabrik/dolt");
    if !dolt.is_dir() {
        return None;
    }
    let mut stack = vec![dolt];
    let mut max: Option<u64> = None;
    while let Some(dir) = stack.pop() {
        let Ok(entries) = fs::read_dir(&dir) else {
            continue;
        };
        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() {
                stack.push(path);
                continue;
            }
            if let Ok(meta) = entry.metadata()
                && let Ok(modified) = meta.modified()
                && let Ok(age) = modified.duration_since(UNIX_EPOCH)
            {
                let secs = age.as_secs();
                max = Some(max.map_or(secs, |m: u64| m.max(secs)));
            }
        }
    }
    max
}

pub(crate) fn dir_size(dir: &Path) -> u64 {
    let mut total: u64 = 0;
    let mut stack = vec![dir.to_path_buf()];
    while let Some(current) = stack.pop() {
        let Ok(entries) = fs::read_dir(&current) else {
            continue;
        };
        for entry in entries.flatten() {
            let path = entry.path();
            if path.is_dir() {
                stack.push(path);
            } else if let Ok(meta) = entry.metadata() {
                let is_tmp_orphan = entry.file_name().to_string_lossy().contains(".tmp.");
                if !is_tmp_orphan {
                    total = total.saturating_add(meta.len());
                }
            }
        }
    }
    total
}
