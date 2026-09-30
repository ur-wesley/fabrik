---
name: guardrails
description: Implementation guardrails. Non-interactive shells, backpressure, beads-only todos, no destructive git without approval.
---

# Guardrails

1. **Non-interactive shells**: `cp -f`, `mv -f`, `rm -rf`, `apt-get -y`, `ssh -o BatchMode=yes`. Never hang on prompt.
2. **Backpressure**: before `bd close`, run test + lint + build from AGENTS.md. Fix first.
3. **Beads only**: `bd ready/show/update --claim/close`. No markdown TODOs. No `bd remember` for decisions (Engram owns that).
4. **Git safety**: no `push`, `reset --hard`, `clean -fd`, `branch -D` without explicit user approval.
5. **Minimal diffs**: touch only needed lines. No speculative abstractions.
