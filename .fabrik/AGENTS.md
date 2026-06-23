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
1.  When a task under `.fabrik/.tasks/` is completed and tests pass:
    *   Move the task file from `.fabrik/.tasks/` to `.fabrik/.tasks/completed/`.
    *   Do NOT commit. The `.opencode/plugins/fabrik.ts` plugin owns commits — it batches every task in the current wave into a single `feat: ...` commit when the wave closes. Per-task commits are forbidden.
2.  If the task is not completed, write your findings directly to the task file under a `## Discoveries / Obstacles` section so the next iteration can read them. Do not exit without archiving if changes are partial.
