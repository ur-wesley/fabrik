# fabrik graph path

## Purpose

Show one shortest chain of edges between two nodes. Used when the question is "how does A reach B", not "what is near this word".

## Who calls it

Agents tracing a connection. bd-neon, from two selected nodes in Lookup. Not the default search.

## Invocation

```text
fabrik [-C repo] graph path <from> <to> [--json]
```

`<from>` and `<to>` are node ids (`file:…` or `sym:…`), not free text. Look them up with [graph-query.md](graph-query.md) or [graph-explain.md](graph-explain.md) first.

## Inputs

| Input | Rule |
| --- | --- |
| `from`, `to` | Must exist in `graph_nodes`. |

## Stored rows

Read-only. Undirected BFS from `from` until `to` or the walk exceeds 20 hops. Direction is reported per edge. If several shortest paths exist, the walk prefers the edge whose `dst` id is lexicographically smaller, so the result is stable.

Unresolved edges (`dst` empty) are not walked, because they have no target node.

## JSON result

```json
{
  "nodes": [],
  "edges": []
}
```

`nodes` is the path in order, including both ends. `edges` has one less entry than `nodes`, in the same order. Each edge includes `src`, `dst`, `kind`, and `label`.

The same shape as query. A disconnected pair returns empty arrays and exit 0.

When `from` equals `to`, `nodes` has that one node and `edges` is empty.

## Text result

```text
sym:cli/cmd/fabrik/main.go:main
  contains <- file:cli/cmd/fabrik/main.go
file:cli/cmd/fabrik/main.go
  imports -> file:cli/internal/flow/flow.go
file:cli/internal/flow/flow.go
```

The arrow shows stored direction. `->` means the edge's `src` is the upper node. `<-` means the upper node is the edge's `dst`.

No path:

```text
no path
```

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Unknown node | 1 | `fabrik: node sym:missing:Nope not found` |
| Bad usage | 2 | `fabrik: usage: fabrik graph path <from> <to>` |
| Empty index | 1 | `fabrik: graph is empty (run fabrik graph build)` |
| No store | 1 | `fabrik: no store (run fabrik init)` |

## Examples

```text
fabrik graph path "sym:cli/cmd/fabrik/main.go:main" "sym:cli/internal/flow/flow.go:Loop"
fabrik graph path "file:cli/cmd/fabrik/main.go" "file:cli/internal/flow/flow.go" --json
```

## bd-neon

Lookup has no second free-text box. A person selects a node, then another, and the pane asks for the path. The command's JSON replaces the neighborhood until the search box is used again.
