# fabrik vc commit

## Purpose

Create a Dolt commit for any remaining working-set changes. Normal `issue` and `mem` commands commit themselves; this is for manual checkpoints or tooling that wrote SQL without committing.

## Who calls it

Humans. Scripts that use raw SQL against Dolt.

## Invocation

```text
fabrik [-C repo] vc commit -m <message> [--json]
```

## Inputs

| Input | Rule |
| --- | --- |
| `-m` | Required. Trimmed, not empty, at most 500 runes. |

## Stored rows

`CALL DOLT_COMMIT('-m', message)`. Fails when the working set is clean.

## JSON result

```json
{
  "hash": "abc123…",
  "message": "checkpoint before experiment"
}
```

## Text result

```text
committed abc123…
```

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Clean working set | 1 | `fabrik: nothing to commit` |
| Missing `-m` | 2 | usage |

## Examples

```text
fabrik vc commit -m "checkpoint before experiment"
```

## bd-neon

Not shown in v1.
