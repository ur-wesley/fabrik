#!/bin/bash
# fabrik.sh
# Master Orchestrator for the Fabrik workflow.
# Run this script to execute the entire workflow back-to-back:
# Grill Session (Interactive) -> PRD-to-Issues (Auto) -> Build Loops (Auto)
# Usage: ./fabrik.sh

# Resolve directory paths relative to script
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
FABRIK_DIR="${SCRIPT_DIR}/.fabrik"
TASKS_DIR="${FABRIK_DIR}/.tasks"
PLAN_PROMPT_FILE="${FABRIK_DIR}/PROMPT_plan.md"
BUILD_PROMPT_FILE="${FABRIK_DIR}/PROMPT_build.md"

clear
echo -e "\033[36m🏭 Starting the Fabrik Workflow...\033[0m"

# ----------------------------------------------------
# Phase 1: Interactive Requirements Alignment
# ----------------------------------------------------
echo -e "\n\033[37m====================================================\033[0m"
echo -e "\033[36mStep 1: Interactive Alignment Session\033[0m"
echo -e "\033[37m====================================================\033[0m"
echo "1. Type /grill-with-docs to stress-test your plan."
echo "2. Type /to-prd to write the requirements to .fabrik/docs/PRD.md."
echo "3. Type /exit to hand over control to the automated pipeline."
read -p "Press [Enter] when ready to launch the OpenCode TUI..."

opencode

# ----------------------------------------------------
# Phase 2: Automated Planning (PRD to Issues)
# ----------------------------------------------------
echo -e "\n\033[37m====================================================\033[0m"
echo -e "\033[36mStep 2: Running Automated Planning (PRD-to-Issues)...\033[0m"
echo -e "\033[37m====================================================\033[0m"

if [ ! -f "$PLAN_PROMPT_FILE" ]; then
    echo "Error: Prompt file $PLAN_PROMPT_FILE not found."
    exit 1
fi

PLAN_PROMPT_TEXT=$(cat "$PLAN_PROMPT_FILE")
opencode run --agent plan "$PLAN_PROMPT_TEXT"

# ----------------------------------------------------
# Phase 3: Automated Feature Building (AFK Loop)
# ----------------------------------------------------
echo -e "\n\033[37m====================================================\033[0m"
echo -e "\033[36mStep 3: Starting Autonomous Feature Assembly...\033[0m"
echo -e "\033[37m====================================================\033[0m"

if [ ! -f "$BUILD_PROMPT_FILE" ]; then
    echo "Error: Prompt file $BUILD_PROMPT_FILE not found."
    exit 1
fi

BUILD_PROMPT_TEXT=$(cat "$BUILD_PROMPT_FILE")
CURRENT_BRANCH=$(git branch --show-current 2>/dev/null || echo "main")

Iteration=0
while true; do
    # Count open markdown task files (excluding subfolders)
    OPEN_TASKS=$(find "$TASKS_DIR" -maxdepth 1 -name "*.md" | wc -l)
    if [ "$OPEN_TASKS" -eq 0 ]; then
        echo -e "\n\033[32m🎉 All tasks implemented, verified, and committed! Work complete!\033[0m"
        break
    fi

    echo -e "\n\033[32m[Iteration $((Iteration + 1))] Executing next task... (Pending tasks: $OPEN_TASKS)\033[0m"
    
    opencode run --agent build "$BUILD_PROMPT_TEXT"

    # Push progress
    echo "Syncing remote branch..."
    git push origin "$CURRENT_BRANCH" || echo "Git push skipped."

    Iteration=$((Iteration + 1))
done
