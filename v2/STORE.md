# Store

Three stores. Only the repo ones are per git checkout.

| Path | Scope | Engine | Holds |
| --- | --- | --- | --- |
| `~/.fabrik/memory.db` | **Machine** | SQLite + FTS5 | Sessions, observations ([MEMORY.sql](MEMORY.sql)) |
| `.fabrik/dolt/` | **Repo** | Embedded Dolt | Issues, deps, notes ([SCHEMA.sql](SCHEMA.sql)) |
| `.fabrik/graph.db` | **Repo** | SQLite | Code graph only |

Override memory directory with `FABRIK_MEMORY_DIR` (default `~/.fabrik`). Created on first `fabrik setup` or first `fabrik mem` / MCP mem tool.

`.fabrik/config.yaml`, `docs/`, `specs/` stay in the repo. [init.md](init.md) creates Dolt + `graph.db`. It does **not** create `memory.db` (machine store already exists or is created lazily).

`fabrik loop` / `fabrik run` need repo `.fabrik/config.yaml` + `.fabrik/dolt/`. **`fabrik mem` and MCP memory tools do not require a Fabrik-init repo** — only cwd for labeling.

Times are RFC3339 UTC text.

## Commits (repo Dolt only)

Mutating `fabrik issue` commands:

1. SQL transaction on Dolt.
2. `CALL DOLT_COMMIT('-m', message)`.
3. [export.md](export.md) for issues JSONL.

Mutating memory (`fabrik mem` / MCP `mem_*`): SQLite transaction on `memory.db`. **No Dolt commit.** No `dolt push` for memory.

[graph-build.md](graph-build.md) writes only repo `graph.db`.

## Ids

| Row | Store | Prefix |
| --- | --- | --- |
| Issue, note | Dolt | `f-`, `n-` |
| Session, observation | Machine SQLite | `s-`, `o-` |

Hash generation: same rule as before (SHA256 + nonce, 8+ hex chars after prefix).

## issues, issue_deps, issue_notes

Unchanged. See prior sections in this file.

## Memory (machine)

### sessions

One open session for the **whole machine** (`ended_at` null). `mem_session_start` returns it if already open.

### observations

| Column | Rule |
| --- | --- |
| `project` | Label only: absolute repo path when cwd had `.fabrik/dolt/`, else basename or empty. **Does not scope reads unless the tool passes `project`.** |
| `symbol_id` | Optional `sym:…`; meaningful with `project` for [graph-context.md](graph-context.md). |
| `topic_key`, `deleted_at` | As before. |

FTS lives **in `memory.db`** (`observations_fts`). Not in repo `graph.db`.

### mem_context / mem_search scope (machine-wide)

| Call | Default scope |
| --- | --- |
| `mem_context` | All projects, newest observations + latest session handoff across machine |
| `mem_search` | All projects |
| Either with `project` arg | Filter `observations.project` = that string (must exist in store or error with `available_projects`) |
| `all_projects: false` + cwd | Optional: restrict to cwd repo path label only when explicitly requested (not default) |

`mem_current_project` returns cwd repo detection. Never errors. **Does not change the default scope of the next read.**

MCP: [mem-tools.md](mem-tools.md). CLI: `mem-*.md` (optional `fabrik mem … --project path`).

## Graph (repo SQLite)

Code index only. No observation FTS here.

## JSON shape (issues)

Issue object unchanged (`assignee`, `ready` computed).

## Other tools

- **CLI:** `fabrik issue` needs repo; `fabrik mem` uses machine db.
- **SQL:** [SCHEMA.sql](SCHEMA.sql) (repo), [MEMORY.sql](MEMORY.sql) (machine).
- **JSONL:** [export.md](export.md) — issues from repo; observations export optional from machine.
