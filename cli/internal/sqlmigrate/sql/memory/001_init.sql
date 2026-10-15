CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  filename TEXT NOT NULL,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
  id VARCHAR(32) PRIMARY KEY,
  started_at VARCHAR(32) NOT NULL,
  ended_at VARCHAR(32),
  goal TEXT NOT NULL DEFAULT '',
  discoveries TEXT NOT NULL DEFAULT '',
  accomplished TEXT NOT NULL DEFAULT '',
  next_steps TEXT NOT NULL DEFAULT '',
  files TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS observations (
  id VARCHAR(32) PRIMARY KEY,
  session_id VARCHAR(32),
  project VARCHAR(1024) NOT NULL DEFAULT '',
  kind VARCHAR(16) NOT NULL DEFAULT 'discovery',
  title VARCHAR(512) NOT NULL,
  body TEXT NOT NULL,
  topic_key VARCHAR(128),
  symbol_id VARCHAR(512),
  deleted_at VARCHAR(32),
  created_at VARCHAR(32) NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS observations_fts USING fts5(
  title,
  body,
  content='observations',
  content_rowid='rowid'
);
