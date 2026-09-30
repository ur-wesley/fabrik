# Fabrik overview

`.fabrik/` is the project overview hub. Root `AGENTS.md` points here for detail.

## Contents
- `config.yaml` — agent, models, tools, skills (single source).
- `CONTEXT.md` — domain language + architecture rules.
- `AGENTS.md` — operational rules (caveman, backpressure, beads+git).
- `skills.md` — skill manifest for Cursor / Pi / OpenCode.
- `agents/` — explore / builder / reviewer subagent defs.
- `docs/PRD.md`, `specs/`, `styleguide/` — product + standards.
- `PROMPT_plan.md`, `PROMPT_build.md`, `loop.sh/ps1`, `fabrik.sh/ps1` — runners.

## Loop
1. `mem_context` + `mem_search` (Engram), graph query if present, `bd ready`.
2. Grill -> PRD -> `bd create` / `bd dep add`.
3. Claim one, TDD build, backpressure, `bd close`.
4. `mem_save` + `mem_session_summary`.

Apps: Cursor, OpenCode, Pi only.
