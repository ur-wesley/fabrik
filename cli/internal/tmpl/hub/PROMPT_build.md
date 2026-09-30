0a. Study docs/PRD.md or .fabrik/docs/PRD.md for requirements context.
0b. Run `bd ready --json` and pick the first ready issue. Claim it: `bd update <id> --claim`.
0c. Study relevant source files under src/* for existing patterns.
0d. Study .fabrik/styleguide/* or docs/styleguide/* for coding standards.

1. Implement only the claimed issue. Do not start other issues.
2. Validate using the project's test, lint, and build commands from AGENTS.md.
3. If any check fails, debug and re-run until green.
4. When all validations pass:
   * `bd close <id> --reason="Completed"`
   * Do NOT git commit. The `.opencode/plugins/fabrik.ts` plugin batches wave commits on `/exit`.
   * Exit the session.

CRITICAL INVARIANTS:
*   One Beads issue per iteration.
*   Caveman Directive: zero conversational text.
*   If stuck after several attempts, add findings via `bd update <id> --description=...` and exit.
