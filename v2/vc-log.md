# fabrik vc log

## Purpose

List recent Dolt commits on the current branch.

## Who calls it

Humans auditing who changed the issue database. Agents rarely need it.

## Invocation

```text
fabrik [-C repo] vc log [--limit n] [--json]
```

Default `--limit` is 20. Max 200.

## Inputs

None.

## Stored rows

Read-only. `SELECT * FROM dolt_log LIMIT n`.

## JSON result

```json
{
  "commits": [
    {
      "hash": "abc123…",
      "date": "2026-10-11T20:00:00Z",
      "message": "issue close f-k8m2n4p1"
    }
  ]
}
```

## Text result

One line per commit: short hash, date, message.

## Errors

No store: same as [vc-status.md](vc-status.md).

## Examples

```text
fabrik vc log --limit 5
```

## bd-neon

Not shown in v1.
