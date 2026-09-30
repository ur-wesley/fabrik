#!/usr/bin/env bash
set -euo pipefail

# Fabrik repo init (mac/linux). Windows: install/init.ps1 (parity).
# Sets up .fabrik overview hub + Cursor/Pi/OpenCode only. Never other agents.

INSTALL_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FABRIK_ROOT="$(cd "$INSTALL_ROOT/.." && pwd)"
REPO_PATH="${1:-$(pwd)}"
SKIP_CHECKS=false
if [[ "${2:-}" == "--skip-checks" ]]; then SKIP_CHECKS=true; fi

if [[ "$SKIP_CHECKS" != true ]]; then
  bash "$INSTALL_ROOT/check.sh" || echo "(advisory: optional tools missing — continuing, bd is required)"
  if ! command -v bd >/dev/null 2>&1; then
    echo "bd not found. Run ./install/setup.sh first." >&2
    exit 1
  fi
fi

if ! command -v bd >/dev/null 2>&1; then
  echo "bd not found. Run ./install/setup.sh first." >&2
  exit 1
fi

cd "$REPO_PATH"

# 1. Beads (todos) — no third-party agents
if [[ -d .beads ]]; then
  echo "Beads already initialized in $REPO_PATH"
else
  bd init --non-interactive --skip-agents --skip-hooks -q
  echo "Beads initialized"
fi

# 2. .fabrik overview hub
mkdir -p .fabrik/docs .fabrik/specs .fabrik/styleguide .fabrik/agents

copy_if_missing() {
  if [[ -f "$1" && ! -f "$2" ]]; then cp "$1" "$2"; fi
}

# runners from this repo's .fabrik (dev source)
for file in PROMPT_plan.md PROMPT_build.md loop.sh fabrik.sh AGENTS.md; do
  copy_if_missing "${FABRIK_ROOT}/.fabrik/${file}" ".fabrik/${file}"
done
# hub files from install templates (canonical)
copy_if_missing "$INSTALL_ROOT/templates/fabrik/README.md" ".fabrik/README.md"
copy_if_missing "$INSTALL_ROOT/templates/fabrik/CONTEXT.md" ".fabrik/CONTEXT.md"
copy_if_missing "$INSTALL_ROOT/templates/fabrik/config.yaml" ".fabrik/config.yaml"
copy_if_missing "$INSTALL_ROOT/templates/fabrik/skills.md" ".fabrik/skills.md"
if [[ ! -f .fabrik/.gitignore && -f "$INSTALL_ROOT/templates/fabrik/gitignore" ]]; then
  cp "$INSTALL_ROOT/templates/fabrik/gitignore" ".fabrik/.gitignore"
fi
if [[ -f "${FABRIK_ROOT}/.fabrik/styleguide/STYLEGUIDE.md" && ! -f .fabrik/styleguide/STYLEGUIDE.md ]]; then
  cp "${FABRIK_ROOT}/.fabrik/styleguide/STYLEGUIDE.md" .fabrik/styleguide/STYLEGUIDE.md
fi
# subagent defs live in hub
for a in orchestrator explore researcher briefer planner builder tester reviewer style-smells security; do
  copy_if_missing "$INSTALL_ROOT/templates/agents/${a}.md" ".fabrik/agents/${a}.md"
done

# 2b. OpenCode config replaces built-in build with Fabrik roster (file refs, no .opencode/agents copies)
copy_if_missing "$INSTALL_ROOT/templates/opencode.json" "$REPO_PATH/opencode.json"

# 3. Skills -> 3 apps only
SKILLS="i-have-adhd caveman ponytail rtk-usage tdd diagnose guardrails"
mkdir -p .cursor/rules .opencode/skills .pi/skills
for s in $SKILLS; do
  src=""
  if [[ -f "$FABRIK_ROOT/skills/${s}.md" ]]; then src="$FABRIK_ROOT/skills/${s}.md"; fi
  [[ -z "$src" ]] && continue
  # cursor rules need .mdc frontmatter; reuse md body with minimal header
  if [[ ! -f ".cursor/rules/${s}.mdc" ]]; then
    { echo "---"; echo "description: Fabrik skill $s"; echo "alwaysApply: false"; echo "---"; echo ""; cat "$src"; } > ".cursor/rules/${s}.mdc"
  fi
  copy_if_missing "$src" ".opencode/skills/${s}.md"
  copy_if_missing "$src" ".pi/skills/${s}.md"
done

# 4. Subagents -> Pi + Cursor copies (OpenCode uses opencode.json file refs)
mkdir -p ".pi/agent/agents" ".cursor/rules/agents"
for a in orchestrator explore researcher briefer planner builder tester reviewer style-smells security; do
  src="$INSTALL_ROOT/templates/agents/${a}.md"
  copy_if_missing "$src" ".pi/agent/agents/${a}.md"
  if [[ ! -f ".cursor/rules/agents/${a}.mdc" ]]; then
    { echo "---"; echo "description: Fabrik subagent $a"; echo "alwaysApply: false"; echo "---"; echo ""; cat "$src"; } > ".cursor/rules/agents/${a}.mdc"
  fi
done

# 5. Per-repo agent wiring, 3 apps only (never .claude/.codex/.agents)
had_agents=false; [[ -e .agents ]] && had_agents=true
had_claude=false; [[ -e .claude ]] && had_claude=true
had_codex=false; [[ -e .codex ]] && had_codex=true
if command -v bd >/dev/null 2>&1; then bd setup cursor >/dev/null 2>&1 || true; bd setup opencode >/dev/null 2>&1 || true; fi
if command -v engram >/dev/null 2>&1; then
  engram setup cursor >/dev/null 2>&1 || true
  engram setup opencode >/dev/null 2>&1 || true
  engram setup pi >/dev/null 2>&1 || true
fi
# bd/engram may side-effect generic dirs; remove only if we didn't have them (3 apps only)
[[ "$had_agents" == false && -e .agents ]] && rm -rf .agents
[[ "$had_claude" == false && -e .claude ]] && rm -rf .claude
[[ "$had_codex" == false && -e .codex ]] && rm -rf .codex

# 6. Root AGENTS.md workflow block (strip Codex block if present)
WORKFLOW_BLOCK="$(cat "$INSTALL_ROOT/templates/workflow-note.md")"
if [[ -f AGENTS.md ]]; then
  if ! grep -q 'Fabrik workflow' AGENTS.md; then
    { echo ""; echo "## Fabrik workflow"; echo ""; printf '%s\n' "$WORKFLOW_BLOCK"; } >> AGENTS.md
    echo "Appended Fabrik workflow to AGENTS.md"
  fi
else
  { echo "# Agent instructions"; echo ""; echo "## Fabrik workflow"; echo ""; printf '%s\n' "$WORKFLOW_BLOCK"; } > AGENTS.md
  echo "Created AGENTS.md"
fi

echo ""
echo "Done. Apps: Cursor, OpenCode, Pi. Hub: .fabrik/README.md"
echo "Next: grill, bd create issues, bd ready, claim, close."
