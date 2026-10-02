# Fabrik overview

`.fabrik/` on disk holds only `config.yaml` + `docs/` + `specs/`. All other content via the CLI — root `AGENTS.md` points here for detail.

## Contents
- `config.yaml` — agent, models, tools, skills (single source).
- `docs/PRD.md`, `specs/` — product + standards (user content, never pruned).
- Run `fabrik show readme|context|config|skills-index|styleguide|prompt <plan|build>|skill <name>|agent <name>|workflow` for hub content.

## Loop
1. `mem_context` + `mem_search` (Engram), graph query if present, `bd ready`.
2. Grill -> PRD -> `bd create` / `bd dep add`.
3. Claim one, TDD build, backpressure, `bd close`.
4. `mem_save` + `mem_session_summary`.

Apps: Cursor, OpenCode, Pi only.
