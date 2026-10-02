# Fabrik skills (Cursor / Pi / OpenCode only)

| Need | Skill | File |
|------|-------|------|
| Less talking (ADHD) | i-have-adhd | `skills/i-have-adhd.md` |
| Less tokens | rtk-usage | `skills/rtk-usage.md` |
| Easier impl + guardrails | ponytail, tdd, diagnose, guardrails | `skills/ponytail.md`, `skills/tdd.md`, `skills/diagnose.md`, `skills/guardrails.md` |
| Plan + execute | plan (primary), orchestrator (after APPROVE) | `fabrik show agent plan` |
| Subagents | explore, researcher, briefer, planner, builder, tester, reviewer, style-smells, security, backlog | `.fabrik/agents/*.md` |
| App briefing | app-brief, app-existing, app-scope, app-stack, app-brand (external `ur-wesley/agent-skills`, not vendored; used by briefer/planner via `docs/IDEA/STACK/BRAND.md`) | — |
| Stack skills | Only when `docs/STACK.md` prescribes them. Never assume. | — |
| Storage + overview | engram, graphify | MCP + `graphify` skill |
| Todos | beads | `bd` CLI |
| Future ideas | to-backlog | `skills/to-backlog.md` + `fabrik show agent backlog` |

Init copies `skills/*.md` to `.cursor/rules/`, `.opencode/skills/`, `.pi/skills/`.
Subagents copy from `.fabrik/agents/` to `.pi/agent/agents/`, `.cursor/agents/`. OpenCode uses `opencode.json` (show-shims via `fabrik show agent <name>`, built-in `build` disabled).
Only Cursor, Pi, OpenCode are configured. No Claude/Codex/other agents.
