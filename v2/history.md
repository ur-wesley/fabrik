# fabrik history

## Purpose

Show how one issue or note changed across Dolt commits. Memory rows (`s-`, `o-`) are not in Dolt; use `fabrik mem get` and SQLite backups for machine memory.

## Who calls it

Humans debugging "who closed this" or "when did this title change". Agents rarely.

## Invocation

```text
fabrik [-C repo] history <id> [--json]
```

`<id>` must start with `f-` or `n-`.

## Inputs

The id must exist in the current branch history (at least one version).

## Stored rows

Read-only. `SELECT * FROM dolt_history_<table> WHERE id = ?` for the matching table, ordered by commit date.

## JSON result

```json
{
  "id": "f-k8m2n4p1",
  "table": "issues",
  "versions": [
    {
      "commit": "abc123…",
      "date": "2026-10-11T18:00:00Z",
      "status": "open",
      "title": "Add issue store"
    },
    {
      "commit": "def456…",
      "date": "2026-10-11T20:00:00Z",
      "status": "done",
      "title": "Add issue store"
    }
  ]
}
```

Field list per table matches [SCHEMA.sql](SCHEMA.sql).

## Text result

One block per version: commit, date, then changed fields.

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Unknown id | 1 | `fabrik: … not found` |
| Bad prefix | 2 | `fabrik: history expects f-, n-, s-, or o- id` |

## Examples

```text
fabrik history f-k8m2n4p1
fabrik history f-k8m2n4p1 --json
```

## bd-neon

Not shown in v1. Could link from issue detail later.
