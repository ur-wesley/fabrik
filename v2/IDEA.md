# Idea

Fabrik runs the agent loop: plan, land work, build, remember, map code. **Agents** use **Fabrik MCP** (`fabrik mcp`). **Humans** use bd-neon on CLI JSON.

## Three stores

| What | Where | Sync |
| --- | --- | --- |
| **Memory** (sessions, observations) | `~/.fabrik/memory.db` on the **machine** | Local only. Not `dolt push`. |
| **Issues** | `.fabrik/dolt/` per **repo** | `fabrik dolt push` / `pull` |
| **Code graph** | `.fabrik/graph.db` per **repo** | Rebuild locally |

Opening another repo does **not** open another memory database. `mem_context` with no `project` shows recent work **across all repos on this machine.**

## Who uses it

| Consumer | Interface |
| --- | --- |
| Cursor, OpenCode, Pi, Antigravity | MCP ([mcp.md](mcp.md), [mem-tools.md](mem-tools.md)) |
| bd-neon | `fabrik issue` / `fabrik mem` CLI |
| `fabrik loop` | In-process repo Dolt |

## Compared with Beads, Engram, Graphify

| Need | Old | v2 |
| --- | --- | --- |
| Todos | `bd` | Repo Dolt + `issue_*` tools |
| Memory | Engram `~/.engram` | `~/.fabrik/memory.db`, same `mem_*` names |
| Code map | Graphify / CodeGraph / GitNexus | `graph_explore`, `graph_impact`, … |

**Together:** `graph_explore(issue_id=…)`; `mem_save(symbol_id=…)` with `project` label; `graph_context` shows matching memories for that repo.

**Not copied:** see [GAPS.md](GAPS.md) (full inventory vs Beads, Engram, Graphify / GitNexus).

## Agent session loop

1. Optional `fabrik dolt pull` in repo.
2. `mem_current_project` → `mem_context` (machine-wide) → `mem_search` if needed.
3. `graph_explore` in repo (auto-refresh index).
4. `issue_ready` → `claim` → `close`.
5. `mem_save` / `mem_session_summary` (machine store).
6. `fabrik dolt push` for **issues only**.

Planning uses `.fabrik/specs/` until APPROVE.

## Concurrency

Repo Dolt: one writer, `store busy`. Machine SQLite: WAL; single-writer pragmas for MCP + CLI.
