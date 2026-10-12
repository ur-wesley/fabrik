# bd-neon board

## Purpose

Kanban over `fabrik issue`. Read-only for status changes.

## What it runs

When `~/.fabrik/listen.json` exists, subscribe to SSE `issues` and `graph` for this repo and refetch. Otherwise on select and when `.fabrik/dolt/` or `.fabrik/graph.db` mtimes change:

```text
fabrik -C <repo> check --json
fabrik -C <repo> issue list --json
fabrik -C <repo> issue ready --json
```

Detail and notes: [issue-show.md](issue-show.md), [issue-note.md](issue-note.md).

Parses Fabrik hash ids (`f-…`). No Beads wire types.

## Project detection

`.fabrik/config.yaml` plus `.fabrik/dolt/` with the `fabrik` database. Watcher includes `.fabrik/dolt/`. Cache staleness uses max mtime under `.fabrik/dolt/` and `graph.db`.

## Sync

bd-neon does not run `dolt pull` or `push`. A person runs those in a terminal; the watcher picks up local changes.
