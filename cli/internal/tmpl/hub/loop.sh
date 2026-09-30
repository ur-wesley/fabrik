#!/usr/bin/env bash
set -euo pipefail

MODE="${1:-build}"
MAX_ITERATIONS="${2:-0}"
ITERATION=0
CURRENT_BRANCH="$(git branch --show-current 2>/dev/null || echo main)"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROMPT_FILE="${SCRIPT_DIR}/PROMPT_${MODE}.md"
STATE_FILE="${SCRIPT_DIR}/state.md"

if [[ ! -f "$PROMPT_FILE" ]]; then
  echo "Error: Prompt file $PROMPT_FILE not found."
  exit 1
fi

if ! command -v bd >/dev/null 2>&1; then
  echo "bd not found. Run install/setup.sh first."
  exit 1
fi

cat > "$STATE_FILE" <<EOF
# Fabrik Loop Status: INITIALIZING

*   **Started At:** $(date "+%Y-%m-%d %H:%M:%S")
*   **Mode:** $MODE
*   **Branch:** $CURRENT_BRANCH
EOF

while true; do
  if [[ $MAX_ITERATIONS -gt 0 && $ITERATION -ge $MAX_ITERATIONS ]]; then
    echo "Reached max iterations: $MAX_ITERATIONS"
    break
  fi

  OPEN_JSON="$(bd list --status=open --json 2>/dev/null || echo '[]')"
  CLOSED_JSON="$(bd list --status=closed --json --limit 100 2>/dev/null || echo '[]')"
  OPEN_COUNT="$(printf '%s' "$OPEN_JSON" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')"
  CLOSED_COUNT="$(printf '%s' "$CLOSED_JSON" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))')"
  READY_JSON="$(bd ready --json 2>/dev/null || echo '[]')"

  NEXT_ID="plan"
  NEXT_DESC="PRD gap analysis and bd create"

  if [[ "$MODE" == "build" ]]; then
    if [[ "$OPEN_COUNT" -eq 0 ]]; then
      echo "No open Beads issues."
      break
    fi
    READY_COUNT="$(printf '%s' "$READY_JSON" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))' 2>/dev/null || echo 0)"
    if [[ "$READY_COUNT" -eq 0 ]]; then
      echo "Open issues exist but none are ready."
      break
    fi
    NEXT_ID="$(printf '%s' "$READY_JSON" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d[0]["id"] if d else "")')"
    NEXT_DESC="$(printf '%s' "$READY_JSON" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d[0]["title"] if d else "")')"
  fi

  clear || true
  echo "===================================================="
  echo "FABRIK $MODE (iteration $((ITERATION + 1)))"
  echo "Open: $OPEN_COUNT | Closed: $CLOSED_COUNT"
  echo "Target: $NEXT_ID — $NEXT_DESC"
  echo "===================================================="

  PROMPT_TEXT="$(cat "$PROMPT_FILE")"
  opencode run "$PROMPT_TEXT"

  git push origin "$CURRENT_BRANCH" 2>/dev/null || true

  ITERATION=$((ITERATION + 1))
  if [[ "$MODE" == "plan" ]]; then
    echo "Planning complete. Run: ./.fabrik/loop.sh build"
    break
  fi
  sleep 3
done
