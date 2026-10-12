# bd-neon lookup

## Purpose

Ask the code graph a question and read the neighborhood. Data lives in `.fabrik/graph.db` (local SQLite, not pushed with `fabrik dolt push`). Build with `fabrik graph build`.

## Who uses it

A person tracing a symbol. Agents use [graph-query.md](graph-query.md) from the CLI before they grep.

## What it runs

Search box (preferred):

```text
fabrik -C <repo> graph explore <text> --budget 40 --json
```

Fallback: `graph query` ([graph-query.md](graph-query.md)).

Click a node:

```text
fabrik -C <repo> graph explain <id> --json
```

Second click, while one node is already selected, asks for a path:

```text
fabrik -C <repo> graph path <from> <to> --json
```

Typing in the search box clears the path and selects nothing.

## What it shows

- A node list: name, kind, `path:line`.
- Under the selected node, its edges. Unresolved calls show the label and no target.
- A path replaces the neighborhood until the next query. Nodes stay in walk order.

The pane does not draw a free graph canvas in v1. It is a list.

## Empty states

| CLI result | Pane |
| --- | --- |
| No Fabrik store | Same init banner as the board (`fabrik init` on click). |
| `indexed: false` / empty index | **Build the code graph** card with **Build graph** (runs `fabrik graph build --json`). |
| `stale_files > 0` (from `graph status`) | Same card with **Rebuild graph**. |
| `no matches` / empty `nodes` | `no matches`. |
| `no path` | `no path` between the two ids. |

On Lookup enter, repo switch, graph SSE, and after a successful build:

```text
fabrik -C <repo> graph status --json
```

Toolbar: **Build** or **Rebuild** (outline) whenever the repo is initialized. Disabled with spinner while build runs. Build is never started on a timer.

## What it does not do

No editor launch, no graph.html export. The path on a file node is text the person can copy.

## Errors

A missing store uses the same `fabrik init` empty state as the board. Explain and path errors for an unknown id are a banner. The previous list stays.
