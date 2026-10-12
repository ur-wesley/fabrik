# graph_detect_changes (MCP + CLI)

## Purpose

Map `git diff` (working tree or `HEAD`) onto symbols and execution-relevant files — GitNexus `detect_changes`. Answers “what should I retest after these edits?”

CLI: `fabrik graph detect_changes [--ref HEAD]`.

## Invocation

MCP `graph_detect_changes`:

| Argument | Rule |
| --- | --- |
| `ref` | Optional git ref for diff base. Default: unstaged + staged vs `HEAD`. |
| `project_path` | Optional. |

## Algorithm

1. `git diff` → changed line ranges per file.
2. Map lines to symbols via `graph_nodes` (innermost containing symbol).
3. For each touched symbol, run shallow `graph_impact` (depth 2).
4. Return union of touched + impacted symbols and file paths.

## Response

```json
{
  "changed_files": ["cli/internal/flow/flow.go"],
  "touched_symbols": ["sym:…:Loop"],
  "impacted_symbols": ["sym:…:Run"],
  "suggested_checks": ["cd cli && go test ./internal/flow/..."]
}
```

`suggested_checks` is heuristic from file extension (Go → `go test` package dir). Not run automatically.

Requires git repo. Requires graph index (refresh stale files first).

## bd-neon

Not in v1 UI.
