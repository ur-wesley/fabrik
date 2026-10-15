# Issue notes on listen

Backend slice: post and read issue notes over the same loopback server as SSE.

## Client setup

Read `~/.fabrik/listen.json` (`url`, `token`). Base URL = `url` with `/events` removed.

## POST /note

Bearer token. JSON body: `repo` (absolute repo root), `id` (`f-…`), `body` (trimmed, 1–8000 runes).

Response 200: same JSON as `fabrik issue note --json`. Triggers export debounce and SSE `issues` with `op: "note"`.

## GET /issue

Bearer token. Query: `repo`, `id`.

Response 200: same JSON as `fabrik issue show --json` (includes `notes`).

## Auth

Same as [v2/listen.md](../../v2/listen.md): loopback only, Bearer required.

## Out of scope

GUI wiring, edit/delete, author field.
