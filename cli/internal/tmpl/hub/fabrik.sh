#!/usr/bin/env bash
set -euo pipefail

AUTO=false
INITIAL_PROMPT=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -a|--auto) AUTO=true; shift ;;
    -p|--prompt) INITIAL_PROMPT="$2"; shift 2 ;;
    *) echo "Unknown parameter: $1"; exit 1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

SETUP_INCOMPLETE=false
for item in docs styleguide CONTEXT.md; do
  if [[ ! -e "${SCRIPT_DIR}/${item}" ]]; then
    SETUP_INCOMPLETE=true
    break
  fi
done

if [[ ! -d "${REPO_ROOT}/.beads" ]]; then
  SETUP_INCOMPLETE=true
fi

if [[ "$SETUP_INCOMPLETE" == true ]]; then
  echo "Fabrik setup incomplete."
  echo "Run: ./install/setup.sh then ./install/init.sh ."
  exit 1
fi

clear || true
echo "Starting Fabrik workflow..."

if [[ "$AUTO" == true ]]; then
  if [[ -z "$INITIAL_PROMPT" ]]; then
    read -r -p "Initial prompt for PRD generation: " INITIAL_PROMPT
  fi
  opencode run "You are an expert product manager. Generate a detailed PRD in .fabrik/docs/PRD.md based on these requirements: $INITIAL_PROMPT"
else
  echo "Step 1: /grill-with-docs, /to-prd, /exit"
  read -r -p "Press enter to launch OpenCode TUI..."
  opencode
fi

echo "Step 2: Planning..."
bash "${SCRIPT_DIR}/loop.sh" plan

echo "Step 3: Build loop..."
bash "${SCRIPT_DIR}/loop.sh" build
