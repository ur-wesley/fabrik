# fabrik dolt pull

## Purpose

Fetch and merge remote Dolt commits into the local database. Run before creating or claiming issues on a shared repo.

## Who calls it

Humans on a new clone. Agents at session start when the hub says to sync.

## Invocation

```text
fabrik [-C repo] dolt pull [--json]
```

## Inputs

Uses the same remote as [dolt-push.md](dolt-push.md).

## Stored rows

Merges remote into local Dolt `main` (or current branch). Cell-level merge when two writers edited different fields. Conflicts print keys; resolve with `vc merge` strategy or manual SQL plus `vc commit`.

After a successful pull, [export.md](export.md) may run to refresh JSONL.

## JSON result

```json
{
  "remote": "origin",
  "pulled": true,
  "commits": 2,
  "conflicts": []
}
```

When already up to date: `"pulled": false`, `"commits": 0`.

## Text result

```text
pulled 2 commits from origin
```

or `already up to date`.

## Errors

No remote, network, or unresolvable merge: exit 1 with message. On merge abandon paths that need hard reset, the driver pin (Dolt 2.2.0) must support `DOLT_RESET('--hard')`.

## Examples

```text
fabrik dolt pull
fabrik issue ready
```

## bd-neon

Refresh after pull is manual (person runs pull in a terminal). File watch sees `.fabrik/dolt/` change and reloads the board.
