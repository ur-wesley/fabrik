# fabrik mem session end

## Purpose

Close the open **global** session.

## Invocation

```text
fabrik mem session end [--json]
```

## Stored rows

Set `ended_at` on open session in `memory.db`. SQLite commit. No Dolt export.

## Errors

No open session → exit 1.

## Examples

```text
fabrik mem session end
```

## bd-neon

Card shows `closed` on refresh.
