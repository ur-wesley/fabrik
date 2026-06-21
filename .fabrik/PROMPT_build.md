0a. Study .fabrik/docs/PRD.md to understand the requirements context.
0b. Study the task list under `.fabrik/.tasks/` and select the lowest-numbered open task file.
0c. Study the relevant source files under src/* to find existing implementation patterns.
0d. Study .fabrik/styleguide/* using parallel subagents to learn the project's coding standards and design principles.

1. Implement the selected task's requirements. Do not start work on other tasks.
2. Validate your implementation:
   * Run the test suite: `npm run test` (or the project test command).
   * Run the linter: `npm run lint` (or the project lint command).
   * Run the build check: `npm run build` (or the project build command).
3. If any check fails, debug the failures and re-run the tests. Do not proceed until tests are green.
4. Once all validations pass:
   * Move the selected task file from `.fabrik/.tasks/` to `.fabrik/.tasks/completed/`.
   * Stage all changes: `git add -A`.
   * Commit with a message: `feat: implement [task-name] (closes #[task-number])`.
   * Exit the session.

CRITICAL INVARIANTS:
*   Never edit more than ONE task per iteration.
*   Enforce the Caveman Directive: zero conversational text, zero filler, zero apologies.
*   If you get stuck and cannot pass validations after several attempts, document your findings in the task file under `## Current Obstacles` and exit so the loop can report the error.
