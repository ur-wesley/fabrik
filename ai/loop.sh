#!/bin/bash
# loop.sh
# Bash loop runner for the AI workflow loop using OpenCode.
# Usage: ./ai/loop.sh [plan|build] [max_iterations]

MODE=${1:-"build"}
MAX_ITERATIONS=${2:-0}
ITERATION=0
CURRENT_BRANCH=$(git branch --show-current 2>/dev/null || echo "main")

# Resolve paths relative to script's directory (ai/)
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
PROMPT_FILE="${SCRIPT_DIR}/PROMPT_${MODE}.md"
TASKS_DIR="${SCRIPT_DIR}/.tasks"

if [ ! -f "$PROMPT_FILE" ]; then
    echo "Error: Prompt file $PromptFile not found."
    exit 1
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Agent:  OpenCode"
echo "Mode:   $MODE"
echo "Prompt: $PROMPT_FILE"
echo "Branch: $CURRENT_BRANCH"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Initialize task folders
mkdir -p "${TASKS_DIR}/completed"

while true; do
    if [ $MAX_ITERATIONS -gt 0 ] && [ $ITERATION -ge $MAX_ITERATIONS ]; then
        echo "Reached max iterations limit: $MAX_ITERATIONS"
        break
    fi

    # If in building mode, verify tasks exist
    if [ "$MODE" = "build" ]; then
        # Count open markdown task files (excluding subfolders)
        OPEN_TASKS=$(find "$TASKS_DIR" -maxdepth 1 -name "*.md" | wc -l)
        if [ "$OPEN_TASKS" -eq 0 ]; then
            echo "🎉 No remaining open tasks in $TASKS_DIR. Work complete!"
            break
        fi
        echo "Pending tasks: $OPEN_TASKS"
    fi

    echo -e "\n[Iteration $((ITERATION + 1))] Starting OpenCode Session..."

    # Run OpenCode CLI
    cat "$PROMPT_FILE" | opencode --agent "$MODE"

    # Push progress
    echo "Syncing remote repository..."
    git push origin "$CURRENT_BRANCH" || echo "Remote push skipped or failed."

    ITERATION=$((ITERATION + 1))

    # Planning mode runs only once to generate tasks
    if [ "$MODE" = "plan" ]; then
        echo "Planning phase complete. Start the build loop with: ./ai/loop.sh build"
        break
    fi

    echo -e "\n======================== TASK CYCLE $ITERATION COMPLETE ========================\n"
done
