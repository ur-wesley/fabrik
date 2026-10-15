use std::fs;
use std::path::Path;

use crate::model::IssueDetail;

use super::index::touch_index_async;
use super::paths::{atomic_write, detail_path, dirname_for};

pub fn load_detail(repo: &Path, id: &str) -> Option<IssueDetail> {
    let dest = detail_path(repo, id)?;
    let bytes = fs::read(&dest).ok()?;
    serde_json::from_slice(&bytes).ok()
}

pub fn save_detail(repo: &Path, detail: &IssueDetail) {
    let Some(dest) = detail_path(repo, &detail.id) else {
        return;
    };
    if let Ok(bytes) = serde_json::to_vec(detail) {
        let size_hint = u64::try_from(bytes.len()).unwrap_or(u64::MAX);
        if atomic_write(&dest, &bytes) {
            touch_index_async(dirname_for(repo), repo.to_path_buf(), size_hint);
        }
    }
}
