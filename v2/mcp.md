# fabrik mcp

## Purpose

Stdio MCP server that exposes issues, Engram-compatible memory, and a CodeGraph-class code index. Agents in Cursor, OpenCode, Pi, and Antigravity call tools here. They do not shell out to `fabrik issue` / `fabrik mem` / `fabrik graph` during normal work.

Humans, bd-neon, and CI keep using the CLI.

## Invocation

```text
fabrik mcp
```

No flags in v1. Memory tools always work (open or create `~/.fabrik/memory.db`). Issue and graph tools walk upward from cwd until `.fabrik/dolt/` exists. If none, issue/graph tools error with `no store`; mem tools still register.

Transport: MCP over stdio (same model as `engram mcp`).

## Registration

[setup.md](setup.md) and [init.md](init.md) write this server into each selected app:

| App | Config location (conceptual) |
| --- | --- |
| Cursor | MCP servers in Cursor settings / project mcp.json |
| OpenCode | `opencode.json` mcp section |
| Pi | Pi MCP adapter entry pointing at `fabrik mcp` |
| Antigravity | Antigravity MCP config |

`fabrik setup` does **not** run `engram setup`, `bd setup`, or install Graphify skills for memory/graph.

Remove legacy Engram entries when Fabrik MCP is added (same pass as today’s `engram setup`).

## Server instructions

Short text shipped with the server (read every session):

1. `mem_current_project` optional; `mem_context` (machine-wide by default), then `mem_search` if needed.
2. Code questions: `graph_explore` first when `.fabrik/graph.db` exists; do not grep the whole repo first.
3. Work queue: `issue_ready` → `issue_show` → `issue_claim` → implement → `issue_close`.
4. Hand off with `mem_session_summary` and `mem_save` for durable decisions.
5. Shared repo: human or script runs `fabrik dolt pull` / `push`; MCP does not auto-sync.

## Default tool surface

Advertised in `tools/list` for agents (small menu):

| Tool | Family |
| --- | --- |
| `mem_current_project` | Memory |
| `mem_context` | Memory |
| `mem_search` | Memory |
| `mem_get_observation` | Memory |
| `mem_save` | Memory |
| `mem_session_summary` | Memory |
| `issue_ready` | Issues |
| `issue_show` | Issues |
| `issue_claim` | Issues |
| `issue_close` | Issues |
| `graph_explore` | Graph |

All other tools below are **registered** and documented in server instructions but omitted from the default list unless `FABRIK_MCP_TOOLS=all` or a comma-separated allowlist is set.

## Full tool catalog

### Memory (Engram names)

See [mem-tools.md](mem-tools.md). Implemented tools call the same logic as CLI `fabrik mem …` where a CLI exists.

### Issues (MCP wrappers)

| Tool | CLI equivalent |
| --- | --- |
| `issue_list` | `fabrik issue list` |
| `issue_ready` | `fabrik issue ready` |
| `issue_show` | `fabrik issue show` |
| `issue_create` | `fabrik issue create` |
| `issue_claim` | `fabrik issue claim` (+ `actor` from MCP client metadata) |
| `issue_close` | `fabrik issue close` |
| `issue_dep` | `fabrik issue dep` (`blocks` or `relates`) |
| `issue_note` | `fabrik issue note` |

### Graph

| Tool | Doc |
| --- | --- |
| `graph_explore` | [graph-explore.md](graph-explore.md) |
| `graph_context` | [graph-context.md](graph-context.md) |
| `graph_impact` | [graph-impact.md](graph-impact.md) |
| `graph_detect_changes` | [graph-detect-changes.md](graph-detect-changes.md) |
| `graph_path` | [graph-path.md](graph-path.md) |
| `graph_explain` | [graph-explain.md](graph-explain.md) |
| `graph_status` | Staleness + counts (like CodeGraph `codegraph_status`) |

Optional `project_path` on graph tools: resolve repo root like `-C` (monorepo subfolder).

## Errors

| Case | MCP result |
| --- | --- |
| Dolt lock busy | `isError: true`, message `store busy (another fabrik process holds the lock)` |
| No store | `isError: true`, message `no store (run fabrik init)` |
| Graph not indexed | `graph_explore` returns text guidance + `indexed: false`, not a transport error |

## bd-neon

bd-neon does not speak MCP in v1. It keeps using CLI JSON.
