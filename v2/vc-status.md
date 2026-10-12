# fabrik vc status

## Purpose

Show the current Dolt branch, HEAD commit, and uncommitted working changes.

## Who calls it

Humans and agents before `vc commit` or sync. Not required after normal `issue` / `mem` writes (those auto-commit).

## Invocation

```text
fabrik [-C repo] vc status [--json]
```

## Inputs

None.

## Stored rows

Read-only. Queries `active_branch()`, `DOLT_HASHOF('HEAD')`, and `dolt_status`.

## JSON result

```json
{
  "branch": "main",
  "head": "abc123…",
  "dirty": false,
  "changes": []
}
```

When dirty, `changes` lists table names with staged/unstaged hints from `dolt_status`.

## Text result

```text
branch main
head abc123…
clean
```

or `dirty` plus one line per changed table.

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| No Dolt database | 1 | `fabrik: no store (run fabrik init)` |

## Examples

```text
fabrik vc status
fabrik vc status --json
```

## bd-neon

Not shown in v1.
