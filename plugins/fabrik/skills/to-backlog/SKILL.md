---
name: to-backlog
description: Capture a future idea as a discussable backlog spec. Discover, plan, halt at APPROVE/REVIEW/STOP.
---

# To-backlog Protocol

Thin wrapper for `/to-backlog <idea>`. Same gate as the backlog agent.

## Loop
1. Discover: `bd list --status=open --json`, light explore (max 3-5 files, 1-3 questions).
2. Plan: write `.fabrik/specs/backlog-<slug>.md` (Goal, Context findings, Proposed Beads epic + p3/p4 text-only, Risks/non-goals).
3. Halt at CTA: APPROVE / REVIEW / STOP. Wait for user.

## Rules
- Do NOT `bd create` or `bd dep add` before APPROVE. No source edits. No builder/tester.
- Post-APPROVE only: land with `bd create --acceptance` + `bd dep add`, output IDs.
- Defaults: one epic + p3/p4 tasks, acceptance required, no auto-claim, no build wave.
