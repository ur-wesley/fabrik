0a. Study docs/PRD.md or .fabrik/docs/PRD.md to learn the application requirements.
0b. Study .fabrik/.tasks/* (if present) to understand what tasks have already been generated.
0c. Study the source files under src/* using parallel subagents to analyze the current implementation state.
0d. Study .fabrik/styleguide/* or docs/styleguide/* using parallel subagents to learn the project's coding standards and design principles.

1. Perform a gap analysis: compare docs/PRD.md or .fabrik/docs/PRD.md requirements against the actual code in src/*.
2. Generate discrete, atomic, decoupled task files under `.fabrik/.tasks/` (e.g., `.fabrik/.tasks/001-setup-db.md`, `.fabrik/.tasks/002-auth-routes.md`).
   * Each file must contain exactly ONE clear vertical slice (Job-to-Be-Done).
   * Ensure task scopes do not overlap to prevent code conflicts.
   * Write the task files in markdown format with a list of checkable requirements.
3. If an existing task file is completed but not archived, move it to `.fabrik/.tasks/completed/`.
4. Update docs/PRD.md or .fabrik/docs/PRD.md if you discover structural gaps or need to document technical decisions.

CRITICAL INVARIANTS:
*   If both docs/PRD.md and .fabrik/docs/PRD.md are missing, empty, or lack requirements, do NOT output conversational questions or exit silently. Instead, immediately write a single task file `.fabrik/.tasks/000-initialize-prd.md` explaining that a PRD must be defined before planning can run, and exit.
*   Do NOT implement any feature code.
*   Do NOT write anything to src/*.
*   Do NOT run git commits for source code changes.
*   Apply the Caveman Directive: output only the list of created tasks, zero fluff.
