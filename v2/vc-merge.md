# fabrik vc merge

## Purpose

Merge another Dolt branch into the current branch. Used after parallel work on Dolt branches, not after `dolt pull` (pull merges remote commits automatically when possible).

## Who calls it

Humans. Pull may report conflicts that need a local branch merge in edge cases.

## Invocation

```text
fabrik [-C repo] vc merge <branch> [--strategy ours|theirs] [--json]
```

## Inputs

| Input | Rule |
| --- | --- |
| `branch` | Source branch name. Must exist and differ from current. |
| `--strategy` | On cell conflict, keep current (`ours`) or incoming (`theirs`). Default `ours`. |

## Stored rows

`DOLT_MERGE` with conflict resolution per Dolt rules. On success, one merge commit.

## JSON result

```json
{
  "merged": true,
  "hash": "abc123…"
}
```

On conflict without resolution:

```json
{
  "merged": false,
  "conflicts": ["issues.f-k8m2n4p1.title"]
}
```

Exit 1 when `merged` is false.

## Text result

```text
merged abc123…
```

or a list of conflict keys.

## Errors

Merge into self: `fabrik: cannot merge branch into itself`

## Examples

```text
fabrik vc merge experiment --strategy theirs
```

## bd-neon

Not shown in v1.
