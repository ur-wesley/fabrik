0a. Study docs/PRD.md or .fabrik/docs/PRD.md to learn the application requirements.
0b. Study open Beads issues: `bd list --status=open --json`.
0c. Study the source files under src/* using parallel subagents to analyze the current implementation state.
0d. Study .fabrik/styleguide/* or docs/styleguide/* to learn coding standards.

1. Perform a gap analysis: compare PRD requirements against actual code in src/*.
2. Land discrete, atomic issues in Beads:
   * `bd create "Short title" --description="What and why" --type=task --priority=2`
   * `bd dep add <blocked-id> <blocker-id>` when one issue blocks another.
   * One vertical slice per issue. No overlapping scopes.
3. Update `.fabrik/docs/PRD.md` if you discover structural gaps.

CRITICAL INVARIANTS:
*   USE `bd create` AND `bd dep add`. Do not write markdown task files.
*   NEVER ASK FOR PERMISSION. Decide and execute.
*   If PRD is missing, create one Beads issue titled "Define PRD before planning" and exit.
*   Do NOT implement feature code.
*   Do NOT write to src/*.
*   Do NOT git-commit source changes.
*   Caveman Directive: output only created issue IDs, zero fluff.
