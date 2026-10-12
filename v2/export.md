# fabrik export

## Purpose

Write passive JSONL snapshots for tools that do not speak SQL.

## Who calls it

`fabrik` after each successful Dolt commit on **issue** writes. Humans can run it by hand. Memory writes do **not** trigger repo export.

## Invocation

```text
fabrik [-C repo] export [--json]
fabrik mem export [--json]
```

## Stored rows

**Repo export** — read-only from Dolt. Writes:

- `.fabrik/export/issues.jsonl` — issue shape from [STORE.md](STORE.md) without `ready`

`.fabrik/export/` is gitignored. Not pushed by `dolt push`.

**Machine export** — read-only from `memory.db`. Writes:

- `~/.fabrik/export/observations.jsonl` (or under `FABRIK_MEMORY_DIR`)

## JSON result (repo)

```json
{
  "issues": 12,
  "path_issues": ".fabrik/export/issues.jsonl"
}
```

## JSON result (mem export)

```json
{
  "observations": 8,
  "path_observations": "/Users/me/.fabrik/export/observations.jsonl"
}
```

## Text result

```text
exported 12 issues
```

```text
exported 8 observations
```

## Errors

Repo: no Dolt → `fabrik: no store (run fabrik init)`.

Mem: missing `memory.db` → creates schema on first `fabrik setup` or lazy open; empty export is valid.

## Examples

```text
fabrik export
fabrik mem export
```

## Other tools

Writers must not treat JSONL as source of truth. Issues: Dolt or `fabrik issue`. Memory: `memory.db` or `fabrik mem`.

## bd-neon

Does not read JSONL. It uses `fabrik issue … --json` and `fabrik mem … --json`.
