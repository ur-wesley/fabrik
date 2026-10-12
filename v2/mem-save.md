# fabrik mem save

## Purpose

Store one durable observation: decision, bugfix, pattern, discovery, or handoff.

## Invocation

```text
fabrik mem save --title text --body text [--kind …] [--json]
fabrik [-C repo] mem save …
```

## Inputs

| Input | Rule |
| --- | --- |
| `--title` | Required, ≤200 runes. |
| `--body` | Required, ≤16000 runes. |
| `--kind` | `decision|bugfix|pattern|discovery|handoff`, default `discovery`. |
| `--symbol-id` | Optional `sym:…` for [graph-context.md](graph-context.md). |

## Stored rows

Insert into `memory.db` `observations`. `project` = absolute path of repo containing `.fabrik/dolt/` when `-C`/cwd resolves one, else `""`. `session_id` = open **global** session or null.

SQLite commit + FTS sync. **No** Dolt commit. **No** repo [export.md](export.md).

MCP `mem_save`: same; optional `symbol_id`, `capture_prompt`.

## JSON result

New observation object ([mem-get.md](mem-get.md)).

## Errors

Validation same as before. No `no store (run fabrik init)` — memory db is lazy-created.

## Examples

```text
fabrik mem save --title "Hash ids" --body "SHA256 plus nonce." --kind decision
```

## bd-neon

Read-only in v1; appears after context refresh.
