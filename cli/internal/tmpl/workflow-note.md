# Fabrik workflow

Use the repo `AGENTS.md` for build, test, and layout. `.fabrik/` is the overview hub (README, CONTEXT, config, skills, agents).

## Session start

1. ADHD output: next action first, numbered steps, Done/Next at end.
2. Engram: `mem_context` and `mem_search` for prior decisions.
3. If `graphify-out/graph.json` exists, query the graph before grepping.
4. Beads: `bd ready` for work with no open blockers.

## Align, land, build

1. Describe the problem. Grill in chat (`grill-with-docs`). Optional `to-prd` into `.fabrik/docs/PRD.md`.
2. Land the plan in Beads: `bd create`, `bd dep add` when one blocks another.
3. Implement with guardrails + TDD: one issue only (`bd show`, `bd update --claim`), minimal diff, ponytail (stdlib first).
4. Use subagents: `orchestrator` routes; `briefer` for app brief, `explore`/`researcher` for study, `planner` for Beads issues, `builder` for 1-2 file edits, `tester` for tests, `reviewer`/`style-smells`/`security` for review.

## Tokens

- Caveman + RTK on: terse bullets, `rtk exec` for noisy shell output, `--oneline/--quiet` flags.
- Never truncate code, paths, commands, errors.

## Backpressure

Before closing an issue, run the project's test, lint, and build commands from `AGENTS.md`. Fix failures before `bd close`.

## Memory

- Decisions and handoffs go to Engram (`mem_save`, `mem_session_summary`).
- Do not duplicate decisions in `bd remember`; Engram is the context store.
- Beads owns todos only.

## Apps

Same loop in Cursor, OpenCode, and Pi only. Pick the app that already has the repo open.
