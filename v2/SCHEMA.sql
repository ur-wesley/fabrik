-- Fabrik v2 Dolt schema (fabrik database) — per repo under .fabrik/dolt/
-- Memory: ~/.fabrik/memory.db (MEMORY.sql), not synced via dolt push.

CREATE TABLE IF NOT EXISTS issues (
  id VARCHAR(32) PRIMARY KEY,
  title VARCHAR(512) NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  acceptance TEXT NOT NULL DEFAULT '',
  status VARCHAR(16) NOT NULL,
  priority TINYINT NOT NULL DEFAULT 2,
  kind VARCHAR(16) NOT NULL DEFAULT 'task',
  assignee VARCHAR(128),
  created_at VARCHAR(32) NOT NULL,
  updated_at VARCHAR(32) NOT NULL,
  started_at VARCHAR(32),
  closed_at VARCHAR(32)
);

CREATE TABLE IF NOT EXISTS issue_deps (
  blocker_id VARCHAR(32) NOT NULL,
  blocked_id VARCHAR(32) NOT NULL,
  kind VARCHAR(16) NOT NULL DEFAULT 'blocks',
  PRIMARY KEY (blocker_id, blocked_id, kind)
);

CREATE TABLE IF NOT EXISTS issue_notes (
  id VARCHAR(32) PRIMARY KEY,
  issue_id VARCHAR(32) NOT NULL,
  body TEXT NOT NULL,
  created_at VARCHAR(32) NOT NULL
);
