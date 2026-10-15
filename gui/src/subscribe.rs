use std::io::{BufRead, BufReader, Write};
use std::net::TcpStream;
use std::path::{Path, PathBuf};
use std::sync::mpsc::{Receiver, Sender, channel};
use std::thread;
use std::time::Duration;

use serde::Deserialize;

use crate::fabrik::fabrik_bin;
use crate::model::Action;

#[derive(Debug, Clone, Deserialize)]
pub struct ListenConfig {
    pub url: String,
    pub token: String,
    #[serde(default)]
    pub pid: Option<u32>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ListenEvent {
    pub topic: String,
    #[serde(default)]
    pub repo: String,
    #[serde(default)]
    pub at: String,
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub op: String,
}

/// Exact port of the CLI `store.ListenJSONPath`: `FABRIK_MEMORY_DIR` wins,
/// otherwise `~/.fabrik/listen.json`. There is no `.fabrik/listen.json`
/// relative fallback — without a home dir there is no config path (`None`).
pub fn listen_json_path() -> Option<PathBuf> {
    if let Ok(dir) = std::env::var("FABRIK_MEMORY_DIR")
        && !dir.trim().is_empty()
    {
        return Some(PathBuf::from(dir).join("listen.json"));
    }
    let home = std::env::var("HOME")
        .or_else(|_| std::env::var("USERPROFILE"))
        .ok()
        .filter(|home| !home.trim().is_empty())?;
    Some(PathBuf::from(home).join(".fabrik").join("listen.json"))
}

pub fn read_listen_config() -> Option<ListenConfig> {
    let path = listen_json_path()?;
    let bytes = std::fs::read(path).ok()?;
    serde_json::from_slice(&bytes).ok()
}

pub fn ensure_listen_config() -> Result<ListenConfig, String> {
    if let Some(cfg) = read_listen_config() {
        return Ok(cfg);
    }
    spawn_listen()?;
    for _ in 0..20 {
        thread::sleep(Duration::from_millis(100));
        if let Some(cfg) = read_listen_config() {
            return Ok(cfg);
        }
    }
    Err("fabrik listen did not write listen.json".to_owned())
}

fn spawn_listen() -> Result<(), String> {
    #[cfg(windows)]
    use std::os::windows::process::CommandExt;
    let mut command = std::process::Command::new(fabrik_bin());
    command
        .args(["listen"])
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped());
    #[cfg(windows)]
    command.creation_flags(0x0800_0000);
    let mut child = command.spawn().map_err(|e| e.to_string())?;
    let stdout = child
        .stdout
        .take()
        .ok_or_else(|| "listen stdout missing".to_owned())?;
    thread::spawn(move || {
        let reader = BufReader::new(stdout);
        let mut lines = reader.lines();
        let _ = lines.next();
        let _ = child.wait();
    });
    Ok(())
}

pub fn parse_sse_data_line(line: &str) -> Option<ListenEvent> {
    let payload = line.strip_prefix("data: ")?;
    serde_json::from_str(payload.trim()).ok()
}

fn parse_events_url(url: &str) -> Result<(String, u16, String), String> {
    let rest = url
        .strip_prefix("http://")
        .or_else(|| url.strip_prefix("https://"))
        .ok_or_else(|| "bad listen url".to_owned())?;
    let (host_port, path) = rest.split_once('/').map_or_else(
        || (rest.to_owned(), String::from("/events")),
        |(hp, p)| (hp.to_owned(), format!("/{p}")),
    );
    let (host, port) = host_port
        .rsplit_once(':')
        .ok_or_else(|| "bad listen host".to_owned())?;
    let port = port
        .parse::<u16>()
        .map_err(|_| "bad listen port".to_owned())?;
    Ok((host.to_owned(), port, path))
}

fn sse_connect(cfg: &ListenConfig) -> Result<TcpStream, String> {
    let (host, port, path) = parse_events_url(&cfg.url)?;
    let addr = format!("{host}:{port}");
    let mut stream = TcpStream::connect(&addr).map_err(|e| e.to_string())?;
    stream
        .set_read_timeout(Some(Duration::from_secs(30)))
        .map_err(|e| e.to_string())?;
    stream
        .write_all(build_sse_request(&host, port, &path, &cfg.token).as_bytes())
        .map_err(|e| e.to_string())?;
    Ok(stream)
}

fn build_sse_request(host: &str, port: u16, path: &str, token: &str) -> String {
    format!(
        "GET {path} HTTP/1.1\r\nHost: {host}:{port}\r\nAuthorization: Bearer {token}\r\nAccept: text/event-stream\r\nConnection: keep-alive\r\n\r\n",
    )
}

fn skip_http_headers(reader: &mut BufReader<TcpStream>) -> Result<(), String> {
    let mut line = String::new();
    while {
        line.clear();
        reader.read_line(&mut line).map_err(|e| e.to_string())?;
        !line.is_empty() && line != "\r\n" && line != "\n"
    } {}
    Ok(())
}

enum SubMsg {
    RepoChanged(Option<PathBuf>),
    LookupActive(bool),
}

pub struct SubscribeService {
    cmd_tx: Sender<SubMsg>,
    action_rx: Receiver<Action>,
}

impl SubscribeService {
    pub fn new() -> Self {
        let (cmd_tx, cmd_rx) = channel();
        let (action_tx, action_rx) = channel();
        thread::spawn(move || subscribe_loop(cmd_rx, action_tx));
        Self { cmd_tx, action_rx }
    }

    pub fn set_repo(&self, repo: Option<&Path>) {
        let _ = self
            .cmd_tx
            .send(SubMsg::RepoChanged(repo.map(Path::to_path_buf)));
    }

    pub fn set_lookup_active(&self, active: bool) {
        let _ = self.cmd_tx.send(SubMsg::LookupActive(active));
    }

    pub fn poll(&self) -> Vec<Action> {
        let mut out = Vec::new();
        while let Ok(a) = self.action_rx.try_recv() {
            out.push(a);
        }
        out
    }
}

impl Default for SubscribeService {
    fn default() -> Self {
        Self::new()
    }
}

fn subscribe_loop(cmd_rx: Receiver<SubMsg>, action_tx: Sender<Action>) {
    let mut current_repo: Option<PathBuf> = None;
    let mut lookup_active = false;
    let mut backoff = Duration::from_millis(250);

    loop {
        let cfg = match ensure_listen_config() {
            Ok(c) => {
                let _ = action_tx.send(Action::ListenConnected(true));
                c
            }
            Err(err) => {
                let _ = action_tx.send(Action::ListenConnected(false));
                let _ = action_tx.send(Action::ListenUnavailable(err));
                thread::sleep(backoff);
                backoff = backoff.min(Duration::from_secs(5)) + Duration::from_millis(100);
                drain_cmds(&cmd_rx, &mut current_repo, &mut lookup_active);
                continue;
            }
        };

        match sse_connect(&cfg) {
            Ok(stream) => {
                backoff = Duration::from_millis(250);
                let mut reader = BufReader::new(stream);
                if skip_http_headers(&mut reader).is_err() {
                    thread::sleep(backoff);
                    continue;
                }
                let mut line = String::new();
                loop {
                    line.clear();
                    match reader.read_line(&mut line) {
                        Ok(0) => break,
                        Ok(_) => {}
                        Err(_) => break,
                    }
                    while let Ok(msg) = cmd_rx.try_recv() {
                        match msg {
                            SubMsg::RepoChanged(r) => current_repo = r,
                            SubMsg::LookupActive(a) => lookup_active = a,
                        }
                    }
                    if line.starts_with(':') {
                        continue;
                    }
                    if let Some(event) = parse_sse_data_line(&line) {
                        dispatch_event(&action_tx, &event, &current_repo, lookup_active);
                    }
                }
            }
            Err(_) => {
                let _ = action_tx.send(Action::ListenConnected(false));
            }
        }
        thread::sleep(backoff);
        backoff = backoff.min(Duration::from_secs(5)) + Duration::from_millis(100);
        drain_cmds(&cmd_rx, &mut current_repo, &mut lookup_active);
    }
}

fn drain_cmds(
    cmd_rx: &Receiver<SubMsg>,
    current_repo: &mut Option<PathBuf>,
    lookup_active: &mut bool,
) {
    while let Ok(msg) = cmd_rx.try_recv() {
        match msg {
            SubMsg::RepoChanged(r) => *current_repo = r,
            SubMsg::LookupActive(a) => *lookup_active = a,
        }
    }
}

fn dispatch_event(
    action_tx: &Sender<Action>,
    event: &ListenEvent,
    current_repo: &Option<PathBuf>,
    lookup_active: bool,
) {
    let id = if event.id.is_empty() {
        None
    } else {
        Some(event.id.clone())
    };
    let op = if event.op.is_empty() {
        None
    } else {
        Some(event.op.clone())
    };
    let live = Action::LiveEvent {
        topic: event.topic.clone(),
        id,
        op,
    };
    match event.topic.as_str() {
        "issues" => {
            if repo_matches(current_repo, &event.repo) {
                let _ = action_tx.send(live);
            }
        }
        "memory" => {
            let _ = action_tx.send(live);
        }
        "graph" => {
            if lookup_active && repo_matches(current_repo, &event.repo) {
                let _ = action_tx.send(live);
            }
        }
        _ => {}
    }
}

fn repo_matches(current: &Option<PathBuf>, event_repo: &str) -> bool {
    match current {
        None => false,
        Some(path) => {
            if event_repo.is_empty() {
                return false;
            }
            path.to_string_lossy() == event_repo
        }
    }
}

#[cfg(test)]
mod tests {
    use std::path::PathBuf;
    use std::sync::mpsc::channel;

    use crate::model::Action;

    use super::{
        ListenEvent, build_sse_request, dispatch_event, listen_json_path, parse_sse_data_line,
        repo_matches,
    };

    fn lock_env() -> std::sync::MutexGuard<'static, ()> {
        crate::cache::TEST_ENV_LOCK
            .lock()
            .unwrap_or_else(std::sync::PoisonError::into_inner)
    }

    struct EnvGuard {
        key: &'static str,
        prior: Option<String>,
    }

    impl EnvGuard {
        fn set(key: &'static str, value: Option<&str>) -> Self {
            let prior = std::env::var(key).ok();
            match value {
                Some(v) => unsafe { std::env::set_var(key, v) },
                None => unsafe { std::env::remove_var(key) },
            }
            Self { key, prior }
        }
    }

    impl Drop for EnvGuard {
        fn drop(&mut self) {
            match self.prior.clone() {
                Some(v) => unsafe { std::env::set_var(self.key, v) },
                None => unsafe { std::env::remove_var(self.key) },
            }
        }
    }

    #[test]
    fn parse_data_event() {
        let ev = parse_sse_data_line(
            r#"data: {"topic":"issues","repo":"/tmp/r","at":"2026-10-11T18:00:00Z"}"#,
        )
        .expect("parse");
        assert_eq!(ev.topic, "issues");
        assert_eq!(ev.repo, "/tmp/r");
    }

    #[test]
    fn listen_path_prefers_memory_dir() {
        let _guard = lock_env();
        let _dir = EnvGuard::set("FABRIK_MEMORY_DIR", Some("/tmp/fabmem"));
        let _home = EnvGuard::set("HOME", Some("/tmp/home"));
        let _profile = EnvGuard::set("USERPROFILE", Some("/tmp/profile"));
        assert_eq!(
            listen_json_path(),
            Some(PathBuf::from("/tmp/fabmem").join("listen.json"))
        );
    }

    #[test]
    fn listen_path_falls_back_to_home_without_relative_fallback() {
        let _guard = lock_env();
        let _dir = EnvGuard::set("FABRIK_MEMORY_DIR", None);
        let _home = EnvGuard::set("HOME", Some("/tmp/home"));
        let _profile = EnvGuard::set("USERPROFILE", Some("/tmp/profile"));
        assert_eq!(
            listen_json_path(),
            Some(
                PathBuf::from("/tmp/home")
                    .join(".fabrik")
                    .join("listen.json")
            )
        );
    }

    #[test]
    fn listen_path_uses_userprofile_when_home_missing() {
        let _guard = lock_env();
        let _dir = EnvGuard::set("FABRIK_MEMORY_DIR", None);
        let _home = EnvGuard::set("HOME", None);
        let _profile = EnvGuard::set("USERPROFILE", Some("/tmp/profile"));
        assert_eq!(
            listen_json_path(),
            Some(
                PathBuf::from("/tmp/profile")
                    .join(".fabrik")
                    .join("listen.json")
            )
        );
    }

    #[test]
    fn listen_path_is_none_without_any_home() {
        let _guard = lock_env();
        let _dir = EnvGuard::set("FABRIK_MEMORY_DIR", None);
        let _home = EnvGuard::set("HOME", None);
        let _profile = EnvGuard::set("USERPROFILE", None);
        assert_eq!(listen_json_path(), None);
    }

    #[test]
    fn sse_request_carries_bearer_token() {
        let request = build_sse_request("127.0.0.1", 4311, "/events", "secret-token");
        assert!(request.starts_with("GET /events HTTP/1.1\r\n"));
        assert!(request.contains("Authorization: Bearer secret-token\r\n"));
        assert!(request.contains("Accept: text/event-stream\r\n"));
        assert!(request.ends_with("\r\n\r\n"));
    }

    #[test]
    fn repo_matches_only_exact_current_repo() {
        let current = Some(PathBuf::from("/tmp/r"));
        assert!(repo_matches(&current, "/tmp/r"));
        assert!(!repo_matches(&current, "/tmp/other"));
        assert!(!repo_matches(&current, ""));
        assert!(!repo_matches(&None, "/tmp/r"));
    }

    fn drain(action_rx: &std::sync::mpsc::Receiver<Action>) -> Vec<Action> {
        let mut out = Vec::new();
        while let Ok(action) = action_rx.try_recv() {
            out.push(action);
        }
        out
    }

    fn listen_event(topic: &str, repo: &str) -> ListenEvent {
        ListenEvent {
            topic: topic.to_owned(),
            repo: repo.to_owned(),
            at: String::new(),
            id: "f-a1b2c3d4".to_owned(),
            op: "update".to_owned(),
        }
    }

    #[test]
    fn dispatch_routes_only_known_topics() {
        let current = Some(PathBuf::from("/tmp/r"));
        for (topic, lookup_active, expected) in [
            ("issues", false, 1),
            ("memory", false, 1),
            ("graph", true, 1),
            ("graph", false, 0),
            ("unknown", false, 0),
        ] {
            let (action_tx, action_rx) = channel();
            dispatch_event(
                &action_tx,
                &listen_event(topic, "/tmp/r"),
                &current,
                lookup_active,
            );
            assert_eq!(
                drain(&action_rx).len(),
                expected,
                "topic {topic} lookup_active={lookup_active}"
            );
        }
    }

    #[test]
    fn dispatch_issues_ignores_other_repos() {
        let (action_tx, action_rx) = channel();
        dispatch_event(
            &action_tx,
            &listen_event("issues", "/tmp/other"),
            &Some(PathBuf::from("/tmp/r")),
            false,
        );
        assert!(drain(&action_rx).is_empty());
    }
}
