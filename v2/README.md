# Fabrik v2

**Memory** on the machine (`~/.fabrik/memory.db`). **Issues** in embedded Dolt (`.fabrik/dolt/`). **Code graph** in SQLite (`.fabrik/graph.db`).

**Agents** use **Fabrik MCP** (`fabrik mcp`): `mem_*`, `issue_*`, and `graph_*`. **Humans** use bd-neon on CLI JSON.

Spec only — code follows these files.

## Shared

- [IDEA.md](IDEA.md) — vision, comparison, agent loop
- [GAPS.md](GAPS.md) — missing vs Beads / Engram / Graphify-GitNexus
- [mcp.md](mcp.md) — MCP server, registration, tool catalog
- [STACK.md](STACK.md) — Dolt (cgo), SQLite, gotreesitter
- [STORE.md](STORE.md) — three stores, ids, commits
- [REPO.md](REPO.md) — monorepo layout
- [MIGRATE.md](MIGRATE.md) — raw SQL migrations
- [SCHEMA.sql](SCHEMA.sql) — repo Dolt DDL (issues)
- [MEMORY.sql](MEMORY.sql) — machine memory DDL
- [GRAPH.sql](GRAPH.sql) — repo graph DDL
- [listen.md](listen.md) — loopback SSE for GUI
- [dolt.md](dolt.md) — VC and sync (issues only)

## Memory

- [mem-tools.md](mem-tools.md) — MCP memory tools (authoritative for agents)
- CLI mirrors: [mem-context.md](mem-context.md), [mem-search.md](mem-search.md), [mem-get.md](mem-get.md), [mem-save.md](mem-save.md), [mem-session-start.md](mem-session-start.md), [mem-session-summary.md](mem-session-summary.md), [mem-session-end.md](mem-session-end.md)

## Issues

- [issue-tools.md](issue-tools.md) — MCP issue tools
- CLI: [issue-list.md](issue-list.md) … [issue-note.md](issue-note.md)

## Code graph

- [graph-explore.md](graph-explore.md) — primary agent tool
- [graph-context.md](graph-context.md), [graph-impact.md](graph-impact.md), [graph-detect-changes.md](graph-detect-changes.md)
- [graph-status.md](graph-status.md)
- CLI: [graph-build.md](graph-build.md), [graph-query.md](graph-query.md), [graph-path.md](graph-path.md), [graph-explain.md](graph-explain.md)

## Version control and sync

- [vc-status.md](vc-status.md) … [export.md](export.md)

## Wiring

- [init.md](init.md), [setup.md](setup.md), [check.md](check.md), [reset.md](reset.md), [loop.md](loop.md), [hub-prompts.md](hub-prompts.md)

## Human UI

- [neon-board.md](neon-board.md), [neon-memory.md](neon-memory.md), [neon-lookup.md](neon-lookup.md)
