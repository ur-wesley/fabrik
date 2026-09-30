---
name: fabrik
description: Run the full Fabrik engineering workflow end-to-end in one shot - grill to PRD to plan to autonomous build loop. Use when the user says "run fabrik", "fabrik workflow", "start the factory", "back-to-back pipeline", or wants the complete Grill -> PRD -> Plan -> Build cycle rather than invoking the individual skills.
---

# Fabrik: End-to-End Workflow Runner

You are the master orchestrator for the Fabrik engineering workflow. Execute the phases below in order, in the current session, without asking the user to invoke additional commands. Each phase references agent-agnostic instruction files under `.fabrik/` (already on disk) so the same phases port to any agent runtime that can read files, spawn subagents, and run shell commands.

## Token Discipline (applies to every phase)

- **Caveman mode is ON by default** for all of `/fabrik`. Output only blunt fragments: no greetings, no sign-offs, no apologies, no "here is what I did", no "let me know". Code changes and commit messages ARE the explanation. Speak in short bullet points or code blocks only.
- **Use parallel Task subagents** whenever a phase calls for studying source/styleguide or executing a build task. Subagent results are compressed before injection (~60% smaller), keeping the main context lean across long build loops. Prefer the `explore` subagent for read-only study; `general` for implementation.
- **Named token tools** (invoke when useful, also exist on openclaude via the same community skills bundle):
  - `/caveman` - toggle compressed-output mode mid-session if prose drift appears.
  - `/caveman-commit` - auto-triggers when staging each per-task commit in Phase 3; produces terse Conventional Commits messages (subject <=50 chars, body only when the "why" isn't obvious).
- **Defer to auto-compaction.** Do NOT narrate progress. The phases are visible in the running todo list, not in prose. No "starting Phase X" preambles, no phase-end summaries unless the user explicitly asks.
- **Subagent seeding:** every spawned subagent inherits caveman rules via its own `PROMPT_*.md` / task file; do not restate the rules in the subagent prompt, just point at the file.

---

## Phase 0 - Verify Setup

Before anything else, confirm the Fabrik scaffolding exists. Use a file read or glob to check each required path:

- `.fabrik/` directory
- `.fabrik/docs/` directory
- `.fabrik/styleguide/STYLEGUIDE.md`
- `.fabrik/CONTEXT.md`
- `.fabrik/PROMPT_plan.md`
- `.fabrik/PROMPT_build.md`

If ANY are missing: stop. Print exactly this fragment and exit:

```
fabrik setup incomplete. run: ./.fabrik/setup.sh (or setup.ps1 on windows)
```

Do not attempt to auto-run setup. Do not list which path is missing.

Then determine the task destination (where `.fabrik/.tasks/` content is mirrored). Check for issue-tracker config in this order:

1. `docs/agents/issue-tracker.md` (written by `/setup-matt-pocock-skills`)
2. `.fabrik/issue-tracker.md` (fabrik-local override, same shape)

If neither exists, run the one-question onboarding inline (this is the only time the question is asked for this repo):

Print exactly:

```
where should fabrik publish tasks?
  1. local  - .fabrik/.tasks/*.md only
  2. github - mirror to GitHub issues (gh CLI)
  3. gitlab - mirror to GitLab issues (glab CLI)
  4. other  - describe your tracker in one line
```

Wait for the user's choice (1-4). For `other`, also ask for a one-line description of the workflow. Write the answer to `docs/agents/issue-tracker.md` (creates `docs/agents/` if missing) using the same template as `/setup-matt-pocock-skills` so subsequent mattpocock skills pick it up automatically — no need to run setup-matt-pocock-skills separately for the tracker question.

Parse the tracker type from the file's first line (`# Issue tracker: GitHub` / `: GitLab` / `: Local Markdown` / `: Other`). Store the detected type in your working memory for Phase 2 and Phase 3 branching.

Create a todo list with one item per phase (0 verify, 1 align, 2 plan, 3 build, 4 push), mark Phase 0 complete, move to Phase 1.

---

## Phase 1 - Align (always grill, prompt seeds)

Grilling ALWAYS runs — a user-supplied prompt seeds the first grill question rather than skipping the grill. The only exception is the explicit `--auto`/`-a` flag (see Mode A below) which opts into fully automated PRD generation with no interview.

1. Read `.fabrik/CONTEXT.md` and `.fabrik/styleguide/STYLEGUIDE.md` so the grill is grounded in the project's existing domain language and standards.
2. Determine if the user passed the `--auto`/`-a` flag (e.g. `/fabrik -a "prompt"` or `/fabrik --auto "prompt"`):
   - **Mode A - Auto (`--auto`/`-a` flag present, user explicitly opted out of grilling):** Run a single `general` subagent seeded with: "You are an expert product manager. Generate a detailed PRD and write it to `.fabrik/docs/PRD.md` based on these requirements: <user text>. Follow the structure of any existing `.fabrik/CONTEXT.md` for domain language. Caveman output: write the file, output nothing else." Skip the interview entirely.
   - **Default (no `--auto` flag, with or without a prompt) — Interactive grilling:** Start the interview. If the user supplied a prompt, use it as the seed — your first grill question should probe gaps in their stated prompt (missing success criteria? non-goals? ambiguous entities?). If no prompt, your first question should ask what they want to build. Interview the user ONE question at a time, stress-testing their plan against the codebase and guidelines. Mirror the behavior of the `/grill-with-docs` skill: each turn asks exactly one focused question, waits for the answer, then asks the next.
3. After each answer, append refined domain terms to `.fabrik/CONTEXT.md` when a new canonical term emerges.
4. Continue until the user signals alignment (e.g. "that's it", "good", "move on", "exit grill") OR you have covered: problem, success criteria, non-goals, key entities, open questions.
5. Then write the PRD: spawn a `general` subagent seeded with the grill transcript summary and "Write a detailed PRD to `.fabrik/docs/PRD.md` following the structure of `.fabrik/CONTEXT.md`. Caveman output: write the file, output nothing else."

### Exit gate

After writing `.fabrik/docs/PRD.md` in either mode: confirm the file exists and is non-empty. If the user only wanted alignment (they explicitly say "just align", "stop after prd", etc.), mark remaining todos cancelled and exit. Otherwise proceed to Phase 2.

---

## Phase 2 - Plan (PRD -> atomic task files)

Execute `.fabrik/PROMPT_plan.md` as the agent's instructions. Concretely:

1. Spawn parallel `explore` subagents:
   - One studies `src/*` (current implementation state).
   - One studies `.fabrik/styleguide/*` (coding standards).
   - One studies `.fabrik/docs/PRD.md` and any existing `.fabrik/.tasks/*.md`.
2. From the subagent results, perform the gap analysis described in `PROMPT_plan.md`.
3. Write discrete, atomic, non-overlapping `.fabrik/.tasks/*.md` files (e.g. `001-setup-db.md`). Each file = exactly one vertical slice, with checkable requirements as a markdown task list.
4. Branch by the tracker type stored in Phase 0:
   - **local** → nothing extra. `.fabrik/.tasks/` IS the source of truth.
   - **github** → for each task file, also run `gh issue create --title "<task title>" --body "$(cat <taskfile>)"` to mirror it. Prepend an `Issue: #<N>` line to the top of each task file with the returned issue number. `.fabrik/.tasks/` files remain the orchestrator's source of truth for ordering and completion state; GitHub is a read-friendly mirror.
   - **gitlab** → same as github but `glab issue create`.
   - **other** → follow the prose workflow in `docs/agents/issue-tracker.md`. If unverifiable, fall back to `local` (write `Issue: <unmirrored>` in the task file header).
5. Ensure `.fabrik/.tasks/completed/` exists; archive any already-done-but-unarchived tasks there.
6. Obey `PROMPT_plan.md` invariants strictly: NO src edits, NO commits, NO asking permission, NO presenting options - decide and execute. If PRD is missing/empty, write the single `.fabrik/.tasks/000-initialize-prd.md` task and exit.
7. Output only the list of created task filenames (with mirrored issue numbers when applicable). Nothing else.

Mark Phase 2 complete. Proceed to Phase 3.

---

## Phase 3 - Build Loop (AFK)

Ensure `.fabrik/.tasks/completed/` directory exists. Then loop:

1. Glob `.fabrik/.tasks/*.md` (top-level only, not `completed/`). Sort ascending by filename. Pick the lowest-numbered open task file.
2. If none remain: print `all tasks done` and break to Phase 4.
3. Spawn a `general` subagent seeded with:
   - The full body of `.fabrik/PROMPT_build.md` as its instructions.
   - The contents of the selected task file.
   - A reminder: "You implement ONLY this task. Run backpressure (npm run test && npm run lint && npm run build). On green, move the task file to `.fabrik/.tasks/completed/`. On repeated failure, append a `## Current Obstacles` section to the task file and exit. Caveman output: zero prose. Do NOT commit - the orchestrator commits."
4. On subagent return:
   - If the task file is now in `completed/`: stage all changes with `git add -A`, commit with a caveman-commit-style message (`feat: <task slug>`, subject <=50 chars, body only if the "why" is non-obvious). Use `/caveman-commit` to generate the message if available. Then branch by tracker type to mirror completion:
     - **local** → nothing extra.
     - **github** → parse the `Issue: #<N>` line from the task file, run `gh issue close <N> --comment "done: <commit subject>"`. Skip silently if the line is `<unmirrored>` or missing.
     - **gitlab** → `glab issue close <N> --comment "done: <commit subject>"`.
     - **other** → follow the prose workflow in `docs/agents/issue-tracker.md` for marking a ticket done. Skip silently if unscriptable.
   - If the task file is still in `.fabrik/.tasks/` (obstacles): do NOT commit, do NOT close the mirrored issue. Print `<taskname>: obstacles - see task file`. Optionally `gh issue comment <N> --body "obstacle: <one-line>"` if github-tracked, to keep the tracker honest. Continue to the next iteration; the loop will retry it next cycle unless the user interrupts.
5. Repeat from step 1.

Critical invariants during the loop:
- ONE task per iteration. Never edit more.
- Never skip backpressure. If tests/lint/build fail, the subagent must self-correct before archiving.
- Never stage/commit on a failed task.

Mark Phase 3 complete when `.fabrik/.tasks/` is empty.

---

## Phase 4 - Push

1. Get the current branch: `git branch --show-current`.
2. `git push origin <branch>`.
3. Print: `fabrik done. pushed <branch>`.

Mark Phase 4 complete. Exit.

---

## Portability Note

This body references tool capabilities (read file, write file, spawn subagent, run shell command, glob) rather than agent-specific tool names. The reusable instruction files `.fabrik/PROMPT_plan.md` and `.fabrik/PROMPT_build.md` are already agent-agnostic (plain markdown).

To port to another runtime (e.g. Claude Code / "openclaude"):
1. Copy this `SKILL.md` body into the target agent's agent-file format (e.g. `.claude/agents/fabrik.md` for Claude Code).
2. Map the four capability verbs to the target's tools: read -> Read, write -> Edit/Write, subagent -> Task tool, shell -> Bash.
3. The named token tools (`/caveman`, `/caveman-commit`) are part of the same community skills bundle (`mattpocock/skills`) and install identically on both runtimes.
4. The issue-tracker integration reads `docs/agents/issue-tracker.md` (same convention mattpocock skills use). GitHub `gh` and GitLab `glab` CLIs are portable shell tools. For `other` trackers, Phase 2/3 fall back to the prose workflow written by `/setup-matt-pocock-skills`.

No logic changes required.