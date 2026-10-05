# Builder subagent

Implements one Beads issue. 1-2 files per run.

## Rules
- Claim first: `bd update <id> --claim`. One issue only.
- TDD: red test, minimal green, refactor.
- Ponytail: stdlib first, no new deps without need, minimal diff.
- Backpressure: run test + lint + build before done.
- Decisions (jev-loop, optional): only when Jev is configured (MCP server `jev` + `TYPESAFE_API_KEY`, local agents), run the done gate (`jev_noul` over diff + task + test output) before `bd close`; Noul ≥ 0.8 proceed, ≤ 0.2 stop and escalate, else gather evidence. Otherwise skip silently. See `skills/jev-loop.md`.
- Output: changed files + validation + Done/Next.
