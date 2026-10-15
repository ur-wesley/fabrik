# neon

Fabrik human UI: Kanban, memory, and code lookup. Crate name is `neon`.
Spec: [v2/neon-board.md](../v2/neon-board.md), [v2/neon-memory.md](../v2/neon-memory.md),
[v2/neon-lookup.md](../v2/neon-lookup.md).

```bash
cargo run --manifest-path gui/Cargo.toml -- /path/to/repo
```

From `gui/`, `cargo dev` runs watch + `cargo run` (aliases in `.cargo/config.toml`). Install `cargo-watch` via `mise install` at the repo root.

Requires `fabrik` on PATH (or `FABRIK_BIN`). Issue store needs `CGO_ENABLED=1` build of the CLI (`mise run cli:build-cgo`, which errors clearly when gcc is missing).

## How it works

Every Fabrik call runs off the UI thread in `RepoWorker` (`src/watch/`): the UI sends a
`WorkerMsg`, the worker thread runs `fabrik … --json`, and the result comes back as an
`Action` that updates state, fires a toast, and refreshes the pane. `fabrik listen` SSE
(`~/.fabrik/listen.json`) adds live refresh on top: issue writes refresh the board,
memory writes refresh the memory pane, graph writes refresh lookup.

Issue IDs are `f-<hex>` (`fabrik issue …` rejects anything else); memory IDs are `o-<hex>`.
The issue store is Dolt (`.fabrik/dolt`); memory and the code graph are machine-wide/repo
SQLite DBs. There are no `bd` commands: the GUI shells `fabrik issue …` / `fabrik mem …` /
`fabrik graph …` only.

## Board (issues)

Reads: `issue list --json`, `issue ready --json`, `issue show <id> --json`.

Writes (detail popup + `+ New` in the toolbar; all via worker + toast + refresh):

```bash
fabrik issue create "title" --body … --acceptance … --priority 0-4 --kind task|bug|feature|epic|chore --json
fabrik issue claim <id> --by <name> --json
fabrik issue unclaim <id> --by <name> --force --json
fabrik issue close <id> --reason … --by <name> --force --json
fabrik issue dep <blocked> <blocker> --kind blocks|relates --json
fabrik issue note <id> <body> --json   # comment composer (Ctrl+Enter)
```

Claim shows for open issues, Unclaim for active ones; `--force` is a checkbox.
Dep targets are added from the dependencies section (`f-…` id + kind toggle).

## Memory

Reads: `mem context --json`, `mem search <q> [--project <label>] --json`,
`mem get <id> --json`, `mem check --json`, `mem stats --json`,
`mem timeline --id <id> --limit <n> --json`, `mem current-project --json`.

Writes (`+ New` / `Save prompt` rows; edit + delete on each observation; delete asks
for confirmation first):

```bash
fabrik mem save --title … --body … --kind decision|bugfix|pattern|discovery|handoff [--topic-key …] [--symbol-id …] --json
fabrik mem update --id <o-…> [--title …] [--body …] [--kind …] [--topic-key …] [--symbol-id …] --json
fabrik mem delete --id <o-…>
fabrik mem save-prompt --title … --capture-prompt … --json
```

## Lookup (code graph)

Reads: `graph status --json`, `graph overview --budget <n> --json` (instant
overview for the empty canvas), `graph explore <q> --budget 40 --json` (falls
back to `graph query`), `graph explain <id> --json`, `graph path <from> <to> --json`,
`graph context <symbol> --json`, `graph impact <symbol> --depth <n> --json`,
`graph detect-changes [--ref <ref>] --json`, `graph explore <q> --issue <f-…> --json`.
`graph build --json` indexes the repo.

The GUI shells `fabrik` from PATH, or `FABRIK_BIN` when set. On Windows the
configured path resolves deterministically: an existing configured file wins
over any sibling, otherwise the sibling with/without `.exe` is used (either
direction), and the resolved absolute path is what error cards display. The instant
overview needs a CLI built after `graph overview` landed: an older binary
fails with `unknown command "overview"`, which the Lookup pane keeps as a
persistent error card (message plus binary path) with a Retry button. Prefer
`mise run gui:run` — it rebuilds the CLI into `cli/fabrik.exe` on Windows
(`cli/fabrik` elsewhere) and points `FABRIK_BIN` at exactly that file — over
`cargo run`, which inherits whatever `fabrik` your shell resolves. Do not
leave an extensionless `cli/fabrik` beside `cli/fabrik.exe` on Windows:
CreateProcess appends `.exe` to extensionless spawn paths, so the `.exe`
would shadow the fresh binary; both names are gitignored build artifacts.

Select a node, then `context` / `impact` inspect it; `changes` maps the working diff to
symbols. When `graph status` reports stale files, the setup banner shows a
`review changes` shortcut. The issue popup's `explore in graph` seeds explore from the
issue (`--issue <f-…>`).
