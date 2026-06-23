#!/bin/bash
# fabrik.sh
# Master Orchestrator for the Fabrik workflow.
# Run this script to execute the entire workflow back-to-back:
# Grill Session (Interactive) -> PRD-to-Issues (Auto) -> Build Loops (Auto)
# Usage: ./.fabrik/fabrik.sh [-a|--auto] [-p|--prompt "Prompt text"]

AUTO=false
INITIAL_PROMPT=""

while [[ "$#" -gt 0 ]]; do
    case $1 in
        -a|--auto) AUTO=true; shift ;;
        -p|--prompt) INITIAL_PROMPT="$2"; shift 2 ;;
        *) echo "Unknown parameter: $1"; exit 1 ;;
    esac
done

# Resolve directory paths relative to script
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
FABRIK_DIR="$SCRIPT_DIR"

# ----------------------------------------------------
# Setup Verification Check
# ----------------------------------------------------
SETUP_INCOMPLETE=false
REQUIRED_ITEMS=(".tasks" "docs" "styleguide" "styleguide/STYLEGUIDE.md" "CONTEXT.md")

for item in "${REQUIRED_ITEMS[@]}"; do
    if [ ! -e "${FABRIK_DIR}/${item}" ]; then
        SETUP_INCOMPLETE=true
        break
    fi
done

if [ "$SETUP_INCOMPLETE" = true ]; then
    clear
    echo -e "\033[31m⚠️ Fabrik setup is incomplete or has not been run in this directory.\033[0m"
    echo -e "\033[33mPlease execute the setup script to initialize the framework and skills:\033[0m"
    echo -e "\033[33m    ./.fabrik/setup.sh\033[0m"
    echo ""
    exit 1
fi

clear
echo -e "\033[36m🏭 Starting the Fabrik Workflow...\033[0m"

# ----------------------------------------------------
# Phase 1: Interactive Requirements Alignment
# ----------------------------------------------------
if [ "$AUTO" = true ]; then
    echo -e "\n\033[37m====================================================\033[0m"
    echo -e "\033[36mStep 1: Automated Requirements Alignment\033[0m"
    echo -e "\033[37m====================================================\033[0m"
    
    if [ -z "$INITIAL_PROMPT" ]; then
        read -p "Please enter the initial prompt for PRD generation: " INITIAL_PROMPT
    fi

    echo -e "\033[33mRunning Automated PRD Generation...\033[0m"
    opencode run "You are an expert product manager. Generate a detailed PRD in docs/PRD.md based on these requirements: $INITIAL_PROMPT"
else
    echo -e "\n\033[37m====================================================\033[0m"
    echo -e "\033[36mStep 1: Interactive Alignment Session\033[0m"
    echo -e "\033[37m====================================================\033[0m"
    echo "1. Type /grill-with-docs to stress-test your plan."
    echo "2. Type /to-prd to write the requirements to docs/PRD.md or .fabrik/docs/PRD.md."
    echo "3. Type /exit to hand over control to the automated pipeline."
    read -p "Press [Enter] when ready to launch the OpenCode TUI..."

    opencode
fi

# ----------------------------------------------------
# Phase 2: Automated Planning (PRD to Issues)
# ----------------------------------------------------
echo -e "\n\033[37m====================================================\033[0m"
echo -e "\033[36mStep 2: Running Automated Planning (PRD-to-Issues)...\033[0m"
echo -e "\033[37m====================================================\033[0m"

# Delegate execution directly to loop.sh
bash "${SCRIPT_DIR}/loop.sh" plan

# ----------------------------------------------------
# Phase 3: Automated Feature Building (AFK Loop)
# ----------------------------------------------------
echo -e "\n\033[37m====================================================\033[0m"
echo -e "\033[36mStep 3: Starting Autonomous Feature Assembly...\033[0m"
echo -e "\n\033[37m====================================================\033[0m"

# Delegate execution directly to loop.sh
bash "${SCRIPT_DIR}/loop.sh" build
