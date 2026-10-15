# fabrik issue ready

## Purpose

Print issues a builder can start now: stored status `open`, and every blocker already `done`.

## Who calls it

The build agent, at the start of an iteration. `fabrik loop build`, in-process, with the same rule. bd-neon, to mark the Ready lane. Humans choosing the next task.

## Invocation

```text
fabrik [-C repo] issue ready [--json]
```

No limit flag. The set is the whole ready queue.

## Inputs

None.

## Stored rows

Read-only. Same order as [issue-list.md](issue-list.md): priority ascending, then id.

An issue is ready when:

- `status` is `open`, and
- no row in `issue_deps` has this issue as `blocked_id` and a `blocker_id` whose status is not `done`.

`active`, `blocked`, and `done` issues are left out even if their blockers are finished. A blocked issue returns to `open` only through the rule in [issue-dep.md](issue-dep.md) and [issue-close.md](issue-close.md).

## JSON result

A JSON array of the same issue objects as list. Every object has `"ready": true`.

```json
[
  {
    "id": "f-m3n4o5p6",
    "title": "Wire the build loop",
    "body": "",
    "acceptance": "",
    "status": "open",
    "priority": 1,
    "kind": "task",
    "created_at": "2026-10-11T17:05:00Z",
    "updated_at": "2026-10-11T17:05:00Z",
    "started_at": null,
    "closed_at": null,
    "ready": true
  }
]
```

## Text result

Same columns as list. Status is always `open`.

```text
f-m3n4o5p6  open  p1  task  Wire the build loop
```

When nothing is ready, text mode prints no lines and exits 0. JSON mode prints `[]`.

## Errors

Same store errors as [issue-list.md](issue-list.md).

## Examples

```text
fabrik issue ready
fabrik issue ready --json
```

## bd-neon

Refresh also runs `fabrik -C <repo> issue ready --json` and keeps the returned ids. The Ready lane shows those issues. The Open lane shows every issue whose stored status is `open`, which is the same set. Both lanes are populated on purpose so a person can look at "what can start" and "what is not started" without a mode switch.
