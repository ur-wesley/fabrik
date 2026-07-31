---
name: devin-fusion
description: Devin Fusion Spec-Driven Development (SDD) autonomous workflow for pi agent, including phased exploration, proposal, spec, design, tasks, batch execution, and verification.
---

# Devin Fusion SDD (Spec-Driven Development) Workflow

Autonomous multi-phase orchestration flow for non-trivial software changes.

## Workflow Phases & Commands

### Phase 0: Init & Exploration (`/sdd-init`, `/sdd-explore`)
- Map repository architecture, dependencies, and testing capabilities.
- Explore codebase to collect empirical evidence before proposing changes.

### Phase 1: Proposal (`/sdd-propose`)
- Define the business problem, target outcome, scope boundaries, and non-goals.
- Run interactive question/alignment rounds to clarify ambiguous requirements.

### Phase 2: Technical Spec (`/sdd-spec`)
- Create formal requirement contracts, API schemas, and data flow invariants.
- Specify exact verification criteria for each requirement.

### Phase 3: Architectural Design (`/sdd-design`)
- Map changes to concrete files and modules (Component breakdown).
- Establish component boundaries, data structures, and state flow.

### Phase 4: Task Breakdown (`/sdd-tasks`)
- Generate ordered, atomic implementation tasks.
- Each task includes target files, step-by-step instructions, and exact verification commands.

### Phase 5: Autonomous Execution (`/sdd-apply`)
- Execute tasks in dependency order.
- Apply strict TDD: Write/update test -> Implement code -> Verify pass.
- Mark off tasks as completed in task state store.

### Phase 6: Verification & Archival (`/sdd-verify`, `/sdd-archive`)
- Run full test suite, build check, and static analysis.
- Verify against original specs and produce walkthrough artifact.
- Archive completed change context into memory store.
