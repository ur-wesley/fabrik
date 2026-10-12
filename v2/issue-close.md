# fabrik issue close

## Purpose

Finish an issue. Status becomes `done`. An optional reason is stored as a note. Dependents that were waiting only on this issue become `open` again.

## Who calls it

The build agent, after the repo's tests pass. Humans closing work from a terminal. bd-neon does not close in v1.

## Invocation

```text
fabrik [-C repo] issue close <id> [--reason text] [--json]
```

`--reason` is optional. When set, it is one argument and becomes a note body.

## Inputs

| Input | Rule |
| --- | --- |
| `id` | One issue id. |
| `--reason` | Optional. Trimmed. If present, must be non-empty and at most 4000 runes. |

## Stored rows

Allowed from `open`, `active`, or `blocked`. In one transaction:

- `status` = `done`
- `closed_at` = now
- `updated_at` = now
- if `--reason` is set, insert one `issue_notes` row
- for each issue this one blocks: if that dependent's status is `blocked` and every remaining blocker is now `done`, set the dependent to `open`, clear `started_at`, and touch `updated_at`

`active` dependents are left `active`. Already `done` dependents stay `done`. Then `DOLT_COMMIT` (`issue close <id>`) and export.

## JSON result

The closed issue object, including notes written by this call. Dependents are not inlined. `ready` is false.

```json
{
  "id": "f-k8m2n4p1",
  "title": "Add issue store",
  "body": "",
  "acceptance": "",
  "status": "done",
  "priority": 2,
  "kind": "task",
  "created_at": "2026-10-11T17:00:00Z",
  "updated_at": "2026-10-11T20:00:00Z",
  "started_at": "2026-10-11T18:00:00Z",
  "closed_at": "2026-10-11T20:00:00Z",
  "ready": false
}
```

JSON for close does not include the show-only arrays. Use [issue-show.md](issue-show.md) for blockers and notes.

## Text result

```text
f-k8m2n4p1  done  p2  task  Add issue store
```

When a reason was stored, a second line:

```text
note n-e5f6g7h8
```

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Unknown id | 1 | `fabrik: issue f-zzzzzzzz not found` |
| Already `done` | 1 | `fabrik: issue f-k8m2n4p1 is done` |
| Blank `--reason` | 2 | `fabrik: reason is empty` |
| Bad usage | 2 | `fabrik: usage: fabrik issue close <id>` |

## Examples

```text
fabrik issue close f-k8m2n4p1 --reason "Tests passed"
fabrik issue close f-u1v2w3x4 --json
```

## bd-neon

No close button. The next refresh shows the card in Done. A dependent that flipped back to `open` shows in Ready and Open.
