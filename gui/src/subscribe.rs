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

pub fn listen_json_path() -> PathBuf {
    if let Ok(dir) = std::env::var("FABRIK_MEMORY_DIR") {
        return PathBuf::from(dir).join("listen.json");
    }
    if let Ok(home) = std::env::var("HOME") {
        return PathBuf::from(home).join(".fabrik").join("listen.json");
    }
    PathBuf::from(".fabrik/listen.json")
}

pub fn read_listen_config() -> Option<ListenConfig> {
    let path = listen_json_path();
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
    let mut child = std::process::Command::new(fabrik_bin())
        .args(["listen"])
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped())
        .spawn()
        .map_err(|e| e.to_string())?;
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
    let port = port.parse::<u16>().map_err(|_| "bad listen port".to_owned())?;
    Ok((host.to_owned(), port, path))
}

fn sse_connect(cfg: &ListenConfig) -> Result<TcpStream, String> {
    let (host, port, path) = parse_events_url(&cfg.url)?;
    let addr = format!("{host}:{port}");
    let mut stream = TcpStream::connect(&addr).map_err(|e| e.to_string())?;
    stream
        .set_read_timeout(Some(Duration::from_secs(30)))
        .map_err(|e| e.to_string())?;
    let request = format!(
        "GET {path} HTTP/1.1\r\nHost: {host}:{port}\r\nAuthorization: Bearer {token}\r\nAccept: text/event-stream\r\nConnection: keep-alive\r\n\r\n",
        path = path,
        host = host,
        port = port,
        token = cfg.token
    );
    stream
        .write_all(request.as_bytes())
        .map_err(|e| e.to_string())?;
    Ok(stream)
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
        Self {
            cmd_tx,
            action_rx,
        }
    }

    pub fn set_repo(&self, repo: Option<&Path>) {
        let _ = self.cmd_tx.send(SubMsg::RepoChanged(repo.map(Path::to_path_buf)));
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
    use super::parse_sse_data_line;

    #[test]
    fn parse_data_event() {
        let ev = parse_sse_data_line(
            r#"data: {"topic":"issues","repo":"/tmp/r","at":"2026-10-11T18:00:00Z"}"#,
        )
        .expect("parse");
        assert_eq!(ev.topic, "issues");
        assert_eq!(ev.repo, "/tmp/r");
    }
}
