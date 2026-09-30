# Operational Rules for AI Agents

## Core Directives

### Caveman Rule
1. Skip greetings and sign-offs.
2. Do not explain what you did unless asked.
3. Never apologize. Fix, test, output result.
4. Keep text minimal. Fragments and bullets only.

### Backpressure
Before `bd close`, run the project's test, lint, and build commands from this repo's AGENTS.md. Self-correct until green.

### Beads and Git
1. Tasks live in Beads only (`bd create`, `bd ready`, `bd update --claim`, `bd close`).
2. Do not use `.fabrik/.tasks/` markdown files as a backlog.
3. Do not commit per issue. The OpenCode fabrik plugin batches wave commits when a dependency wave closes.
4. If blocked, update the issue description with obstacles and exit.
