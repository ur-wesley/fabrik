# Fabrik Plugin for Antigravity and Gemini CLI

Fabrik brings a unified, disciplined AI agent workflow to **Google Antigravity (AGY)** and **Gemini CLI**:
- **Beads (`bd`)** for dependency-aware task tracking and atomic issue claiming.
- **Engram** for persistent memory across sessions and compactions.
- **Graphify** for visual and structural codebase mapping.
- **TDD & Backpressure** to ensure all tests, linters, and builds pass before work is closed.

## Local projects (Gemini CLI)

Run `fabrik init --apps gemini` to write `.gemini/skills/` shims + register MCP in `~/.gemini/settings.json`.
The global extension (`gemini extensions install <repo>/plugins/fabrik`) is updated by reinstalling — the source of truth is this repo.

## Core Session Loop

1. **Orient**: Check Engram context (`mem_context`, `mem_search`) and query Graphify (`graphify-out/graph.json`).
2. **Align**: Grill requirements and document decisions in `.fabrik/docs/PRD.md`.
3. **Plan**: Propose changes and generate `.fabrik/specs/<slug>.md` using the `plan` agent.
4. **Land**: On `APPROVE`, land atomic tasks into Beads (`bd create`, `bd dep add`).
5. **Build**: Claim tasks (`bd update <id> --claim`), write tests first (TDD), implement minimal diffs, pass quality gates, and close (`bd close <id>`).
6. **Remember**: Save architectural decisions and discoveries in Engram (`mem_save`, `mem_session_summary`).

## Subagents

Invoke with `agy --agent <name>` or via subagent delegation:
- `plan`: Explore and propose a plan into `.fabrik/specs/<slug>.md`.
- `orchestrator`: Router executing post-APPROVE waves (planner -> builder -> tester).
- `build`: Direct implementation of approved specs or ready Beads issues.
- `builder`: Implements a single Beads task with strict TDD.
- `tester`: Executes test suites, verifies pass/fail, and isolates regressions.
- `explore`: Read-only codebase study and pattern inspection.
- `researcher`: External research (docs, web, memory).
- `planner`: Converts approved specs into structured Beads issues with dependencies.
- `reviewer`: Code and diff review against spec requirements.
- `style-smells`: Code quality audit (YAGNI, KISS, DRY, SOLID).
- `security`: Security and vulnerability audit.
- `backlog`: Groom and discover backlog items into Beads.

## Bundled Skills

- `fabrik`: Fabrik CLI commands and navigation reference.
- `beads`: Task tracking and issue management with `bd`.
- `tdd`: Strict test-driven development workflow.
- `diagnose`: Bug isolation and minimal root cause diagnosis.
- `guardrails`: Quality boundaries and error prevention.
- `i-have-adhd`: Fast, structured ADHD communication style.
- `jev-loop`: Jev MCP decision loop for large catalogs.
- `mattpocock-planner`: Spec-first planning protocol.
- `ponytail`: Stdlib-first minimal-diff engineering.
- `rtk-usage`: Quiet CLI execution with token savings.
- `to-backlog`: Backlog discovery and grooming.
