# fabrik issue show

## Purpose

Print one issue with its blockers, the issues it blocks, and its notes.

## Who calls it

Agents before editing. bd-neon when a card opens. Humans checking acceptance and why something is blocked.

## Invocation

```text
fabrik [-C repo] issue show <id> [--json]
```

`<id>` is required and is a single issue id (`f-k8m2n4p1`).

## Inputs

| Input | Rule |
| --- | --- |
| `id` | Must match `^f-[0-9a-f]{8,32}$`. |

## Stored rows

Read-only. Loads the issue, every `issue_deps` row where it is `blocked_id` or `blocker_id`, and every `issue_notes` row for it.

Blockers and dependents are ordered by id. Notes are ordered by `created_at`, then note id.

## JSON result

The issue object plus three arrays. Blocker and dependent objects are small issue objects: id, title, status, priority, kind. They do not repeat body or notes.

```json
{
  "id": "f-q7r8s9t0",
  "title": "Claim refuses a blocked issue",
  "body": "See acceptance.",
  "acceptance": "claim on a blocked id exits 1",
  "status": "blocked",
  "priority": 1,
  "kind": "bug",
  "created_at": "2026-10-11T17:10:00Z",
  "updated_at": "2026-10-11T18:00:00Z",
  "started_at": null,
  "closed_at": null,
  "ready": false,
  "blockers": [
    {
      "id": "f-k8m2n4p1",
      "title": "Add issue store",
      "status": "active",
      "priority": 2,
      "kind": "task"
    }
  ],
  "blocking": [],
  "notes": [
    {
      "id": "n-a1b2c3d4",
      "body": "Reproduced with f-q7r8s9t0 depending on f-k8m2n4p1",
      "created_at": "2026-10-11T18:00:00Z"
    }
  ]
}
```

`blockers` are issues that block this one. `blocking` are issues this one blocks.

## Text result

```text
f-q7r8s9t0  blocked  p1  bug
Claim refuses a blocked issue

See acceptance.

Acceptance:
claim on a blocked id exits 1

Blocked by:
f-k8m2n4p1  active  Add issue store

Blocks:
(none)

Notes:
n-a1b2c3d4  2026-10-11T18:00:00Z  Reproduced with f-q7r8s9t0 depending on f-k8m2n4p1
```

Empty body, acceptance, blockers, blocking, or notes print `(none)` for that section.

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Id missing or malformed | 2 | `fabrik: usage: fabrik issue show <id>` |
| Id not in the store | 1 | `fabrik: issue f-zzzzzzzz not found` |
| No store | 1 | `fabrik: no store (run fabrik init)` |

## Examples

```text
fabrik issue show f-q7r8s9t0
fabrik issue show f-q7r8s9t0 --json
```

## bd-neon

Opening a card runs `fabrik -C <repo> issue show <id> --json` and fills the detail modal: title, body, acceptance, blockers, dependents, notes, and the status sidebar. The modal does not offer create, claim, or close in v1. A note composer calls [issue-note.md](issue-note.md).
