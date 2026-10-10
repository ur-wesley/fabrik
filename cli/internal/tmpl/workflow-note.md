# Fabrik workflow

Use the repo `AGENTS.md` for build, test, and layout. Run `fabrik show workflow|skill|agent|prompt ...` for hub content — `.fabrik/` on disk holds only `config.yaml` + `docs/` + `specs/`.

## Session start

1. ADHD output: next action first, numbered steps, Done/Next at end. Reply in the same language as the user's input.
2. Engram: `mem_context` and `mem_search` for prior decisions.
3. If `graphify-out/graph.json` exists, query the graph before grepping.
4. Beads: `bd ready` for work with no open blockers.

## Plan, land, build

1. **Plan** — plan agent (default in OpenCode): explore, write `.fabrik/specs/<slug>.md`, discuss. No Beads yet.
2. **APPROVE** — switch to build (single-issue fast path, inline land if needed) OR orchestrator (wave: planner lands issues (if needed), then builder + tester only).
3. **REVIEW** — stay in plan; amend spec. No land, no build.
4. Optional: grill (`grill-with-docs`), `to-prd` into `.fabrik/docs/PRD.md` before or during plan.
5. Subagents on demand: `explore`/`researcher` for study, `briefer` only for new repos missing IDEA/STACK, `reviewer`/`style-smells`/`security` only when asked or high-risk scope.

## Tokens

- i-have-adhd output style only: next action first, numbered steps, Done/Next at end. rtk-usage on: `rtk exec` for noisy output.
- Never truncate code, paths, commands, errors.

## Backpressure

Before closing an issue, run the project's test, lint, and build commands from `AGENTS.md`. Fix failures before `bd close`.

## Memory

- Decisions and handoffs go to Engram (`mem_save`, `mem_session_summary`).
- Do not duplicate decisions in `bd remember`; Engram is the context store.
- Beads owns todos only.

## Apps

Same loop in Cursor, OpenCode, Pi, and Antigravity. Pick the app that already has the repo open.
