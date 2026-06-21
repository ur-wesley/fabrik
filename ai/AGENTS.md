# Operational Rules for AI Agents

## Core Directives

### 🦖 The Caveman Rule (Zero Apologies, Zero Filler)
1.  **Skip all greetings and sign-offs.** Do not say "Here is the code", "Let me know if this works", or "I have finished".
2.  **Do not explain what you did** unless explicitly asked. The code changes and git commit messages are the explanation.
3.  **Never apologize.** If a test fails, do not say "Sorry about that". Fix the issue, run the tests, and output the result.
4.  **Keep text responses to the absolute minimum.** Use short bullet points or code blocks. Speak in fragments.

### 🧪 Backpressure Commands
Before marking a task as done, you must execute these commands to verify your changes. If any command fails, you must self-correct and re-run them.

*   **Test Suite:** `npm run test` (or `pytest` / your project's test command)
*   **Linter:** `npm run lint` (or `eslint` / `ruff` etc.)
*   **Build Check:** `npm run build`

### 📦 Git & Issue State Management
1.  When a task under `ai/.tasks/` is completed and tests pass:
    *   Move the task file from `ai/.tasks/` to `ai/.tasks/completed/`.
    *   Run `git add -A`.
    *   Run `git commit -m "feat: [brief description] (closes #[issue_number])"`.
2.  If the task is not completed, write your findings directly to the task file under a `## Discoveries / Obstacles` section so the next iteration can read them. Do not exit without committing if changes are partial.
