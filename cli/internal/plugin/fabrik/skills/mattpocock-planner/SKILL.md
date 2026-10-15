---
name: mattpocock-planner
description: Matt Pocock's structured AI engineering framework for planning, organizing, grilling with docs, and breaking specs into tracer bullet tickets.
---

# Matt Pocock Planning & Organizing Framework

Structured workflow for managing software engineering tasks from ambiguous idea to shipped code without "vibing".

## Core Commands & Phases

### 1. Repository Setup & Context (`/setup-matt-pocock-skills`)
- Maintain a single source of truth context file (e.g. `CONTEXT.md` or `AGENTS.md`) documenting domain models, stack constraints, and issue tracking.
- Verify repository structure and architecture rules before taking any action.

### 2. Grill With Docs (`/grill-with-docs`)
- Before writing implementation code, interview the user/codebase to uncover:
  - Domain invariants and business rules
  - Current-state vs future-state gaps
  - Non-goals and explicit boundaries
  - Tradeoffs and failure modes
- Anchor all answers directly in existing codebase files or explicit documentation.

### 3. Specification Formulation (`/to-spec`)
- Convert discussions into concrete, testable specifications.
- Structure specs into:
  - **Goal**: Clear objective and user benefit.
  - **Inputs & State**: Expected data contracts and types.
  - **Invariants**: Mandatory technical and domain rules.
  - **Verification**: Exact test cases and acceptance criteria.

### 4. Tracer Bullet Tickets (`/to-tickets`)
- Deconstruct the specification into bite-sized, sequential "tracer bullet" work units.
- Each ticket must:
  - Be executable independently in a clean sub-task context.
  - Deliver a working vertical slice (end-to-end functionality).
  - Contain explicit verification criteria (e.g. test command to pass).

### 5. Execution & Strict TDD
- Write failing test first -> implement minimum code -> pass test -> refactor.
- Run static analysis / type checks continuously.
