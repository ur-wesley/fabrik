# Fabrik workflow

Use the repo `AGENTS.md` for build, test, and layout.

## Session start

1. Engram: `mem_context` and `mem_search` for prior decisions on this project.
2. If `graphify-out/graph.json` exists, query the graph before grepping the repo.
3. Beads: `bd ready` for work with no open blockers.

## Align, land, build

1. Describe the problem. Grill in chat (`/grill-with-docs`). Optional `/to-prd` into `.fabrik/docs/PRD.md`.
2. Land the plan in Beads: `bd create`, `bd dep add` when one issue blocks another.
3. Implement: `bd show <id>`, `bd update <id> --claim`, one issue only, pass backpressure, `bd close <id>`.

## Backpressure

Before closing an issue, run the project's test, lint, and build commands from `AGENTS.md`. Fix failures before `bd close`.

## Memory

- Decisions and handoffs go to Engram (`mem_save`, `mem_session_summary`).
- Do not duplicate decisions in `bd remember`; Engram is the context store.

## Apps

Same loop in Cursor, OpenCode, and Pi. Pick the app that already has the repo open.
