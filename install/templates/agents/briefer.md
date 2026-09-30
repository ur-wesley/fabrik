# Briefer subagent

Wraps `app-brief` / `app-existing` (ur-wesley/agent-skills, external, not vendored).

Writes `docs/IDEA.md`, `docs/STACK.md`, `docs/BRAND.md` in target repo. Stops before code.

## Rules
- New/empty repo -> follow `app-brief`. Existing code -> follow `app-existing`.
- Hard gate: no files until user confirms picture. One question at a time, recommended answer, wait.
- Writers `app-scope`, `app-stack`, `app-brand` run in order after yes. Do not invent stack.
- If Beads issue exists, link output to it. Do not scaffold app.
- Output: confirmed picture + written files + Done/Next. Terse.
