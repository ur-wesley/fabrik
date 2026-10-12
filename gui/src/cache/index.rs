use std::fs;
use std::path::{Path, PathBuf};

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

use super::paths::{atomic_write, cache_root, dir_size};

pub(crate) const MAX_ENTRIES: usize = 12;
pub(crate) const MAX_BYTES: u64 = 10 * 1024 * 1024;
pub(crate) const TTL_SECS: i64 = 7 * 24 * 3600;

static INDEX_LOCK: std::sync::Mutex<()> = std::sync::Mutex::new(());

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub(crate) struct CacheIndex {
    #[serde(default)]
    pub(crate) entries: Vec<IndexEntry>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub(crate) struct IndexEntry {
    dirname: String,
    repo: PathBuf,
    last_used: DateTime<Utc>,
    size_bytes: u64,
}

fn index_path() -> Option<PathBuf> {
    cache_root().map(|r| r.join("index.json"))
}

fn load_index() -> CacheIndex {
    let Some(path) = index_path() else {
        return CacheIndex::default();
    };
    let Ok(bytes) = fs::read(&path) else {
        return CacheIndex::default();
    };
    serde_json::from_slice(&bytes).unwrap_or_default()
}

fn save_index(index: &CacheIndex) {
    let Some(path) = index_path() else {
        return;
    };
    if let Ok(bytes) = serde_json::to_vec(index) {
        let _ = atomic_write(&path, &bytes);
    }
}

pub(crate) fn touch_index(dirname: &str, repo: &Path, size_bytes: u64) {
    let Some(root) = cache_root() else {
        return;
    };
    let _guard = INDEX_LOCK
        .lock()
        .unwrap_or_else(std::sync::PoisonError::into_inner);
    let mut index = load_index();
    let now = Utc::now();
    if let Some(entry) = index.entries.iter_mut().find(|e| e.dirname == dirname) {
        entry.repo = repo.to_path_buf();
        entry.last_used = now;
        entry.size_bytes = size_bytes;
    } else {
        index.entries.push(IndexEntry {
            dirname: dirname.to_owned(),
            repo: repo.to_path_buf(),
            last_used: now,
            size_bytes,
        });
    }
    save_index(&index);
    enforce_caps(&root, &mut index);
}

fn enforce_caps(root: &Path, index: &mut CacheIndex) {
    let now = Utc::now();
    let mut expired: Vec<String> = Vec::new();
    for entry in &index.entries {
        let age = now.signed_duration_since(entry.last_used).num_seconds();
        if age > TTL_SECS {
            expired.push(entry.dirname.clone());
        }
    }
    for dirname in &expired {
        let _ = fs::remove_dir_all(root.join(dirname));
    }
    index.entries.retain(|e| !expired.contains(&e.dirname));
    index.entries.sort_by_key(|entry| entry.last_used);
    while index.entries.len() > MAX_ENTRIES {
        if let Some(oldest) = index.entries.first().cloned() {
            let _ = fs::remove_dir_all(root.join(&oldest.dirname));
            index.entries.remove(0);
        } else {
            break;
        }
    }
    for entry in &mut index.entries {
        let size = dir_size(&root.join(&entry.dirname));
        entry.size_bytes = size;
    }
    let mut total: u64 = 0;
    for entry in &index.entries {
        total = total.saturating_add(entry.size_bytes);
    }
    while total > MAX_BYTES {
        if let Some(oldest) = index.entries.first().cloned() {
            let _ = fs::remove_dir_all(root.join(&oldest.dirname));
            total = total.saturating_sub(oldest.size_bytes);
            index.entries.remove(0);
        } else {
            break;
        }
    }
    save_index(index);
}

pub(crate) fn touch_index_async(dirname: String, repo: PathBuf, size_hint: u64) {
    std::thread::spawn(move || {
        touch_index(&dirname, &repo, size_hint);
    });
}
