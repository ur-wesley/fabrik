# fabrik issue note

## Purpose

Append a comment to an issue. Notes are how a stuck agent records what it tried, and how a person replies on the card.

## Who calls it

Agents that stop without closing. bd-neon, from the detail composer. Close uses the same table via `--reason` and does not go through this command.

## Invocation

```text
fabrik [-C repo] issue note <id> <body> [--json]
```

`body` is one argument. It may contain newlines.

## Inputs

| Input | Rule |
| --- | --- |
| `id` | An existing issue. Any status, including `done`. |
| `body` | Trimmed, not empty, at most 8000 runes. |

## Stored rows

One new `issue_notes` row. Touch issue `updated_at`. `DOLT_COMMIT` (`issue note <note-id>`) and export. Status does not change.

## JSON result

The new note, not the whole issue.

```json
{
  "id": "n-i9j0k1l2",
  "issue_id": "f-m3n4o5p6",
  "body": "Parser fixture still fails on generics",
  "created_at": "2026-10-11T20:30:00Z"
}
```

## Text result

```text
n-i9j0k1l2  f-m3n4o5p6  2026-10-11T20:30:00Z
Parser fixture still fails on generics
```

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Unknown issue | 1 | `fabrik: issue f-zzzzzzzz not found` |
| Empty body | 2 | `fabrik: note is empty` |
| Body too long | 1 | `fabrik: note is too long` |
| Bad usage | 2 | `fabrik: usage: fabrik issue note <id> <body>` |

## Examples

```text
fabrik issue note f-m3n4o5p6 "Parser fixture still fails on generics"
fabrik -C /src/app issue note f-m3n4o5p6 "Looks good" --json
```

## bd-neon

When `~/.fabrik/listen.json` exists, post via listen (same bearer token as SSE):

```text
POST {base}/note
{"repo": "/abs/path", "id": "f-…", "body": "…"}
```

Then `GET {base}/issue?repo=…&id=…` to refresh the open detail. Fallback: `fabrik -C <repo> issue note <id> <body> --json` and `issue show`. Empty input never sends. No edit, no delete.
