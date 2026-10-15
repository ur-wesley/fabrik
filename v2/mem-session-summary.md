# fabrik mem session summary

## Purpose

Write the handoff onto the open global session and insert a searchable `handoff` observation.

## Invocation

```text
fabrik mem session summary [--goal …] [--discoveries …] [--accomplished …] [--next …] [--files …] [--json]
```

At least one flag required. `--files` newline-separated paths.

## Stored rows

Update open session + insert `handoff` observation with `project` from cwd repo when known. SQLite only. Does not set `ended_at`.

MCP `mem_session_summary` — same fields.

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| No open session | 1 | `fabrik: no open session` |

## Examples

```text
fabrik mem session summary --goal "Spec memory" --next "Implement MCP"
```

## bd-neon

Read-only session card; refresh updates.
