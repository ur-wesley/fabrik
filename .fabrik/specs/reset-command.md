# Spec: Fabrik Reset Command (`fabrik reset`)

## Summary
When `fabrik init` (or `setup`) runs in a repository, it configures client applications (Cursor, OpenCode, Pi) and the repository root:
- Generates `opencode.json` with Fabrik's agent roster (`plan`, `build`, `orchestrator`, etc.) and `default_agent: "plan"`.
- Installs thin shims or full rules/skills in `.cursor/rules/`, `.cursor/agents/`, `.opencode/skills/`, `.pi/skills/`, and `.pi/agent/agents/`.
- Appends or creates the `## Fabrik workflow` section in root `AGENTS.md`.
- Scaffolds `.fabrik/` with `config.yaml`, `docs/`, and `specs/`.

This spec introduces `fabrik reset [path]`, a command to cleanly remove Fabrik's project-level configuration and wiring so that client applications and the repository immediately revert to their default behavior.

## CLI Interface
```
Usage:
  fabrik reset [path] [flags]

Flags:
  -y, --yes              answer yes to all prompts (interactive confirmation by default)
      --non-interactive  alias for --yes (CI)
      --dry-run          print actions without changing anything
      --beads            remove beads integration and .beads as well (skips prompt)
      --no-beads         keep beads integration and .beads (skips prompt)
      --repo string      repo path (default: current working directory)
```

## Behavior and Deletion Rules

### 1. OpenCode Configuration (`opencode.json` & `.opencode/`)
- If `opencode.json` was created by Fabrik (matches embedded template or defines Fabrik agents with `default_agent: "plan"`):
  - Delete `opencode.json` completely, restoring OpenCode's default agent behavior.
  - If `opencode.json` contains custom user agents or keys, remove Fabrik agent entries and remove `default_agent: "plan"`.
- Remove all Fabrik skill files in `.opencode/skills/*.md`.
- Remove legacy `.opencode/agents/*.md` if present.
- Prune empty directories: `.opencode/skills`, `.opencode/agents`, `.opencode`.

### 2. Cursor Configuration (`.cursor/`)
- Remove all Fabrik skill rules in `.cursor/rules/*.mdc` (and legacy `.cursor/rules/*.md`).
- Remove all Fabrik agent files in `.cursor/agents/*.md`.
- Remove legacy `.cursor/rules/agents/` if present.
- If beads removal is confirmed/requested: unwire beads integration (remove `beads.mdc` and `hooks.json`).
- Prune empty directories: `.cursor/agents`, `.cursor/rules/agents`, `.cursor/rules`, `.cursor`.

### 3. Pi Configuration (`.pi/`)
- Remove all Fabrik skill files in `.pi/skills/*.md`.
- Remove all Fabrik agent files in `.pi/agent/agents/*.md`.
- Prune empty directories: `.pi/agent/agents`, `.pi/agent`, `.pi/skills`, `.pi`.

### 4. Root `AGENTS.md`
- Strip the `## Fabrik workflow` section from `AGENTS.md`.
- If beads removal is confirmed/requested: also strip the beads integration block (`<!-- BEGIN BEADS INTEGRATION ... --> ... <!-- END BEADS INTEGRATION -->`).
- If the remaining content is empty or consists only of `# Agent instructions`, delete `AGENTS.md`.
- If `AGENTS.md` contains other user content, preserve the file with the stripped sections.

### 5. Hub Directory (`.fabrik/`)
- Remove the entire `.fabrik/` directory completely, including all configs, legacy dumps, specs, and docs.

### 6. Beads Removal (`.beads/` & hooks)
- When running interactively (unless `--beads` or `--no-beads` is specified), prompt the user:
  `Remove Beads integration and .beads as well?`
- If confirmed or `--beads` is passed:
  - Invoke `bd setup cursor --remove` and `bd setup opencode --remove` (if `bd` is available on PATH).
  - Remove `.cursor/rules/beads.mdc` and `.cursor/hooks.json`.
  - Remove directory `.beads/`.
  - Remove `.beads.gate.lock` if present.
- If declined or `--no-beads` is passed, Beads is left untouched.

### 7. Interactive Confirmation & Dry Run
- If interactive and not `--yes`/`-y`: prompts `Reset Fabrik configuration in <path>?`.
- If `--dry-run`: previews all file deletions, modifications, and directory removals without making any disk modifications.

## Package Architecture
- `cli/internal/reseter`: core logic (`Config`, `Deps`, `Plan(repo, cfg)`, `Reset(ctx, cfg, deps)`).
- `cli/cmd/fabrik/reset.go`: Cobra command registration and flag binding.
- `cli/cmd/fabrik/root.go`: Register `newResetCmd()` in root.
- `cli/internal/reseter/reseter_test.go`: unit tests verifying round-trip reset, preserving user files, dry-run, AGENTS.md stripping, beads prompt and removal, etc.
