# fabrik listen

## Purpose

Loopback HTTP for local GUI refresh (SSE) and issue note/read helpers. Agents use MCP for mutations during work.

## Invocation

```text
fabrik listen [--addr 127.0.0.1:0] [--json]
```

Default bind `127.0.0.1:0` (ephemeral port). Remote bind is refused.

## Startup

Stdout prints one JSON line (unless `--json` only writes that line and exits — v1: process stays running):

```json
{ "url": "http://127.0.0.1:38421/events", "token": "…" }
```

Same object written to `~/.fabrik/listen.json`. Removed on clean exit.

Second instance: exit 1, `fabrik: listen already running`.

## HTTP

All routes: loopback client only, `Authorization: Bearer <token>` from `listen.json`. Wrong token: 401. Non-loopback: 403.

Base URL: `listen.json` `url` with `/events` stripped (same as `/notify`).

### `GET /events`

Response: `text/event-stream`. Each event:

```json
{ "topic": "issues", "repo": "/abs/path", "at": "2026-10-11T18:00:00Z", "id": "f-…", "op": "claim" }
```

`id` and `op` are set for issue writes when known (`claim`, `write`, `note`, `export`). Memory topic keeps `repo` as cwd label or `""`.

### `POST /note`

Append a note (same store row as `fabrik issue note`).

```text
POST /note
Content-Type: application/json

{"repo": "/abs/path/to/repo", "id": "f-m3n4o5p6", "body": "Looks good"}
```

200 body: note JSON (`id`, `issue_id`, `body`, `created_at`). Then export + SSE `issues` with `op: "note"`.

| Case | Status |
| --- | --- |
| Empty body | 400 |
| Body too long | 400 |
| Bad/missing `id` or `repo` | 400 |
| Unknown issue / no store | 404 |

### `GET /issue`

Read one issue including `notes` (same JSON as `fabrik issue show --json`).

```text
GET /issue?repo=/abs/path/to/repo&id=f-m3n4o5p6
```

200: issue object. 404 when missing. Read-only (no SSE).

### SSE topics

| topic | When | repo |
| --- | --- | --- |
| `issues` | Dolt issue write committed | absolute repo root |
| `memory` | memory.db write committed | `""` |
| `graph` | graph.db write committed | absolute repo root |

## Notify path

After successful mutating `fabrik issue`, `fabrik mem`, `fabrik graph build`, or MCP equivalents: read `listen.json`, POST JSON body to `{url without /events}/notify` or internal channel if same process. If listen is down or POST fails, **the write still succeeds**.

## bd-neon

Prefer SSE when `listen.json` exists. Fall back to mtime watch on `.fabrik/dolt/`, `graph.db`, `memory.db` when not.
