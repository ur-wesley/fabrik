# fabrik issue list

## Purpose

Print the issues in the repo. This is the full board, not the ready queue. Use [issue-ready.md](issue-ready.md) for work that can start.

## Who calls it

Agents skimming the backlog. bd-neon, on every board refresh. Humans in a terminal.

## Invocation

```text
fabrik [-C repo] issue list [--status open|active|blocked|done] [--json]
```

`--status` filters the stored status. Omit it to return every issue. There is no `--all`, `--flat`, `--limit`, or `--type`.

## Inputs

None beyond the flags. The database must already exist.

## Stored rows

Read-only. Order is `priority` ascending, then `id` ascending (lexicographic on the hash).

## JSON result

A JSON array of issue objects from [STORE.md](STORE.md). `ready` is true only when that issue would also appear in `issue ready`.

```json
[
  {
    "id": "f-k8m2n4p1",
    "title": "Add issue store",
    "body": "",
    "acceptance": "list and ready work",
    "status": "open",
    "priority": 2,
    "kind": "task",
    "created_at": "2026-10-11T17:00:00Z",
    "updated_at": "2026-10-11T17:00:00Z",
    "started_at": null,
    "closed_at": null,
    "ready": true
  }
]
```

An empty repo prints `[]`.

## Text result

One issue per line:

```text
f-k8m2n4p1  open     p2  task  Add issue store
f-q7r8s9t0  blocked  p1  bug   Fix claim when a blocker is open
```

Columns are id, status, priority, kind, title. Status is padded to 7 characters. A trailing summary line is not printed.

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| No Dolt database | 1 | `fabrik: no store (run fabrik init)` |
| `--status` not one of the four values | 2 | `fabrik: status must be open, active, blocked, or done` |
| Database busy or corrupt | 1 | `fabrik: store: …` |

## Examples

```text
fabrik issue list
fabrik issue list --status done --json
fabrik -C /src/app issue list --status active
```

## bd-neon

The board calls `fabrik -C <repo> issue list --json` for every card. It does not read the database file. Lane placement is in [neon-board.md](neon-board.md). A filesystem change under `.fabrik/` triggers the same call again.
