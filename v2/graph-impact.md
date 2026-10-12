# graph_impact (MCP + CLI)

## Purpose

Blast radius: what breaks if this symbol changes — GitNexus `impact` / CodeGraph `codegraph_impact`. Walk **incoming** `calls` and `imports` edges up to depth N, grouped by depth.

CLI: `fabrik graph impact <symbol> [--depth 3]`.

## Invocation

MCP `graph_impact`:

| Argument | Default |
| --- | --- |
| `symbol` | required |
| `depth` | 3 |
| `project_path` | optional |

## Response

```json
{
  "root": "sym:…:Loop",
  "by_depth": {
    "1": [{ "id": "sym:…:Run", "confidence": "extracted" }],
    "2": []
  },
  "total": 1
}
```

`inferred` edges count but are labeled; agents should treat them as hints.

## bd-neon

Optional “impact” action on selected node in Lookup.
