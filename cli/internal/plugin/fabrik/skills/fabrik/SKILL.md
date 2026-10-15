---
name: fabrik
description: Core Fabrik CLI workflow assistant. Use for initializing repositories, checking tool dependencies, inspecting hub content, running AFK loops, and managing agent configurations.
---

# Fabrik CLI Assistant

Fabrik is the cross-platform AI workflow for Cursor, OpenCode, Pi, Antigravity, and Gemini CLI.

## Session start

1. ADHD output: next action first, numbered steps, Done/Next at end.
2. Memory via Fabrik MCP: `mem_current_project`, then `mem_context` / `mem_search` when needed.
3. Code map: `graph_explore` when the repo is indexed (Graphify).
4. Work queue: `issue_ready` (or `fabrik issue ready`) for unblocked work.

## Commands

- `fabrik init [PATH] [--apps cursor,pi,antigravity,gemini,opencode]`: init `.fabrik/` hub + app shims.
- `fabrik setup [--apps ...]`: machine setup + repo init.
- `fabrik check`: dolt/graph store status (advisory tool probe).
- `fabrik show workflow|skill|agent|prompt ...`: hub content (single source of truth).
- `fabrik list <skills|agents|all>`: what `fabrik show` can print.
- `fabrik run [--auto -p "goals"]` / `fabrik loop <plan|build>`: AFK waves.
- `fabrik migrate --prune`: thin legacy dumps (keeps config.yaml, docs/, specs/).
- `fabrik version`: CLI version.

## Stack

| Layer | Tool |
|-------|------|
| Issues | Beads (`bd`) — dependency-aware tasks, atomic claim |
| Memory | Engram via Fabrik MCP (`mem_context`, `mem_search`, `mem_save`, `mem_session_summary`, `issue_ready`) |
| Code map | Graphify (`graphifyy`) — structural map, `graph_explore` |
| Pi MCP | `@piarium/pi-mcp-adapter` |

No `bun`/`uv` requirement. Jev (`noul`/`choice`/`score`/`evaluate`) is optional — decision gates only when configured.

## Plan, land, build

1. **Plan** — plan agent: explore, write `.fabrik/specs/<slug>.md`, discuss. No issues yet.
2. **APPROVE** — planner lands issues (if needed), then builder + tester only.
3. **REVIEW** — stay in plan; amend spec. No land, no build.
4. Backpressure: run project test/lint/build gates before closing an issue.
5. Memory: decisions via `mem_save`; handoff via `mem_session_summary`. Beads owns todos only.
