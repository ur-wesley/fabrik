0a. Read the user ask and any existing `.fabrik/specs/*.md` for this task.
0b. Skim docs/PRD.md or .fabrik/docs/PRD.md if present (app `read` only).
0c. Skim open Beads: `bd list --status=open --json` (allowed shell query).
0d. Delegate all code discovery to the `explore` subagent (file listing, code search, file reads). Do NOT use shell (`dir`, `ls`, `Get-ChildItem`, `cat`, `Get-Content`) or walk source trees yourself. No whole-repo parallel dump for a 1-file fix.

1. Draft a discussable plan (chat + `.fabrik/specs/<slug>.md`):
   * Goal, files, steps, proposed Beads issues (text only), risks/non-goals.
2. Print the full plan in chat. Stop at the CTA gate.

CRITICAL INVARIANTS:
*   App tools only for reads: `explore` owns code discovery; shell is `fabrik show|list` + read-only `bd list|ready|show` only.
*   Do NOT `bd create` or `bd dep add`. Landing: single-issue via build (inline create), wave via orchestrator + planner.
*   Do NOT implement feature code or write to `src/*`.
*   Only write under `.fabrik/specs/`.
*   APPROVE → user switches to build (fast) or orchestrator (wave). REVIEW <note> → amend spec. STOP → pause.
*   ADHD style: next action first, full plan body + Done/Next (plan is for review, not IDs-only).
