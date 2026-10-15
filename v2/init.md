# fabrik init

## Purpose

Create the v2 Dolt database, SQLite graph shell, and thin `.fabrik` hub in one repo. Machine install is [setup.md](setup.md).

## Who calls it

A person, once per repo. Agents do not init unless asked.

## Invocation

```text
fabrik init [path] [--apps cursor,pi,antigravity,opencode] [--json]
```

`path` defaults to the current directory. Warns on stderr when the directory is not a git repo. Does not run `git init`.

## Inputs

| Input | Rule |
| --- | --- |
| `path` | Target directory. |
| `--apps` | Subset of `cursor`, `pi`, `antigravity`, `opencode`. |

## Stored rows

When `.fabrik/dolt/` is absent:

1. Create embedded Dolt database `fabrik` under `.fabrik/dolt/` using [SCHEMA.sql](SCHEMA.sql).
2. Set committer name and email from `git config user.name` / `user.email`, or `fabrik` / `fabrik@local`.
3. If `git remote get-url origin` succeeds, register `fabrik dolt remote add origin <url>`.

Create empty `.fabrik/graph.db` with **code graph** tables only when missing (no observation FTS).

Also writes `.fabrik/config.yaml`, `docs/`, `specs/`, app shims, and gitignore lines for `.fabrik/dolt/`, `.fabrik/graph.db`, `.fabrik/export/`.

Registers `fabrik mcp` for selected apps ([setup.md](setup.md)). No network downloads. Does not create `~/.fabrik/memory.db` (that is [setup.md](setup.md) or first mem use).

Schema version lives in Dolt `meta` table (v1 = `1`). Newer schema than the binary understands is an error.

## JSON result

```json
{
  "repo": "/src/app",
  "dolt": "/src/app/.fabrik/dolt",
  "graph": "/src/app/.fabrik/graph.db",
  "created": true
}
```

`created` is false when the Dolt database already existed.

## Text result

```text
dolt /src/app/.fabrik/dolt
graph /src/app/.fabrik/graph.db
```

## Errors

| Case | Exit | Message |
| --- | --- | --- |
| Path is a file | 1 | `fabrik: not a directory: …` |
| Schema too new | 1 | `fabrik: store schema N is newer than this fabrik` |
| Unknown app | 2 | `fabrik: unknown app: …` |

## Examples

```text
fabrik init
fabrik init /src/app --apps cursor,opencode
```

## bd-neon

A project folder has `.fabrik/config.yaml` and `.fabrik/dolt/` with the `fabrik` database.
