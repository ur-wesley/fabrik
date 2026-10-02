# Shared CTA gate (plan → build OR land → wave)

- **APPROVE / APPROVE build** → switch to build: direct implement 1 spec / 1 Beads issue (no planner hop, create single issue inline if needed, no commit).
- **APPROVE orchestrator** → switch to orchestrator: planner lands, then builder + tester wave (no commit).
- **REVIEW <note>** → stay in plan; amend spec. No Beads, no build.
- **CONTINUE** = auto-backpressure. **STOP** = pause.
