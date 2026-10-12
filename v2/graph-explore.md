# graph_explore (MCP + CLI)

## Purpose

One call answers “how does X work”, “what calls Y”, or “what is near Z” — like CodeGraph `codegraph_explore`. Returns line-numbered source grouped by file, call paths between hits, blast-radius summary, and edge confidence (`extracted` | `inferred`).

Replaces ad-hoc `fabrik graph query` for agents. CLI alias: `fabrik graph explore <query>`.

## Who calls it

Agents via MCP (default tool). Humans via CLI.

## Invocation

MCP tool `graph_explore`:

| Argument | Rule |
| --- | --- |
| `query` | Natural language and/or symbol names, file paths. |
| `issue_id` | Optional `f-…`. Server reads title, body, acceptance, notes and adds symbols mentioned there to the seed set. |
| `project_path` | Optional repo subpath for monorepos. |
| `budget` | Max symbols, default 40. |

## Before answering

1. If `graph_files` is empty, return `{ "indexed": false, "hint": "run fabrik graph build in repo root" }` without error transport.
2. Else run incremental index for files where mtime ≠ `graph_files` (same rules as [graph-build.md](graph-build.md)).
3. Seed symbols from query token match + issue text.
4. Expand with BFS on `calls` / `imports` edges (undirected for neighborhood), prefer `extracted` edges.
5. Include dynamic-dispatch hop: when callee is unresolved (`inferred`), list candidate symbols and mark hop as `dynamic`.

## Response shape (JSON)

```json
{
  "indexed": true,
  "files": [
    {
      "path": "cli/internal/flow/flow.go",
      "snippets": [
        { "line_start": 176, "line_end": 210, "symbol": "Loop", "text": "…" }
      ]
    }
  ],
  "paths": [
    { "from": "sym:…:main", "to": "sym:…:Loop", "hops": ["extracted", "extracted"] }
  ],
  "blast_radius": {
    "callers": ["sym:…:Run"],
    "dependents_count": 3
  }
}
```

Text mode for CLI: same content, markdown sections.

## Resolution rules

- Import resolves to file in repo → `extracted` `imports` edge.
- Call targets one definition in resolved import → `extracted` `calls`.
- Multiple definitions → `inferred` edge with `candidates[]`.
- Unique global name in repo → `extracted` (fallback).

Languages: Go, Rust, TS, TSX, JS, JSX (TS grammar), Python, Java, Kotlin, C#.

## Errors

| Case | Behaviour |
| --- | --- |
| No `.fabrik/graph.db` | Treat as not indexed |
| Store missing | `fabrik: no store` |

## bd-neon

Lookup mode should call `fabrik graph explore` (CLI) with the search text.
