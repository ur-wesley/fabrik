# graph_status (MCP + CLI)

## Purpose

Index health: file count, symbol count, stale files (mtime mismatch), last build time.

CLI: `fabrik graph status [--json]`.

## Response

```json
{
  "indexed": true,
  "files": 42,
  "symbols": 400,
  "stale_files": 2,
  "last_build_at": "2026-10-11T20:00:00Z"
}
```

When `graph_files` is empty, `indexed: false`.

MCP tool `graph_status`. Does not run a full build; lists stale paths only.
