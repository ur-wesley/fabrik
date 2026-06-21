# fabrik.ps1
# Master Orchestrator for the Fabrik workflow.
# Run this script to execute the entire workflow back-to-back:
# Grill Session (Interactive) -> PRD-to-Issues (Auto) -> Build Loops (Auto)
# Usage: .\.fabrik\fabrik.ps1

# Ensure paths relative to the script directory (.fabrik/)
$FabrikDir = $PSScriptRoot
$TasksDir = Join-Path $FabrikDir ".tasks"
$PlanPromptFile = Join-Path $FabrikDir "PROMPT_plan.md"
$BuildPromptFile = Join-Path $FabrikDir "PROMPT_build.md"

# ----------------------------------------------------
# Setup Verification Check
# ----------------------------------------------------
$RequiredSetupItems = @(
    ".tasks",
    "docs",
    "styleguide",
    "styleguide/STYLEGUIDE.md",
    "CONTEXT.md"
)
$SetupIncomplete = $false
foreach ($Item in $RequiredSetupItems) {
    $Path = Join-Path $FabrikDir $Item
    if (-not (Test-Path $Path)) {
        $SetupIncomplete = $true
        break
    }
}

if ($SetupIncomplete) {
    Clear-Host
    Write-Host "[WARNING] Fabrik setup is incomplete or has not been run in this directory." -ForegroundColor Red
    Write-Host "Please execute the setup script to initialize the framework and skills:" -ForegroundColor Yellow
    Write-Host "    .\.fabrik\setup.ps1" -ForegroundColor Yellow
    Write-Host ""
    exit 1
}

Clear-Host
Write-Host "[FABRIK] Starting the Fabrik Workflow..." -ForegroundColor Cyan

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
