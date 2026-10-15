# fabrik issue create

## Purpose

Add one issue in status `open`. This is how landed work enters the queue after APPROVE. Planning does not call it.

## Who calls it

The planner agent, a human landing a task by hand, and bd-neon via its v2-write-board `+ New` control.

## Invocation

```text
fabrik [-C repo] issue create <title>
    [--body text]
    [--acceptance text]
    [--priority 0|1|2|3|4]
    [--kind task|bug|feature|epic|chore]
    [--json]
```

The title is one argument. Quoting is the caller's job.

## Inputs

| Input | Rule |
| --- | --- |
| `title` | Trimmed, not empty, at most 200 runes. |
| `--body` | Optional markdown. Default empty. |
| `--acceptance` | Optional markdown. Default empty. |
| `--priority` | Optional integer 0–4. Default 2. |
| `--kind` | Optional. Default `task`. |

## Stored rows

One new `issues` row with a hash id. Status `open`. Then `DOLT_COMMIT` with message `issue create <id>`, then [export.md](export.md). No deps and no notes.

`ready` is true on the returned object because a new issue has no blockers.

## JSON result

The new issue object.

```json
{
  "id": "f-u1v2w3x4",
  "title": "Document claim",
  "body": "",
  "acceptance": "Docs name the exit codes",
  "status": "open",
  "priority": 2,
  "kind": "task",
  "created_at": "2026-10-11T19:00:00Z",
  "updated_at": "2026-10-11T19:00:00Z",
  "started_at": null,
  "closed_at": null,
  "ready": true
}
```

## Text result

```text
f-u1v2w3x4  open  p2  task  Document claim
```

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Title missing or blank | 2 | `fabrik: usage: fabrik issue create <title>` |
| Title longer than 200 runes | 1 | `fabrik: title is too long` |
| Bad priority or kind | 2 | `fabrik: priority must be 0..4` or `fabrik: kind must be task, bug, feature, epic, or chore` |
| No store | 1 | `fabrik: no store (run fabrik init)` |

## Examples

```text
fabrik issue create "Document claim" --acceptance "Docs name the exit codes"
fabrik issue create "Fix parser hang" --kind bug --priority 1 --json
```

## bd-neon

`+ New` in the toolbar opens a create control with title, body, acceptance, priority (0-4), and kind (task|bug|feature|epic|chore). It runs `fabrik issue create` with `--json`; the new issue shows up after the worker refresh.
