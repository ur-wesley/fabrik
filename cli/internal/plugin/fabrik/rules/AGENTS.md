# Fabrik Workflow Rules

These rules are enforced whenever the Fabrik plugin is active.

## Non-Interactive Shell Commands

ALWAYS use non-interactive flags with file operations to avoid hanging on confirmation prompts:
- Overwrite files without prompting: `cp -f source dest`, `mv -f source dest`, `rm -f file`
- Recursive operations: `rm -rf directory`, `cp -rf source dest`
- Other commands that prompt: `git checkout -f`, package manager flags (`-y`, `--non-interactive`)

## Session Start Protocol

1. **ADHD Output Style**:
   - State next action first
   - Numbered steps for operations
   - End with Done and Next actions
   - Reply in the same language as the user's prompt
2. **Engram Persistent Memory**:
   - Call `mem_context` and `mem_search` to recall prior decisions and architectural context before starting.
3. **Graphify Code Map**:
   - If `graphify-out/graph.json` exists, query the graph before grepping the codebase.
4. **Beads Issue Tracking**:
   - Run `bd ready` to discover ready work with no open blockers.

## Plan, Land, Build Workflow

1. **Plan** (Plan subagent / `/plan`):
   - Read-only on source code.
   - Propose changes by writing a spec to `.fabrik/specs/<slug>.md`.
   - Never create Beads issues or write source code during the Plan phase.
2. **APPROVE Router**:
   - On user `APPROVE`: switch to build (single-issue fast path) or orchestrator (multi-issue wave: planner lands issues, builder implements, tester verifies).
3. **REVIEW**:
   - Amend the spec in `.fabrik/specs/<slug>.md` and wait for new approval.
4. **Backpressure & Quality Gates**:
   - TDD loop: test first, minimal implementation, verify pass.
   - Run project test, lint, and build gates before closing any issue.
   - Fix all failures before running `bd close <id>`.

## Memory vs Task Discipline

- **Decisions and Architecture**: Persist to Engram (`mem_save`, `mem_session_summary`).
- **Tasks and Todos**: Manage solely in Beads (`bd create`, `bd update --claim`, `bd close`).
- Do NOT use `bd remember` for architectural decisions.
- Do NOT use markdown TODO files when Beads is active.

## Token Conservation

- Use `rtk exec` for noisy commands when available.
- Never truncate code, paths, commands, or error outputs.
