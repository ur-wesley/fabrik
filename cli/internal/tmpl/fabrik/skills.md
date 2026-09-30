# Fabrik skills (Cursor / Pi / OpenCode only)

| Need | Skill | File |
|------|-------|------|
| Less talking (ADHD) | i-have-adhd | `skills/i-have-adhd.md` |
| Less tokens | caveman, rtk-usage | `skills/caveman.md`, `skills/rtk-usage.md` |
| Easier impl + guardrails | ponytail, tdd, diagnose, guardrails | `skills/ponytail.md`, `skills/tdd.md`, `skills/diagnose.md`, `skills/guardrails.md` |
| Subagents | orchestrator, explore, researcher, briefer, planner, builder, tester, reviewer, style-smells, security | `.fabrik/agents/*.md` |
| App briefing | app-brief, app-existing, app-scope, app-stack, app-brand (external `ur-wesley/agent-skills`, not vendored; used by briefer/planner via `docs/IDEA/STACK/BRAND.md`) | — |
| Stack skills | Only when `docs/STACK.md` prescribes them. Never assume. | — |
| Storage + overview | engram, graphify | MCP + `graphify` skill |
| Todos | beads | `bd` CLI |

Init copies `skills/*.md` to `.cursor/rules/`, `.opencode/skills/`, `.pi/skills/`.
Subagents copy from `.fabrik/agents/` to `.pi/agent/agents/`, `.cursor/rules/agents/`. OpenCode uses `opencode.json` (file refs, built-in `build` disabled).
Only Cursor, Pi, OpenCode are configured. No Claude/Codex/other agents.
