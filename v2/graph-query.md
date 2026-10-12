# fabrik graph query

## Purpose

Legacy CLI neighborhood search. **Agents should use MCP `graph_explore`** ([graph-explore.md](graph-explore.md)) instead.

## Who calls it

Scripts and bd-neon Lookup when not yet on `graph explore`. Refreshes stale files before query (same as explore).

## Invocation

```text
fabrik [-C repo] graph query <text> [--budget n] [--json]
```

## Inputs

| Input | Rule |
| --- | --- |
| `text` | One argument. Trimmed, not empty, at most 200 runes. Split on whitespace. Each word is a case-insensitive substring against node `name` and `path`. A node matches when every word hits. |
| `--budget` | Max nodes in the result, including seeds. Integer 1–200. Default 40. |

## Stored rows

Read-only.

1. If `graph.db has no graph_files rows, fail. The index has never been built.
2. Seed nodes are the matches, ordered by symbol before file, then shorter name, then path. At most `budget` seeds.
3. BFS walks edges as undirected from those seeds. Direction is kept on the edge object. Stop when the node set reaches `budget`.
4. Result edges are the walked edges whose both ends are in the result. Unresolved edges (`dst` empty) are included when their `src` is in the result, and they do not add a node.

## JSON result

```json
{
  "nodes": [
    {
      "id": "sym:cli/internal/flow/flow.go:Loop",
      "kind": "symbol",
      "name": "Loop",
      "path": "cli/internal/flow/flow.go",
      "line": 176,
      "lang": "go"
    }
  ],
  "edges": [
    {
      "src": "sym:cli/internal/flow/flow.go:Loop",
      "dst": "sym:cli/internal/flow/flow.go:bdList",
      "kind": "calls",
      "label": ""
    }
  ]
}
```

No matches: `{"nodes":[],"edges":[]}` and exit 0. That is different from a missing index.

## Text result

```text
sym:cli/internal/flow/flow.go:Loop  go  cli/internal/flow/flow.go:176
  calls -> sym:cli/internal/flow/flow.go:bdList
```

Unresolved:

```text
  calls -> (unresolved) bdList
```

No matches: `no matches`.

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Empty text | 2 | `fabrik: usage: fabrik graph query <text>` |
| `--budget` out of range | 2 | `fabrik: budget must be 1..200` |
| `graph_files` empty | 1 | `fabrik: graph is empty (run fabrik graph build)` |
| No store | 1 | `fabrik: no store (run fabrik init)` |

## Examples

```text
fabrik graph query "Loop"
fabrik graph query "issue claim" --budget 20 --json
```

## bd-neon

The Lookup field calls `fabrik -C <repo> graph query <text> --budget 40 --json`. Nodes are a list. Edges are lines under the selected node. Clicking a node calls [graph-explain.md](graph-explain.md). The empty-index error replaces the list with the build hint.
