# Issue MCP tools

## Purpose

Beads-shaped work queue as MCP tools beside memory and graph. Same rules as CLI [issue-*.md](issue-list.md). Mutations commit to Dolt and run [export.md](export.md).

## Tools

### issue_ready

Returns JSON array of ready issues (`open`, all blockers `done`). Same as `fabrik issue ready --json`.

### issue_show

Argument: `id`. Full show payload with blockers, blocking, notes.

### issue_list

Optional `status` filter.

### issue_create

Arguments: `title`, optional `body`, `acceptance`, `priority`, `kind`.

### issue_claim

Arguments: `id`, optional `actor` (default: MCP client name from `clientInfo.name`, else `agent`).

Sets status `active`, stores `assignee` = actor on issue row.

### issue_close

Arguments: `id`, optional `reason`.

### issue_dep

Arguments: `blocked_id`, `blocker_id`, `kind` = `blocks` | `relates`.

- `blocks`: same as CLI [issue-dep.md](issue-dep.md).
- `relates`: records link only; does **not** affect `ready` or status.

### issue_note

Arguments: `issue_id`, `body`.

## Lock

If embedded Dolt is busy, all issue tools return MCP error `store busy`.
