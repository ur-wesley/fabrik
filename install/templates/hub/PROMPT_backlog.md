0a. Read the idea (`/to-backlog <idea>`) and any existing `.fabrik/specs/backlog-*.md` for it.
0b. Skim open Beads: `bd list --status=open --json`. Skip land on APPROVE if a ready issue already matches.
0c. Light explore only (max 3-5 files, use explore/researcher subagent if needed). No whole-repo dump.

1. Draft a discussable backlog spec (chat + `.fabrik/specs/backlog-<slug>.md`):
   * Goal, context findings, proposed Beads epic + p3/p4 tasks (text only), risks/non-goals.
2. Print the full spec in chat. Stop at the CTA gate.

CRITICAL INVARIANTS:
*   Do NOT `bd create` or `bd dep add` before APPROVE. Landing happens only after user APPROVE.
*   Do NOT implement feature code or write to `src/*`. Do NOT spawn builder/tester.
*   Only write under `.fabrik/specs/backlog-<slug>.md`.
*   Defaults: one epic + p3/p4 tasks, `--acceptance` required, no auto-claim, no build wave.
*   APPROVE → land Beads (`bd create` + `bd dep add`), output IDs, still no build. REVIEW <note> → amend spec. STOP → pause.
*   ADHD style: next action first, full spec body + Done/Next (spec is for review, not IDs-only).
