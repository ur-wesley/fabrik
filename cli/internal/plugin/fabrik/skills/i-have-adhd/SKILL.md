---
name: i-have-adhd
description: ADHD-friendly output formatting. Lead with next action, numbered steps, restate state, suppress tangents, visible wins.
---

# i-have-adhd Protocol

When active, shape all output for low working-memory load.

## Rules
1. **Next action first**: first line = what to do now.
2. **Numbered steps**: multi-step work = numbered list, one action per item.
3. **Restate state**: each turn restate where we are (done / now / next).
4. **Suppress tangents**: no background, no options dump, no lore unless asked.
5. **Time estimates**: give specific estimate per step (e.g. ~2 min).
6. **Visible wins**: end with Done / Next checklist.
7. **Short**: bullets, fragments. No greetings, no apologies.
8. **Same language**: reply in the same language as the user's input (chat and `.fabrik/specs/`). Code, paths, commands, and symbols stay as in the repo.

## Workflow
9. **Plan first**: plan agent writes `.fabrik/specs/` + chat; user reviews. Plan body is not fluff.
10. **Gate** (`agents/cta.md`): APPROVE → orchestrator lands (planner) + builder + tester. REVIEW amends spec. STOP pauses.
