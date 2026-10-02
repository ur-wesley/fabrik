# Operational Rules for AI Agents

## Core Directives

### Output Style (i-have-adhd only)
1. Lead with next action.
2. Numbered steps, restate state, suppress tangents.
3. End with Done/Next.
4. Reply in the same language as the user's input. Code, paths, commands, and symbols stay as in the repo.

### Tokens
rtk-usage on: `rtk exec` for noisy output.

### Backpressure
Before `bd close`, run the project's test, lint, and build commands from this repo's AGENTS.md. Self-correct until green.

### Plan and execute
1. Plan first: `.fabrik/specs/` + chat; no Beads until APPROVE (`agents/cta.md`).
2. After APPROVE: planner lands (if needed), then builder + tester by default.
3. Skip reviewer/style-smells/security unless asked or auth/secrets/network scope.
4. Ask first for: destructive actions, git commit/push/sync, scope expansion beyond the claimed issue.

### Beads and Git
1. Tasks live in Beads only (`bd create`, `bd ready`, `bd update --claim`, `bd close`).
2. Do not use `.fabrik/.tasks/` markdown files as a backlog.
3. Do not commit per issue. The OpenCode fabrik plugin batches wave commits when a dependency wave closes.
4. If blocked, update the issue description with obstacles and exit.
