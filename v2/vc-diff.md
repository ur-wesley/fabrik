# fabrik vc diff

## Purpose

Show cell-level diffs for the working set or between two commits.

## Who calls it

Humans before push. Agents when debugging a merge.

## Invocation

```text
fabrik [-C repo] vc diff [--json]
fabrik [-C repo] vc diff <commit-a> <commit-b> [--json]
```

With no commits, diff is working set vs `HEAD`. With two commits, diff that range.

## Inputs

Commit hashes are full or abbreviated Dolt commit ids from `vc log`.

## Stored rows

Read-only. `dolt_diff` / `dolt_diff_stat` as appropriate.

## JSON result

```json
{
  "tables": [
    {
      "name": "issues",
      "rows_added": 1,
      "rows_removed": 0,
      "rows_modified": 0
    }
  ]
}
```

A `--verbose` flag is not in v1. Table stats only.

## Text result

```text
issues  +1 ~0 -0
```

## Errors

Unknown commit: `fabrik: unknown commit …`

## Examples

```text
fabrik vc diff
fabrik vc diff HEAD~3 HEAD
```

## bd-neon

Not shown in v1.
