# Dolt and version control

**Issues** live in embedded Dolt under `.fabrik/dolt/`. Versioned SQL, history, branches, git-ref sync. Fabrik tables in [SCHEMA.sql](SCHEMA.sql). **Memory** is **not** in Dolt — see [MEMORY.sql](MEMORY.sql) and [STORE.md](STORE.md).

## Modes

| Mode | Default | Data | Writers |
| --- | --- | --- | --- |
| Embedded | yes | `.fabrik/dolt/` | One `fabrik` process at a time (file lock) |
| Server | later | `.fabrik/dolt/` via TCP | Many clients (`fabrik dolt serve`) |

v1 ships embedded only.

## Auto-commit on write

Mutating `fabrik issue` commands call `DOLT_COMMIT` with a short message. `fabrik mem` uses SQLite on `memory.db` only — no Dolt commit.

## Version control commands

| Command | Doc |
| --- | --- |
| `fabrik vc status` | [vc-status.md](vc-status.md) |
| `fabrik vc log` | [vc-log.md](vc-log.md) |
| `fabrik vc diff` | [vc-diff.md](vc-diff.md) |
| `fabrik vc commit -m …` | [vc-commit.md](vc-commit.md) |
| `fabrik vc branch` / `checkout` | [vc-branch.md](vc-branch.md) |
| `fabrik vc merge <branch>` | [vc-merge.md](vc-merge.md) |
| `fabrik history <id>` | [history.md](history.md) |

Dolt branches are independent of git branches.

## Sync

| Command | Doc |
| --- | --- |
| `fabrik dolt remote add|list` | `init` sets `origin` from git when present |
| `fabrik dolt pull` | [dolt-pull.md](dolt-pull.md) |
| `fabrik dolt push` | [dolt-push.md](dolt-push.md) |

Remote data uses `refs/dolt/data`. **`dolt push` moves issues only**, not memory or graph.

Workflow: pull before creating or claiming work; push before handoff.

## Export

[export.md](export.md) writes issue JSONL after each issue Dolt commit.

## What Dolt does not hold

- Machine memory (`~/.fabrik/memory.db`)
- Code graph (`.fabrik/graph.db`) — rebuild with `fabrik graph build`

## Pin and cgo

Driver pin: target Dolt **v2.2.0** semantics (`DOLT_RESET('--hard')` on abandoned pull). `cli/go.mod` uses `github.com/dolthub/driver` (tracks embedded Dolt). Release CI uses `CGO_ENABLED=1` and ICU on macOS.

## Standalone `dolt` CLI

Optional for `dolt sql` debugging and server mode.
