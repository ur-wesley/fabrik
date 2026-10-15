# fabrik issue claim

## Purpose

Take one ready issue and mark it `active` so other builders can see it is being worked.

## Who calls it

The build agent, immediately after it picks the first row from [issue-ready.md](issue-ready.md). One claim per iteration. bd-neon does not claim in v1.

## Invocation

```text
fabrik [-C repo] issue claim [<id>] [--by actor] [--json]
```

Omit `id` to atomically claim the next ready issue (priority, then id). MCP: `issue_claim` with optional `id` and `actor` (default MCP client name).

## Inputs

| Input | Rule |
| --- | --- |
| `id` | One issue id. |
| `--by` | Optional assignee string, max 128 runes. Default `fabrik`. |

## Stored rows

The issue must be ready: status `open`, every `blocks` blocker `done`. Then:

- `status` becomes `active`
- `assignee` = `--by` / `actor`
- `started_at` becomes now if it was null
- `updated_at` becomes now

Then `DOLT_COMMIT` (`issue claim <id>`) and export. No note is added. A second claim is an error.

## JSON result

The updated issue object. `ready` is false. `started_at` is set.

```json
{
  "id": "f-m3n4o5p6",
  "title": "Wire the build loop",
  "body": "",
  "acceptance": "",
  "status": "active",
  "priority": 1,
  "kind": "task",
  "created_at": "2026-10-11T17:05:00Z",
  "updated_at": "2026-10-11T19:10:00Z",
  "started_at": "2026-10-11T19:10:00Z",
  "closed_at": null,
  "ready": false
}
```

## Text result

```text
f-m3n4o5p6  active  p1  task  Wire the build loop
```

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Unknown id | 1 | `fabrik: issue f-zzzzzzzz not found` |
| Status is `active` | 1 | `fabrik: issue f-m3n4o5p6 is active` |
| Status is `done` | 1 | `fabrik: issue f-m3n4o5p6 is done` |
| Status is `blocked`, or status is `open` but a blocker is not done | 1 | `fabrik: issue f-q7r8s9t0 is blocked by f-k8m2n4p1` |
| Bad usage | 2 | `fabrik: usage: fabrik issue claim <id>` |

The blocked message names the lowest blocker id that is not `done`.

## Examples

```text
fabrik issue claim f-m3n4o5p6
fabrik issue claim f-m3n4o5p6 --json
```

## bd-neon

No claim button. After an agent claims, the file watch refreshes the board and the card moves to the Active lane.
