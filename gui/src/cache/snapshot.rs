use std::collections::HashSet;
use std::fs;
use std::path::Path;
use std::path::PathBuf;

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

use crate::model::{Issue, Snapshot};

use super::index::{TTL_SECS, touch_index, touch_index_async};
use super::paths::{
    atomic_write, beads_mtime, cache_root, dir_size, dirname_for, snapshot_path,
};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CachedSnapshot {
    pub repo: PathBuf,
    pub fetched_at: DateTime<Utc>,
    pub beads_mtime: Option<u64>,
    pub issues: Vec<Issue>,
    pub ready_ids: HashSet<String>,
}

#[derive(Debug, Clone)]
pub struct LoadedCache {
    pub issues: Vec<Issue>,
    pub ready_ids: HashSet<String>,
    pub fetched_at: DateTime<Utc>,
    pub stale: bool,
}

pub fn save_snapshot(repo: &Path, snapshot: &Snapshot) {
    let Some(dest) = snapshot_path(repo) else {
        return;
    };
    let cached = CachedSnapshot {
        repo: repo.to_path_buf(),
        fetched_at: snapshot.fetched_at,
        beads_mtime: beads_mtime(repo),
        issues: snapshot.issues.clone(),
        ready_ids: snapshot.ready_ids.clone(),
    };
    let Ok(bytes) = serde_json::to_vec(&cached) else {
        return;
    };
    if !atomic_write(&dest, &bytes) {
        return;
    }
    if let Some(dir) = dest.parent() {
        touch_index(&dirname_for(repo), repo, dir_size(dir));
    }
}

fn load_inner(repo: &Path) -> Option<(CachedSnapshot, String, u64)> {
    let dirname = dirname_for(repo);
    let dest = cache_root().map(|root| root.join(&dirname).join("snapshot.json"))?;
    let bytes = fs::read(&dest).ok()?;
    let size_hint = u64::try_from(bytes.len()).unwrap_or(u64::MAX);
    let cached: CachedSnapshot = serde_json::from_slice(&bytes).ok().or_else(|| {
        let _ = fs::remove_file(&dest);
        None
    })?;
    let age = Utc::now()
        .signed_duration_since(cached.fetched_at)
        .num_seconds();
    if age > TTL_SECS {
        let _ = fs::remove_file(&dest);
        return None;
    }
    Some((cached, dirname, size_hint))
}

#[cfg(test)]
pub fn load_for_repo(repo: &Path) -> Option<LoadedCache> {
    let (cached, dirname, size_hint) = load_inner(repo)?;
    let current = beads_mtime(repo);
    let stale = current != cached.beads_mtime;
    touch_index_async(dirname, repo.to_path_buf(), size_hint);
    Some(LoadedCache {
        issues: cached.issues,
        ready_ids: cached.ready_ids,
        fetched_at: cached.fetched_at,
        stale,
    })
}

pub fn load_cached_fast(repo: &Path) -> Option<LoadedCache> {
    let (cached, dirname, size_hint) = load_inner(repo)?;
    touch_index_async(dirname, repo.to_path_buf(), size_hint);
    Some(LoadedCache {
        issues: cached.issues,
        ready_ids: cached.ready_ids,
        fetched_at: cached.fetched_at,
        stale: true,
    })
}

pub fn save_snapshot_async(snapshot: Snapshot) {
    std::thread::spawn(move || {
        let repo = snapshot.repo.clone();
        save_snapshot(&repo, &snapshot);
    });
}
