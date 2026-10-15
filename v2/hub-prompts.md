# Hub prompt changes (implementation note)

When implementing v2 in [cli/internal/tmpl/workflow-note.md](../cli/internal/tmpl/workflow-note.md) and hub prompts:

| Replace | With |
| --- | --- |
| `mem_context`, `mem_search`, `mem_save`, `mem_session_summary` (MCP) | Keep names — now Fabrik MCP, not Engram binary |
| `bd ready`, `bd update --claim`, `bd close` | `issue_ready`, `issue_claim`, `issue_close` MCP tools (or `fabrik issue …` CLI) |
| Graphify / `graphify query` when `graphify-out/` exists | `graph_explore` MCP (auto-index) |
| `engram setup` in installer | `fabrik mcp` registration per [setup.md](setup.md) |
| `tools.beads`, `tools.engram`, `tools.graphify` in config | `tools.fabrik_mcp: true` (suggested) |

Session start order: `mem_current_project` → `mem_context` (all projects) → `graph_explore` when in a Fabrik repo → `issue_ready`. Memory is `~/.fabrik/memory.db`, not repo Dolt.

This file is not a runtime spec; it tracks template work for the Go CLI.
