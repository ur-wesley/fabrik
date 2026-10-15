# fabrik dolt push

## Purpose

Push local Dolt commits to the configured remote. Issue and memory data move on `refs/dolt/data`, not on the code branch.

## Who calls it

Humans and agents at session end. CI after bots land issues. Not automatic on every write.

## Invocation

```text
fabrik [-C repo] dolt push [--json]
```

## Inputs

Requires a remote registered (see [init.md](init.md)). Uses lease-based push with retry on contention, same family as Dolt git remotes.

## Stored rows

Read-only on Fabrik tables. Updates remote ref.

## JSON result

```json
{
  "remote": "origin",
  "pushed": true,
  "commits": 3
}
```

When nothing to push: `"pushed": false`, `"commits": 0`, exit 0.

## Text result

```text
pushed 3 commits to origin
```

or `nothing to push`.

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| No remote | 1 | `fabrik: no dolt remote (fabrik dolt remote add origin <url>)` |
| Auth or network failure | 1 | `fabrik: dolt push: …` |
| Unmerged local state | 1 | `fabrik: dolt push: uncommitted changes` (working set dirty) |

Commit or stash via `vc commit` before push when dirty.

## Examples

```text
fabrik dolt pull
fabrik issue create "Ship sync docs"
fabrik dolt push
```

## bd-neon

Footer may show `sync?` when local HEAD is ahead of last known remote. v1 does not push from the UI.
