# graph_context (MCP + CLI)

## Purpose

360° view of one symbol: definition snippet, callers, callees, file, related tests (symbols in `*_test.go` or `*.test.ts` that reference the name), and observations from **machine** `memory.db` where `symbol_id` matches **and** `project` equals this repo’s absolute path.

CLI: `fabrik graph context <symbol_id|name>`.

## Invocation

MCP `graph_context`:

| Argument | Rule |
| --- | --- |
| `symbol` | `sym:…` id or bare name (resolve with same rules as explore; ambiguous returns candidates). |
| `project_path` | Optional. |

## Response

```json
{
  "symbol": { "id": "sym:…:Loop", "name": "Loop", "path": "…", "line": 176 },
  "callers": [],
  "callees": [],
  "observations": [{ "id": "o-…", "title": "…", "kind": "decision" }],
  "tests": [{ "path": "…_test.go", "symbol": "TestLoop" }]
}
```

Refreshes stale files for the symbol’s path before read (same as explore).

## bd-neon

Detail pane when a graph node is selected; optional observations section.
