mod detail;
mod index;
mod paths;
mod snapshot;

#[allow(unused_imports)]
pub use detail::{load_detail, save_detail};
#[allow(unused_imports)]
pub use paths::{beads_mtime, dirname_for};
#[allow(unused_imports)]
pub use snapshot::{
    CachedSnapshot, LoadedCache, load_cached_fast, save_snapshot, save_snapshot_async,
};

#[cfg(test)]
pub use snapshot::load_for_repo;

#[cfg(test)]
#[allow(clippy::redundant_pub_crate)]
pub(crate) static TEST_ENV_LOCK: std::sync::Mutex<()> = std::sync::Mutex::new(());

#[cfg(test)]
mod tests {
    use std::collections::HashSet;
    use std::path::PathBuf;
    use std::sync::MutexGuard;
    use std::time::{Duration, SystemTime};

    use chrono::Utc;

    use crate::model::{Issue, IssueDetail, Snapshot};

    use super::TEST_ENV_LOCK as ENV_LOCK;
    use super::{beads_mtime, dirname_for, load_detail, load_for_repo, save_detail, save_snapshot};
    use crate::cache::index::CacheIndex;
    use crate::cache::index::MAX_ENTRIES;
    use crate::cache::paths::snapshot_path;

    fn lock_env() -> MutexGuard<'static, ()> {
        ENV_LOCK
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }

    fn temp_root(name: &str) -> PathBuf {
        let base = std::env::temp_dir().join("fabrik/neon-cache-tests").join(name);
        let _ = std::fs::remove_dir_all(&base);
        std::fs::create_dir_all(&base).unwrap();
        base
    }

    fn issue(id: &str) -> Issue {
        Issue {
            id: id.to_owned(),
            title: id.to_owned(),
            status: "open".to_owned(),
            priority: 2,
            kind: "task".to_owned(),
            assignee: None,
            updated_at: None,
            description: None,
            acceptance_criteria: None,
        }
    }

    fn snapshot(repo: &str, ids: &[&str]) -> Snapshot {
        Snapshot {
            repo: PathBuf::from(repo),
            issues: ids.iter().map(|id| issue(id)).collect(),
            ready_ids: HashSet::new(),
            fetched_at: Utc::now(),
        }
    }

    fn detail(id: &str) -> IssueDetail {
        IssueDetail {
            id: id.to_owned(),
            title: id.to_owned(),
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
            dependencies: Vec::new(),
            dependents: Vec::new(),
            dependency_count: 0,
            dependent_count: 0,
            comment_count: 0,
            comments: Vec::new(),
        }
    }

    fn repo_with_beads(root: &std::path::Path, name: &str) -> PathBuf {
        let repo = root.join(name);
        std::fs::create_dir_all(repo.join(".beads")).unwrap();
        std::fs::write(repo.join(".beads").join("issues.jsonl"), "{}\n").unwrap();
        repo
    }

    #[test]
    fn roundtrip_snapshot_and_detail() {
        let _guard = lock_env();
        let root = temp_root("roundtrip");
        unsafe { std::env::set_var("NEON_CACHE_DIR", &root) };
        let repo = repo_with_beads(&root, "repo-a");
        let snap = snapshot("x", &["a-1", "a-2"]);
        let snap = Snapshot {
            repo: repo.clone(),
            ..snap
        };
        save_snapshot(&repo, &snap);
        let loaded = load_for_repo(&repo).expect("cache hit");
        assert_eq!(loaded.issues.len(), 2);
        assert_eq!(loaded.issues[0].id, "a-1");

        let det = detail("a-1");
        save_detail(&repo, &det);
        let back = load_detail(&repo, "a-1").expect("detail hit");
        assert_eq!(back.id, "a-1");
        unsafe { std::env::remove_var("NEON_CACHE_DIR") };
    }

    #[test]
    fn corrupt_snapshot_falls_back_to_none() {
        let _guard = lock_env();
        let root = temp_root("corrupt");
        unsafe { std::env::set_var("NEON_CACHE_DIR", &root) };
        let repo = repo_with_beads(&root, "repo-b");
        let snap = Snapshot {
            repo: repo.clone(),
            ..snapshot("x", &["b-1"])
        };
        save_snapshot(&repo, &snap);
        let path = snapshot_path(&repo).unwrap();
        std::fs::write(&path, "{not json").unwrap();
        assert!(load_for_repo(&repo).is_none());
        unsafe { std::env::remove_var("NEON_CACHE_DIR") };
    }

    #[test]
    fn stale_then_fresh_replace() {
        let _guard = lock_env();
        let root = temp_root("stale-fresh");
        unsafe { std::env::set_var("NEON_CACHE_DIR", &root) };
        let repo = repo_with_beads(&root, "repo-c");
        let old = Snapshot {
            repo: repo.clone(),
            ..snapshot("x", &["old-1"])
        };
        save_snapshot(&repo, &old);
        let loaded = load_for_repo(&repo).expect("old cached");
        assert_eq!(loaded.issues[0].id, "old-1");
        let fresh = Snapshot {
            repo: repo.clone(),
            ..snapshot("x", &["new-1", "new-2"])
        };
        save_snapshot(&repo, &fresh);
        let reloaded = load_for_repo(&repo).expect("fresh cached");
        assert_eq!(reloaded.issues.len(), 2);
        assert_eq!(reloaded.issues[0].id, "new-1");
        unsafe { std::env::remove_var("NEON_CACHE_DIR") };
    }

    #[test]
    fn eviction_caps_entries() {
        let _guard = lock_env();
        let root = temp_root("evict");
        unsafe { std::env::set_var("NEON_CACHE_DIR", &root) };
        for i in 0..15 {
            let repo = repo_with_beads(&root, &format!("repo-{i}"));
            let snap = Snapshot {
                repo: repo.clone(),
                ..snapshot("x", &["e-1"])
            };
            save_snapshot(&repo, &snap);
            std::thread::sleep(Duration::from_millis(5));
        }
        let index_bytes = std::fs::read(root.join("index.json")).expect("index exists");
        let index: CacheIndex = serde_json::from_slice(&index_bytes).unwrap();
        assert!(index.entries.len() <= MAX_ENTRIES);
        unsafe { std::env::remove_var("NEON_CACHE_DIR") };
    }

    #[test]
    fn mtime_change_marks_stale_but_still_serves() {
        let _guard = lock_env();
        let root = temp_root("mtime");
        unsafe { std::env::set_var("NEON_CACHE_DIR", &root) };
        let repo = repo_with_beads(&root, "repo-m");
        let snap = Snapshot {
            repo: repo.clone(),
            ..snapshot("x", &["m-1"])
        };
        save_snapshot(&repo, &snap);
        let first = load_for_repo(&repo).expect("hit");
        assert!(!first.stale);
        let future = SystemTime::now() + Duration::from_secs(60);
        let handle = std::fs::File::options()
            .write(true)
            .open(repo.join(".beads").join("issues.jsonl"))
            .unwrap();
        handle.set_modified(future).unwrap();
        drop(handle);
        assert!(beads_mtime(&repo).is_some());
        let second = load_for_repo(&repo).expect("stale-while-revalidate");
        assert_eq!(second.issues.len(), 1);
        assert!(second.stale);
        unsafe { std::env::remove_var("NEON_CACHE_DIR") };
    }

    #[test]
    fn no_cross_repo_pollution() {
        let _guard = lock_env();
        let root = temp_root("nopollute");
        unsafe { std::env::set_var("NEON_CACHE_DIR", &root) };
        let repo_a = repo_with_beads(&root, "repo-pa");
        let repo_b = repo_with_beads(&root, "repo-pb");
        let snap_a = Snapshot {
            repo: repo_a.clone(),
            ..snapshot("x", &["only-a"])
        };
        save_snapshot(&repo_a, &snap_a);
        assert_ne!(dirname_for(&repo_a), dirname_for(&repo_b));
        assert!(load_for_repo(&repo_b).is_none());
        let hit_a = load_for_repo(&repo_a).expect("a hit");
        assert_eq!(hit_a.issues[0].id, "only-a");
        unsafe { std::env::remove_var("NEON_CACHE_DIR") };
    }
}
