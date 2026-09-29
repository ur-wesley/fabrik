#!/usr/bin/env bash
set -euo pipefail

INSTALL_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FABRIK_ROOT="$(cd "$INSTALL_ROOT/.." && pwd)"
REPO_PATH="${1:-$(pwd)}"

if ! command -v bd >/dev/null 2>&1; then
  echo "bd not found. Run ./install/setup.sh first." >&2
  exit 1
fi

cd "$REPO_PATH"

if [[ -d .beads ]]; then
  echo "Beads already initialized in $REPO_PATH"
else
  bd init --non-interactive --skip-agents --skip-hooks -q
  echo "Beads initialized (embedded Dolt in .beads/)"
  echo "Run bd setup cursor (and engram setup) from install/setup if needed."
fi

mkdir -p .fabrik/docs .fabrik/specs .fabrik/styleguide

TEMPLATE_DIR="${FABRIK_ROOT}/.fabrik"
for file in PROMPT_plan.md PROMPT_build.md loop.ps1 loop.sh fabrik.ps1 fabrik.sh AGENTS.md setup.ps1 setup.sh CONTEXT.md; do
  if [[ -f "${TEMPLATE_DIR}/${file}" && ! -f ".fabrik/${file}" ]]; then
    cp "${TEMPLATE_DIR}/${file}" ".fabrik/${file}"
  fi
done
if [[ -f "${TEMPLATE_DIR}/styleguide/STYLEGUIDE.md" && ! -f .fabrik/styleguide/STYLEGUIDE.md ]]; then
  cp "${TEMPLATE_DIR}/styleguide/STYLEGUIDE.md" .fabrik/styleguide/STYLEGUIDE.md
fi

if [[ ! -f .fabrik/config.yaml ]]; then
  if [[ -f "$FABRIK_ROOT/.fabrik/config.yaml" ]]; then
    cp "$FABRIK_ROOT/.fabrik/config.yaml" .fabrik/config.yaml
  else
    cat > .fabrik/config.yaml <<'EOF'
agent: pi

models:
  default: inherit

session:
  auto_lock: true
  auto_commit: true

tools:
  rtk: true
  engram: true
  caveman: true
  ponytail: true

skills:
  - caveman
  - ponytail
  - grill-with-docs
  - to-prd
  - to-issues
  - tdd
EOF
  fi
fi

WORKFLOW_BLOCK="$(cat "$INSTALL_ROOT/templates/workflow-note.md")"
if [[ -f AGENTS.md ]]; then
  if ! grep -q 'Fabrik workflow' AGENTS.md; then
    {
      echo ""
      echo "## Fabrik workflow"
      echo ""
      printf '%s\n' "$WORKFLOW_BLOCK"
    } >> AGENTS.md
    echo "Appended Fabrik workflow to AGENTS.md"
  fi
else
  {
    echo "# Agent instructions"
    echo ""
    echo "## Fabrik workflow"
    echo ""
    printf '%s\n' "$WORKFLOW_BLOCK"
  } > AGENTS.md
  echo "Created AGENTS.md"
fi

echo ""
echo "Optional: graphify . when search is slow."
echo "Next: grill, bd create issues, bd ready, claim, close."
