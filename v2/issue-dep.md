# fabrik issue dep

## Purpose

Record that one issue blocks another. The blocked issue cannot be claimed until every blocker is `done`.

## Who calls it

The planner, when landing a chain of issues. Humans wiring a blocker by hand. bd-neon does not edit deps in v1.

## Invocation

```text
fabrik [-C repo] issue dep <blocked> <blocker> [--kind blocks|relates] [--json]
```

Read the arguments as "blocker blocks blocked". `fabrik issue dep f-q7r8s9t0 f-k8m2n4p1` means `f-k8m2n4p1` blocks `f-q7r8s9t0`.

There is no remove command in v1.

## Inputs

| Input | Rule |
| --- | --- |
| `blocked` | Issue that will wait. |
| `blocker` | Issue that must finish first. |
| `--kind` | `blocks` (default) or `relates`. |

Both ids must exist. They must differ. `blocks` edges must not create a cycle.

## Stored rows

Insert (`blocker_id`, `blocked_id`, `kind`). If that triple already exists, no-op.

For `kind=blocks` only: when the insert is new and the blocker is not `done` and the blocked issue's status is `open`, set that issue to `blocked`, touch `updated_at`, and leave `started_at` null.

`relates` never changes status and does not affect `issue ready`.

When the blocked issue is `active` or `done`, the dep is still stored and the status is left alone. Then `DOLT_COMMIT` (`issue dep <blocked> <blocker>`) and export.

## JSON result

The blocked issue as an issue object, after the status change.

```json
{
  "id": "f-q7r8s9t0",
  "title": "Claim refuses a blocked issue",
  "body": "",
  "acceptance": "",
  "status": "blocked",
  "priority": 1,
  "kind": "bug",
  "created_at": "2026-10-11T17:10:00Z",
  "updated_at": "2026-10-11T18:00:00Z",
  "started_at": null,
  "closed_at": null,
  "ready": false
}
```

## Text result

```text
f-k8m2n4p1 blocks f-q7r8s9t0
f-q7r8s9t0  blocked  p1  bug  Claim refuses a blocked issue
```

If the pair already existed:

```text
f-k8m2n4p1 already blocks f-q7r8s9t0
```

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Either id missing | 1 | `fabrik: issue f-zzzzzzzz not found` |
| Same id twice | 1 | `fabrik: an issue cannot block itself` |
| Cycle | 1 | `fabrik: dep would cycle at f-k8m2n4p1` |
| Bad usage | 2 | `fabrik: usage: fabrik issue dep <blocked> <blocker>` |

The cycle message names the id where the walk returned to a seen issue.

## Examples

```text
fabrik issue dep f-q7r8s9t0 f-k8m2n4p1
fabrik issue dep f-y5z6a7b8 f-q7r8s9t0 --json
```

## bd-neon

Deps show up inside [issue-show.md](issue-show.md) as blockers and dependents. The board lanes follow stored status, so a newly blocked issue leaves Ready and Open and appears under Blocked after the next refresh.
