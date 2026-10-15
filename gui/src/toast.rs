use std::time::{Duration, Instant};

pub const MAX_TOASTS: usize = 3;
const INFO_TTL: Duration = Duration::from_secs(4);

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ToastKind {
    Info,
    Error,
}

#[derive(Debug, Clone)]
pub struct Toast {
    pub id: u64,
    pub message: String,
    pub kind: ToastKind,
    pub until: Option<Instant>,
}

#[derive(Debug, Clone, Default)]
pub struct ToastStore {
    items: Vec<Toast>,
    next_id: u64,
    pub last_listen_error: Option<String>,
}

impl ToastStore {
    pub fn items(&self) -> &[Toast] {
        &self.items
    }

    pub fn push_info(&mut self, message: String, now: Instant) {
        if let Some(existing) = self
            .items
            .iter_mut()
            .find(|t| t.kind == ToastKind::Info && t.message == message)
        {
            existing.until = now.checked_add(INFO_TTL);
            return;
        }
        let until = now.checked_add(INFO_TTL);
        let id = self.next_id();
        self.push(Toast {
            id,
            message,
            kind: ToastKind::Info,
            until,
        });
    }

    pub fn push_error(&mut self, message: String) {
        self.items.retain(|t| t.kind != ToastKind::Error);
        let id = self.next_id();
        self.push(Toast {
            id,
            message,
            kind: ToastKind::Error,
            until: None,
        });
    }

    pub fn clear_errors(&mut self) {
        self.items.retain(|t| t.kind != ToastKind::Error);
        self.last_listen_error = None;
    }

    pub fn dismiss(&mut self, id: u64) {
        self.items.retain(|t| t.id != id);
    }

    pub fn tick(&mut self, now: Instant) -> bool {
        let before = self.items.len();
        self.items.retain(|t| match t.until {
            Some(until) => until > now,
            None => true,
        });
        self.items.len() != before
    }

    fn push(&mut self, toast: Toast) {
        if toast.kind == ToastKind::Info {
            while self.items.len() >= MAX_TOASTS {
                if let Some(pos) = self.items.iter().position(|t| t.kind == ToastKind::Info) {
                    self.items.remove(pos);
                } else {
                    break;
                }
            }
        }
        self.items.push(toast);
        while self.items.len() > MAX_TOASTS {
            self.items.remove(0);
        }
    }

    fn next_id(&mut self) -> u64 {
        let id = self.next_id;
        self.next_id = self.next_id.saturating_add(1);
        id
    }
}

pub fn live_event_message(topic: &str, id: Option<&str>, op: Option<&str>) -> Option<String> {
    match topic {
        "issues" => {
            let msg = match (id.filter(|s| !s.is_empty()), op.filter(|s| !s.is_empty())) {
                (Some(id), Some("claim")) => format!("Issue {id} claimed"),
                (Some(id), Some("note")) => format!("Note on {id}"),
                (Some(id), _) => format!("Issue {id} updated"),
                _ => "Issues updated".to_owned(),
            };
            Some(msg)
        }
        "memory" => Some("Memory updated".to_owned()),
        "graph" => Some("Graph updated".to_owned()),
        _ => None,
    }
}

pub fn error_line(message: &str, max_chars: usize) -> String {
    if message.len() <= max_chars {
        return message.to_owned();
    }
    let mut end = max_chars;
    while end > 0 && !message.is_char_boundary(end) {
        end -= 1;
    }
    format!("{}…", &message[..end])
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn live_issue_claim_label() {
        assert_eq!(
            live_event_message("issues", Some("f-abc"), Some("claim")).as_deref(),
            Some("Issue f-abc claimed")
        );
    }

    #[test]
    fn duplicate_info_refreshes_not_stack() {
        let mut store = ToastStore::default();
        let now = Instant::now();
        store.push_info("Issues updated".to_owned(), now);
        assert_eq!(store.items().len(), 1);
        store.push_info("Issues updated".to_owned(), now);
        assert_eq!(store.items().len(), 1);
    }

    #[test]
    fn push_error_replaces_prior_error() {
        let mut store = ToastStore::default();
        store.push_error("a".to_owned());
        store.push_error("b".to_owned());
        assert_eq!(store.items().len(), 1);
        assert_eq!(store.items()[0].message, "b");
    }

    #[test]
    fn tick_drops_expired_info() {
        let mut store = ToastStore::default();
        let now = Instant::now();
        store.push_info("x".to_owned(), now);
        assert!(!store.tick(now));
        assert_eq!(store.items().len(), 1);
        assert!(store.tick(now + INFO_TTL + Duration::from_millis(1)));
        assert!(store.items().is_empty());
    }
}
