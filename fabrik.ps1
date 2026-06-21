# fabrik.ps1
# Master Orchestrator for the Fabrik workflow.
# Run this script to execute the entire workflow back-to-back:
# Grill Session (Interactive) -> PRD-to-Issues (Auto) -> Build Loops (Auto)
# Usage: .\fabrik.ps1

# Ensure paths relative to the script directory
$FabrikDir = Join-Path $PSScriptRoot ".fabrik"
$TasksDir = Join-Path $FabrikDir ".tasks"
$PlanPromptFile = Join-Path $FabrikDir "PROMPT_plan.md"
$BuildPromptFile = Join-Path $FabrikDir "PROMPT_build.md"

Clear-Host
Write-Host "🏭 Starting the Fabrik Workflow..." -ForegroundColor Cyan

# ----------------------------------------------------
# Phase 1: Interactive Requirements Alignment
# ----------------------------------------------------
Write-Host "`n====================================================" -ForegroundColor Gray
Write-Host "Step 1: Interactive Alignment Session" -ForegroundColor Cyan
Write-Host "====================================================" -ForegroundColor Gray
Write-Host "1. Type /grill-with-docs to stress-test your plan." -ForegroundColor Gray
Write-Host "2. Type /to-prd to write the requirements to .fabrik/docs/PRD.md." -ForegroundColor Gray
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

if (-not (Test-Path $PlanPromptFile)) {
    Write-Error "Error: Prompt file $PlanPromptFile not found."
    exit 1
}

$PlanPromptText = Get-Content -Raw -Path $PlanPromptFile
opencode run --agent plan $PlanPromptText

# ----------------------------------------------------
# Phase 3: Automated Feature Building (AFK Loop)
# ----------------------------------------------------
Write-Host "`n====================================================" -ForegroundColor Gray
Write-Host "Step 3: Starting Autonomous Feature Assembly..." -ForegroundColor Cyan
Write-Host "====================================================" -ForegroundColor Gray

if (-not (Test-Path $BuildPromptFile)) {
    Write-Error "Error: Prompt file $BuildPromptFile not found."
    exit 1
}

$BuildPromptText = Get-Content -Raw -Path $BuildPromptFile
$Branch = (git branch --show-current).Trim()

$Iteration = 0
while ($true) {
    # Check if any tasks exist in .fabrik/.tasks/
    $OpenTasks = Get-ChildItem -Path $TasksDir -Filter "*.md" -File
    if ($OpenTasks.Count -eq 0) {
        Write-Host "`n🎉 All tasks implemented, verified, and committed! Work complete!" -ForegroundColor Green
        break
    }

    Write-Host "`n[Iteration $($Iteration + 1)] Executing next task... (Pending tasks: $($OpenTasks.Count))" -ForegroundColor Green
    
    opencode run --agent build $BuildPromptText

    # Push to origin
    Write-Host "Syncing branch..." -ForegroundColor Gray
    git push origin $Branch

    $Iteration++
}
