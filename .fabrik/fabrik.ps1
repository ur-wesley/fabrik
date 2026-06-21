# fabrik.ps1
# Master Orchestrator for the Fabrik workflow.
# Run this script to execute the entire workflow back-to-back:
# Grill Session (Interactive) -> PRD-to-Issues (Auto) -> Build Loops (Auto)
# Usage: .\.fabrik\fabrik.ps1

Clear-Host
Write-Host "🏭 Starting the Fabrik Workflow..." -ForegroundColor Cyan

# ----------------------------------------------------
# Phase 1: Interactive Requirements Alignment
# ----------------------------------------------------
Write-Host "`n====================================================" -ForegroundColor Gray
Write-Host "Step 1: Interactive Alignment Session" -ForegroundColor Cyan
Write-Host "====================================================" -ForegroundColor Gray
Write-Host "1. Type /grill-with-docs to stress-test your plan." -ForegroundColor Gray
Write-Host "2. Type /to-prd to write the requirements to docs/PRD.md or .fabrik/docs/PRD.md." -ForegroundColor Gray
Write-Host "3. Type /exit to hand over control to the automated pipeline." -ForegroundColor Yellow
Write-Host "Press enter when ready to launch the OpenCode TUI..."
Read-Host

opencode

# ----------------------------------------------------
# Phase 2: Automated Planning (PRD to Issues)
# ----------------------------------------------------
Write-Host "`n====================================================" -ForegroundColor Gray
Write-Host "Step 2: Running Automated Planning (PRD-to-Issues)..." -ForegroundColor Cyan
Write-Host "====================================================" -ForegroundColor Gray

# Delegate execution directly to loop.ps1
& "$PSScriptRoot/loop.ps1" -Mode plan

# ----------------------------------------------------
# Phase 3: Automated Feature Building (AFK Loop)
# ----------------------------------------------------
Write-Host "`n====================================================" -ForegroundColor Gray
Write-Host "Step 3: Starting Autonomous Feature Assembly..." -ForegroundColor Cyan
Write-Host "====================================================" -ForegroundColor Gray

# Delegate execution directly to loop.ps1
& "$PSScriptRoot/loop.ps1" -Mode build
