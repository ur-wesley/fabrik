CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  filename TEXT NOT NULL,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS meta (
  name TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

INSERT INTO meta (name, value) VALUES ('schema_version', '1')
ON CONFLICT(name) DO UPDATE SET value = excluded.value;

CREATE TABLE IF NOT EXISTS issues (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  acceptance TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 2,
  kind TEXT NOT NULL DEFAULT 'task',
  assignee TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  started_at TEXT,
  closed_at TEXT
);

CREATE TABLE IF NOT EXISTS issue_deps (
  blocker_id TEXT NOT NULL,
  blocked_id TEXT NOT NULL,
  kind TEXT NOT NULL DEFAULT 'blocks',
  PRIMARY KEY (blocker_id, blocked_id, kind)
);

CREATE TABLE IF NOT EXISTS issue_notes (
  id TEXT PRIMARY KEY,
  issue_id TEXT NOT NULL,
  body TEXT NOT NULL,
  created_at TEXT NOT NULL
);
