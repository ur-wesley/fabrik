# fabrik reset

## Purpose

Remove v2 data and app wiring. Keeps `.fabrik/config.yaml`, `docs/`, and `specs/`.

## Invocation

```text
fabrik reset [path] [--dry-run]
```

## Stored rows

Deletes:

- `.fabrik/dolt/` (entire directory)
- `.fabrik/graph.db` and WAL/SHM sidecars
- `.fabrik/export/`
- app shims and gitignore lines this init added

Does not delete `.beads/`.

## Text result

```text
deleted .fabrik/dolt
deleted .fabrik/graph.db
```

## bd-neon

Next refresh shows `fabrik init`.
