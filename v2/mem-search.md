# fabrik mem search

## Purpose

Full-text search over observation titles and bodies in **machine** memory.

## Who calls it

Agents when `mem context` did not answer. bd-neon search box. Humans hunting a past decision.

## Invocation

```text
fabrik mem search <query> [--limit n] [--project path] [--json]
```

## Inputs

| Input | Rule |
| --- | --- |
| `query` | Trimmed, not empty, at most 200 runes. Prefix query per word (`word*`). No `"` in query. |
| `--limit` | 1–100, default 20. |
| `--project` | Optional filter on `observations.project`. |

FTS: `observations_fts` in `memory.db` (`title`, `body`).

## Stored rows

Read-only. Rank bm25; tie-break newer `created_at`. MCP `mem_search` matches.

## JSON result

Array of observation objects (includes `project`).

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Empty query | 2 | usage |
| Unknown `--project` | 1 | `fabrik: project not found in memory` |

No `fabrik init` required.

## Examples

```text
fabrik mem search "issue ids"
fabrik mem search "claim" --project /src/app --limit 5 --json
```

## bd-neon

`fabrik mem search <query> --json` (optional `-C` for default project filter in UI settings only).
