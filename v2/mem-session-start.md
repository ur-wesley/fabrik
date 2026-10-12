# fabrik mem session start

## Purpose

Open a **machine-wide** session row so later saves and the summary attach to it.

## Invocation

```text
fabrik mem session start [--json]
```

## Stored rows

If any session has `ended_at` null, return it (no write).

Else insert session in `memory.db`, SQLite commit. **No** Dolt.

## JSON result

Session object + `created` boolean.

## Errors

None for missing memory file (created on write).

## Examples

```text
fabrik mem session start --json
```

## bd-neon

Session card from [mem-context.md](mem-context.md).
