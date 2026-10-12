# Fabrik workflow

Use the repo `AGENTS.md` for build, test, and layout. Run `fabrik show workflow|skill|agent|prompt ...` for hub content — `.fabrik/` on disk holds only `config.yaml` + `docs/` + `specs/`.

## Session start

1. ADHD output: next action first, numbered steps, Done/Next at end. Reply in the same language as the user's input.
2. `mem_current_project`, then `mem_context` (all projects) and `mem_search` when needed — Fabrik MCP memory tools.
3. In a Fabrik repo: `graph_explore` before grepping.
4. `issue_ready` (or `fabrik issue ready`) for work with no open blockers.

## Plan, land, build

1. **Plan** — plan agent (default in OpenCode): explore, write `.fabrik/specs/<slug>.md`, discuss. No issues yet.
2. **APPROVE** — switch to build (single-issue fast path, inline land if needed) OR orchestrator (wave: planner lands issues (if needed), then builder + tester only).
3. **REVIEW** — stay in plan; amend spec. No land, no build.
4. Optional: grill (`grill-with-docs`), `to-prd` into `.fabrik/docs/PRD.md` before or during plan.
5. Subagents on demand: `explore`/`researcher` for study, `briefer` only for new repos missing IDEA/STACK, `reviewer`/`style-smells`/`security` only when asked or high-risk scope.

## Tokens

- i-have-adhd output style only: next action first, numbered steps, Done/Next at end. rtk-usage on: `rtk exec` for noisy output.
- Never truncate code, paths, commands, errors.

## Backpressure

Before closing an issue, run the project's test, lint, and build commands from `AGENTS.md`. Fix failures before `issue_close` or `fabrik issue close`.

## Memory

- Decisions and handoffs via Fabrik MCP (`mem_save`, `mem_session_summary`).
- Machine memory lives in `~/.fabrik/memory.db`, not repo Dolt.
- Issues/todos: `fabrik issue` / `issue_*` MCP tools only.

## Apps

Same loop in Cursor, OpenCode, Pi, and Antigravity. Pick the app that already has the repo open.
