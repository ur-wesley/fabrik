#!/bin/bash
# fabrik.sh
# Master Orchestrator for the Fabrik workflow.
# Run this script to execute the entire workflow back-to-back:
# Grill Session (Interactive) -> PRD-to-Issues (Auto) -> Build Loops (Auto)
# Usage: ./.fabrik/fabrik.sh

# Resolve directory paths relative to script
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

clear
echo -e "\033[36m🏭 Starting the Fabrik Workflow...\033[0m"

# ----------------------------------------------------
# Phase 1: Interactive Requirements Alignment
# ----------------------------------------------------
echo -e "\n\033[37m====================================================\033[0m"
echo -e "\033[36mStep 1: Interactive Alignment Session\033[0m"
echo -e "\033[37m====================================================\033[0m"
echo "1. Type /grill-with-docs to stress-test your plan."
echo "2. Type /to-prd to write the requirements to docs/PRD.md or .fabrik/docs/PRD.md."
echo "3. Type /exit to hand over control to the automated pipeline."
read -p "Press [Enter] when ready to launch the OpenCode TUI..."

opencode

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
