#!/bin/bash
# loop.sh
# Bash loop runner for the AI workflow loop using OpenCode.
# Includes real-time execution tracking, runtime timers, and a live state dashboard.
# Usage: ./.fabrik/loop.sh [plan|build] [max_iterations]

MODE=${1:-"build"}
MAX_ITERATIONS=${2:-0}
ITERATION=0
CURRENT_BRANCH=$(git branch --show-current 2>/dev/null || echo "main")

# Resolve paths relative to script's directory (.fabrik/)
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
PROMPT_FILE="${SCRIPT_DIR}/PROMPT_${MODE}.md"
TASKS_DIR="${SCRIPT_DIR}/.tasks"
STATE_FILE="${SCRIPT_DIR}/state.md"

if [ ! -f "$PROMPT_FILE" ]; then
    echo "Error: Prompt file $PROMPT_FILE not found."
    exit 1
fi

# Initialize task folders
mkdir -p "${TASKS_DIR}/completed"

# Clear/Initialize the live state file
START_TIME_STR=$(date "+%Y-%m-%d %H:%M:%S")
cat << EOF > "$STATE_FILE"
# Fabrik Loop Status: INITIALIZING

*   **Started At:** $START_TIME_STR
*   **Mode:** \`$MODE\`
*   **Branch:** \`$CURRENT_BRANCH\`
*   **Status:** \`Running Setup...\`
EOF

while true; do
    if [ $MAX_ITERATIONS -gt 0 ] && [ $ITERATION -ge $MAX_ITERATIONS ]; then
        echo "Reached max iterations limit: $MAX_ITERATIONS"
        break
    fi

    # Fetch pending tasks and find the next task details
    OPEN_TASKS=0
    NEXT_TASK_NAME="None"
    NEXT_TASK_DESC="No description available."
    
    if [ -d "$TASKS_DIR" ]; then
        OPEN_TASKS=$(find "$TASKS_DIR" -maxdepth 1 -name "*.md" | wc -l)
    fi

    # If in building mode, verify tasks exist
    if [ "$MODE" = "build" ]; then
        if [ "$OPEN_TASKS" -eq 0 ]; then
            echo -e "\n🎉 No remaining open tasks. Work complete!"
            END_TIME_STR=$(date "+%Y-%m-%d %H:%M:%S")
            cat << EOF > "$STATE_FILE"
# Fabrik Loop Status: FINISHED / IDLE

*   **Last Run Complete:** $END_TIME_STR
*   **Outcome:** All tasks executed successfully!
*   **Status:** \`IDLE\`
EOF
            break
        fi
        
        # Identify the next task file (sorted alphabetically)
        NEXT_TASK_FILE=$(find "$TASKS_DIR" -maxdepth 1 -name "*.md" | sort | head -n 1)
        NEXT_TASK_NAME=$(basename "$NEXT_TASK_FILE")
        
        # Read the first non-header line for description
        while IFS= read -r line; do
            trimmed=$(echo "$line" | xargs)
            if [ -n "$trimmed" ] && [[ ! "$trimmed" =~ ^# ]]; then
                NEXT_TASK_DESC="$trimmed"
                break
            fi
        done < "$NEXT_TASK_FILE"
    else
        NEXT_TASK_NAME="Planning Phase (Gap Analysis & PRD Triage)"
        NEXT_TASK_DESC="Comparing docs/PRD.md against the src/ directory to generate task issues."
    fi

    CYCLE_START=$(date +%s)
    CYCLE_START_STR=$(date "+%H:%M:%S")

    # Update terminal screen dashboard
    clear
    echo "===================================================="
    echo "🏭 FABRIK DASHBOARD (Iteration: $((ITERATION + 1)))"
    echo "===================================================="
    echo "Mode:         $MODE"
    echo "Branch:       $CURRENT_BRANCH"
    echo -e "Target Task:  \033[33m$NEXT_TASK_NAME\033[0m"
    echo "Description:  $NEXT_TASK_DESC"
    echo "Started At:   $CYCLE_START_STR"
    echo "Remaining:    $OPEN_TASKS tasks"
    echo "===================================================="
    echo -e "Status: \033[35mRunning OpenCode agent run...\033[0m"

    # Update the live state file
    cat << EOF > "$STATE_FILE"
# Fabrik Loop Status: RUNNING 🔄

*   **Last Update:** $(date "+%Y-%m-%d %H:%M:%S")
*   **Iteration:** $((ITERATION + 1))
*   **Current Task:** \`$NEXT_TASK_NAME\`
*   **Description:** $NEXT_TASK_DESC
*   **Cycle Started At:** $CYCLE_START_STR
*   **Pending Queue:** $OPEN_TASKS tasks
*   **Status:** \`Active - Executing OpenCode Run\`
EOF

    # Read the prompt file contents into a variable
    PROMPT_TEXT=$(cat "$PROMPT_FILE")

    # Run OpenCode run command
    opencode run --agent "$MODE" "$PROMPT_TEXT"

    # Calculate runtime duration
    CYCLE_END=$(date +%s)
    DURATION=$((CYCLE_END - CYCLE_START))
    MIN=$((DURATION / 60))
    SEC=$((DURATION % 60))
    DURATION_STR="${MIN}m ${SEC}s"

    echo "Syncing remote branch..."
    git push origin "$CURRENT_BRANCH" || echo "Git push skipped."

    # Write completion details to screen
    echo -e "\033[32mTask complete! Duration: $DURATION_STR\033[0m"

    # Log task completion in state file
    LOG_ENTRY="*   [$(date "+%H:%M:%S")] Completed \`$NEXT_TASK_NAME\` in $DURATION_STR"
    
    # Append log entry to state file
    if ! grep -q "## Recent Run Log" "$STATE_FILE"; then
        echo -e "\n## Recent Run Log" >> "$STATE_FILE"
    fi
    echo "$LOG_ENTRY" >> "$STATE_FILE"

    # Update state header to sleeping
    sed -i 's/# Fabrik Loop Status: RUNNING.*/# Fabrik Loop Status: SLEEP \/ SYNCING 😴/g' "$STATE_FILE"
    sed -i "s/\*   \*\*Status:\*\*/\*   \*\*Status:\*\* \`Waiting for next iteration (Last cycle took $DURATION_STR)\`/g" "$STATE_FILE"

    ITERATION=$((ITERATION + 1))

    # Planning mode runs only once to generate tasks
    if [ "$MODE" = "plan" ]; then
        echo "Planning phase complete. Start the build loop with: ./.fabrik/loop.sh build"
        break
    fi

    echo -e "\n======================== TASK CYCLE $ITERATION COMPLETE ========================\n"
    sleep 3
done
