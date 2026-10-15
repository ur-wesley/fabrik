# bd-neon board

## Purpose

Kanban over `fabrik issue` with v2-write-board scope: create, claim, unclaim, close, dep, and note.

## What it runs

When `~/.fabrik/listen.json` exists, subscribe to SSE `issues` and `graph` for this repo and refetch. Otherwise on select and when `.fabrik/dolt/` or `.fabrik/graph.db` mtimes change:

```text
fabrik -C <repo> check --json
fabrik -C <repo> issue list --json
fabrik -C <repo> issue ready --json
```

Writes (detail popup + `+ New` in the toolbar; all with `--json` via worker + toast + refresh):

```text
fabrik -C <repo> issue create "title" --body … --acceptance … --priority 0-4 --kind task|bug|feature|epic|chore --json
fabrik -C <repo> issue claim <id> --by <name> --json
fabrik -C <repo> issue unclaim <id> --by <name> --force --json
fabrik -C <repo> issue close <id> --reason … --by <name> --force --json
fabrik -C <repo> issue dep <blocked> <blocker> --kind blocks|relates --json
fabrik -C <repo> issue note <id> <body> --json
```

`--by` names the actor; `--force` is a checkbox for unclaim/close.

Detail and notes: [issue-show.md](issue-show.md), [issue-note.md](issue-note.md).

Parses Fabrik hash ids (`f-…`). No Beads wire types.

## Project detection

`.fabrik/config.yaml` plus `.fabrik/dolt/` with the `fabrik` database. Watcher includes `.fabrik/dolt/`. Cache staleness uses max mtime under `.fabrik/dolt/` and `graph.db`.

## Sync

bd-neon does not run `dolt pull` or `push`. A person runs those in a terminal; the watcher picks up local changes.
