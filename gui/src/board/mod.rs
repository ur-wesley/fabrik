use std::collections::{HashMap, HashSet};
use std::path::{Path, PathBuf};
use std::time::{Duration, Instant};

use chrono::{DateTime, Utc};

use crate::model::{
    ALL_LANES, Filter, Issue, IssueDetail, IssueRow, KanbanLane, Snapshot, ViewMode,
    format_clock, matches_filter, matches_search,
};

const FLASH_MS: u64 = 1200;

#[allow(clippy::struct_excessive_bools)]
pub struct BoardStore {
    pub issues: Vec<Issue>,
    pub ready_ids: HashSet<String>,
    pub view_mode: ViewMode,
    pub filter: Filter,
    pub search: String,
    pub search_focused: bool,
    pub selected: usize,
    pub detail_id: Option<String>,
    pub detail: Option<IssueDetail>,
    pub detail_cache: HashMap<String, IssueDetail>,
    pub comment_draft: String,
    pub comment_posting: bool,
    pub comment_error: Option<String>,
    pub collapsed_lanes: HashSet<Filter>,
    pub last_ok: Option<DateTime<Utc>>,
    pub error: Option<String>,
    pub loading: bool,
    pub stale: bool,
    pub flash_until: HashMap<String, Instant>,
    previous_updated: HashMap<String, DateTime<Utc>>,
}

impl BoardStore {
    pub fn new(collapsed_lanes: HashSet<Filter>) -> Self {
        Self {
            issues: Vec::new(),
            ready_ids: HashSet::new(),
            view_mode: ViewMode::Kanban,
            filter: Filter::Ready,
            search: String::new(),
            search_focused: false,
            selected: 0,
            detail_id: None,
            detail: None,
            detail_cache: HashMap::new(),
            comment_draft: String::new(),
            comment_posting: false,
            comment_error: None,
            collapsed_lanes,
            last_ok: None,
            error: None,
            loading: false,
            stale: false,
            flash_until: HashMap::new(),
            previous_updated: HashMap::new(),
        }
    }

    pub fn collapsed_vec(&self) -> Vec<Filter> {
        self.collapsed_lanes.iter().copied().collect()
    }

    pub fn collapsed_lane_keys(&self) -> Vec<String> {
        self.collapsed_lanes
            .iter()
            .map(|f| f.key().to_owned())
            .collect()
    }

    pub const fn issue_count(&self) -> usize {
        self.issues.len()
    }

    pub fn last_ok_label(&self) -> String {
        self.last_ok
            .map_or_else(|| "waiting".to_owned(), format_clock)
    }

    pub fn visible_rows(&self) -> Vec<IssueRow> {
        self.rows_for_filter(self.filter)
    }

    pub fn lanes(&self) -> Vec<KanbanLane> {
        ALL_LANES
            .iter()
            .map(|filter| KanbanLane {
                filter: *filter,
                label: filter.label(),
                cards: self.rows_for_filter(*filter),
            })
            .collect()
    }

    pub fn reset_for_repo(&mut self, repo: Option<&PathBuf>, repo_initialized: bool) {
        self.issues.clear();
        self.ready_ids.clear();
        self.detail = None;
        self.detail_id = None;
        self.detail_cache.clear();
        self.selected = 0;
        self.error = None;
        self.comment_draft.clear();
        self.comment_posting = false;
        self.comment_error = None;
        self.flash_until.clear();
        self.previous_updated.clear();
        if repo_initialized {
            if let Some(r) = repo
                && let Some(hit) = crate::cache::load_cached_fast(r)
            {
                let mut seeded = HashMap::new();
                for issue in &hit.issues {
                    if let Some(updated) = issue.updated_at {
                        seeded.insert(issue.id.clone(), updated);
                    }
                }
                self.previous_updated = seeded;
                self.issues = hit.issues;
                self.ready_ids = hit.ready_ids;
                self.last_ok = Some(hit.fetched_at);
                self.loading = true;
                self.stale = hit.stale;
            } else {
                self.last_ok = None;
                self.loading = true;
                self.stale = false;
            }
        } else {
            self.last_ok = None;
            self.loading = false;
            self.stale = false;
        }
    }

    pub fn clear_all(&mut self) {
        self.issues.clear();
        self.ready_ids.clear();
        self.detail = None;
        self.detail_id = None;
        self.detail_cache.clear();
        self.comment_draft.clear();
        self.comment_posting = false;
        self.comment_error = None;
        self.selected = 0;
        self.error = None;
        self.last_ok = None;
        self.loading = false;
        self.stale = false;
        self.flash_until.clear();
        self.previous_updated.clear();
    }

    pub fn open_detail(&mut self, id: &str, repo: Option<&PathBuf>) -> bool {
        self.detail_id = Some(id.to_string());
        self.comment_draft.clear();
        self.comment_posting = false;
        self.comment_error = None;
        if let Some(cached) = self.detail_cache.get(id) {
            self.detail = Some(cached.clone());
        } else if let Some(repo) = repo {
            if let Some(disk) = crate::cache::load_detail(repo, id) {
                self.detail_cache.insert(id.to_owned(), disk.clone());
                self.detail = Some(disk);
            } else if let Some(preview) = self.preview_detail(id) {
                self.detail = Some(preview);
            } else {
                self.detail = None;
            }
        } else if let Some(preview) = self.preview_detail(id) {
            self.detail = Some(preview);
        } else {
            self.detail = None;
        }
        true
    }

    pub fn close_detail(&mut self) {
        self.detail_id = None;
        self.detail = None;
        self.comment_draft.clear();
        self.comment_posting = false;
        self.comment_error = None;
    }

    pub fn move_selection(&mut self, delta: i32) {
        let count = self.visible_rows().len();
        if count == 0 {
            self.selected = 0;
            return;
        }
        let max = i32::try_from(count.saturating_sub(1)).unwrap_or(0);
        let current = i32::try_from(self.selected).unwrap_or(0);
        let shifted = current.checked_add(delta).unwrap_or(current);
        let next = shifted.clamp(0, max);
        let next = usize::try_from(next).unwrap_or(0);
        if next != self.selected {
            self.selected = next;
        }
    }

    pub fn apply_detail_loaded(&mut self, repo: &Path, detail: Box<IssueDetail>, current_repo: Option<&PathBuf>) {
        if current_repo.as_ref().map(|p| p.as_path()) != Some(repo) {
            return;
        }
        crate::cache::save_detail(repo, &detail);
        self.detail_cache
            .insert(detail.id.clone(), (*detail).clone());
        if self.detail_id.as_ref() == Some(&detail.id) {
            self.detail = Some(*detail);
            self.error = None;
            if self.comment_posting {
                self.comment_posting = false;
                self.comment_draft.clear();
                self.comment_error = None;
            }
        }
    }

    pub fn apply_failed(&mut self, repo: Option<&Path>, message: String, current_repo: Option<&PathBuf>) {
        if let Some(repo) = repo
            && current_repo.as_ref().map(|p| p.as_path()) != Some(repo)
        {
            return;
        }
        self.error = Some(message.clone());
        self.loading = false;
        if self.comment_posting {
            self.comment_posting = false;
            self.comment_error = Some(message);
        }
    }

    pub fn apply_tick(&mut self) -> bool {
        let now = Instant::now();
        let before = self.flash_until.len();
        self.flash_until.retain(|_, until| *until > now);
        self.flash_until.len() != before
    }

    pub fn apply_snapshot(&mut self, snapshot: Snapshot, current_repo: Option<&PathBuf>) {
        if current_repo.as_ref().map(|p| p.as_path()) != Some(snapshot.repo.as_path()) {
            return;
        }
        crate::cache::save_snapshot_async(snapshot.clone());
        let now = Instant::now();
        let flash_duration = Duration::from_millis(FLASH_MS);
        let mut next_updated = HashMap::new();

        for issue in &snapshot.issues {
            if let Some(updated) = issue.updated_at {
                if let Some(previous) = self.previous_updated.get(&issue.id)
                    && *previous != updated
                    && let Some(until) = now.checked_add(flash_duration)
                {
                    self.flash_until.insert(issue.id.clone(), until);
                }
                next_updated.insert(issue.id.clone(), updated);
            }
        }

        self.previous_updated = next_updated;
        self.issues = snapshot.issues;
        self.ready_ids = snapshot.ready_ids;
        self.last_ok = Some(snapshot.fetched_at);
        self.error = None;
        self.loading = false;
        self.stale = false;
    }

    fn preview_detail(&self, id: &str) -> Option<IssueDetail> {
        self.issues
            .iter()
            .find(|issue| issue.id == id)
            .map(|issue| IssueDetail {
                id: issue.id.clone(),
                title: issue.title.clone(),
                description: issue.description.clone(),
                acceptance_criteria: issue.acceptance_criteria.clone(),
                status: issue.status.clone(),
                priority: issue.priority,
                kind: issue.kind.clone(),
                assignee: issue.assignee.clone(),
                owner: None,
                parent: None,
                created_at: None,
                updated_at: issue.updated_at,
                started_at: None,
                closed_at: None,
                dependencies: Vec::new(),
                dependents: Vec::new(),
                dependency_count: 0,
                dependent_count: 0,
                comment_count: 0,
                comments: Vec::new(),
            })
    }

    fn rows_for_filter(&self, filter: Filter) -> Vec<IssueRow> {
        let now = Instant::now();
        self.issues
            .iter()
            .filter(|issue| matches_filter(issue, filter, &self.ready_ids))
            .filter(|issue| matches_search(issue, &self.search))
            .map(|issue| IssueRow {
                id: issue.id.clone(),
                title: issue.title.clone(),
                status: issue.status.clone(),
                priority: issue.priority,
                kind: issue.kind.clone(),
                flashed: self
                    .flash_until
                    .get(&issue.id)
                    .is_some_and(|until| *until > now),
            })
            .collect()
    }
}

#[cfg(test)]
mod tests;
