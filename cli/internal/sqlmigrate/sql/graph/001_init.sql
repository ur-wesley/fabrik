CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  filename TEXT NOT NULL,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS graph_nodes (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  name TEXT NOT NULL,
  path TEXT NOT NULL DEFAULT '',
  line INTEGER NOT NULL DEFAULT 0,
  lang TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS graph_edges (
  src TEXT NOT NULL,
  dst TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  confidence TEXT NOT NULL DEFAULT 'extracted'
);

CREATE INDEX IF NOT EXISTS graph_edges_src ON graph_edges(src);
CREATE INDEX IF NOT EXISTS graph_edges_dst ON graph_edges(dst);

CREATE TABLE IF NOT EXISTS graph_files (
  path TEXT PRIMARY KEY,
  mtime TEXT NOT NULL,
  lang TEXT NOT NULL DEFAULT '',
  ok INTEGER NOT NULL DEFAULT 1
);
