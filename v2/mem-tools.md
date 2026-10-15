# Memory MCP tools

## Purpose

Machine-wide memory: one database for the whole machine (`~/.fabrik/memory.db`), not per repo. Switching git repos does not switch memory.

Storage is SQLite + FTS5 in that file ([MEMORY.sql](MEMORY.sql)). Issues and code graph stay in the repo ([STORE.md](STORE.md)).

## Protocol (agents)

1. `mem_current_project` — where cwd points (never errors).
2. `mem_context` — MCP default: **full bodies for cwd repo**; other projects as title-only rows. Pass `project` to scope. `mem_search` stays all-projects.
3. `mem_search` — **default: all projects**; pass `project` only to narrow.
4. `mem_get_observation` — full body.
5. `mem_save` / `mem_update` — writes label `project` = detected repo path when cwd is inside a Fabrik repo.
6. `mem_session_start` → work → `mem_session_summary` → `mem_session_end`.

One open session per MCP `actor` (client name), not per repo.

## Tools

### mem_current_project

`{ "project": "<abs path or name>", "source": "cwd|env|none" }`. Never errors. Does not filter subsequent calls unless they pass `project`.

### mem_context

Arguments: optional `project`, optional `limit` (default 20), optional `scope` (`project` | `personal` — v1: both read same table; `personal` reserved).

Returns:

- Latest session row (any project) or the open global session.
- Newest `limit` observations **across all projects** when `project` omitted.
- Footer stats: session count, observation count, **list of distinct `project` labels**.

### mem_search

Arguments: `query`, optional `limit`, optional `project`, optional `all_projects` (default true when `project` omitted).

FTS5 on `memory.db`. No Dolt. No repo `graph.db`.

### mem_get_observation

Argument: `id` (`o-…`).

### mem_save

Arguments: `title`, `content`, `kind`, optional `topic_key`, optional `symbol_id`, optional `capture_prompt`.

Sets `project` from cwd repo root (walk up to `.fabrik/dolt`) or empty string. SQLite insert + FTS sync. **No** `fabrik dolt push`.

### mem_update / mem_delete

Update or soft-delete in `memory.db`. FTS sync.

### mem_timeline

Chronological neighbours around an observation id (machine-wide unless `project` set).

### mem_session_start / mem_session_summary / mem_session_end

Global session. Summary also inserts searchable `handoff` observation with current `project` label.

### mem_save_prompt

Prompt observation linked to open session + `project` label.

### mem_stats

Machine totals + project list. No Dolt commit hash.

### mem_suggest_topic_key

Suggests a stable `topic_key` for an evolving decision thread.

## graph_context linkage

`graph_context` in the **repo** loads observations where `symbol_id` matches **and** `project` equals the open repo’s absolute path.

## Not implemented

`mem_judge`, `mem_compare`, `mem_merge_projects`, Obsidian, `mem_doctor` (optional later).

## CLI mirror

`fabrik mem …` uses the same `memory.db`. Optional `--project` on read tools. Writes always stamp `project` from `-C` / cwd when detectable.
