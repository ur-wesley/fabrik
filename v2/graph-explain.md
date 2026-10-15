# fabrik graph explain

## Purpose

Print one node and the edges that touch it. This is the drill-down after a query hit.

## Who calls it

Agents that already have a node id. bd-neon, on click.

## Invocation

```text
fabrik [-C repo] graph explain <id> [--json]
```

## Inputs

| Input | Rule |
| --- | --- |
| `id` | A `file:` or `sym:` node id. The whole id is one argument. |

## Stored rows

Read-only. The node, plus every edge where `src` or `dst` equals that id. Edges are ordered by kind (`contains`, `imports`, `calls`), then by the other end's id. Cap 100 edges. When more exist, the JSON includes `"truncated": true` and the first 100.

## JSON result

```json
{
  "node": {
    "id": "sym:cli/internal/flow/flow.go:Loop",
    "kind": "symbol",
    "name": "Loop",
    "path": "cli/internal/flow/flow.go",
    "line": 176,
    "lang": "go"
  },
  "edges": [
    {
      "src": "file:cli/internal/flow/flow.go",
      "dst": "sym:cli/internal/flow/flow.go:Loop",
      "kind": "contains",
      "label": ""
    }
  ],
  "truncated": false
}
```

## Text result

```text
sym:cli/internal/flow/flow.go:Loop  symbol  go  cli/internal/flow/flow.go:176
  contains <- file:cli/internal/flow/flow.go
  calls -> (unresolved) bdList
```

Same arrow rule as [graph-path.md](graph-path.md). When truncated, a last line `truncated`.

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Unknown id | 1 | `fabrik: node file:missing.go not found` |
| Bad usage | 2 | `fabrik: usage: fabrik graph explain <id>` |
| Empty index | 1 | `fabrik: graph is empty (run fabrik graph build)` |
| No store | 1 | `fabrik: no store (run fabrik init)` |

## Examples

```text
fabrik graph explain "sym:cli/internal/flow/flow.go:Loop"
fabrik graph explain "file:cli/internal/flow/flow.go" --json
```

## bd-neon

Clicking a node in Lookup runs this command and shows the node header plus the edge list. A file node is the hook for opening that path in the editor. v1 only displays the path. It does not launch an editor.
