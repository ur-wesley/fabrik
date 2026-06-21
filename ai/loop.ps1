# loop.ps1
# Native Windows PowerShell runner for the AI workflow loop using OpenCode.
# Usage: .\ai\loop.ps1 -Mode "build" -MaxIterations 20

param (
    [ValidateSet("plan", "build")]
    [string]$Mode = "build",

    [int]$MaxIterations = 0
)

# Resolve paths relative to the script's directory (ai/)
$PromptFile = Join-Path $PSScriptRoot "PROMPT_$Mode.md"
$TasksDir = Join-Path $PSScriptRoot ".tasks"
$CompletedDir = Join-Path $TasksDir "completed"

if (-not (Test-Path $PromptFile)) {
    Write-Error "Prompt file $PromptFile not found."
    exit 1
}

$Branch = (git branch --show-current).Trim()
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
Write-Host "Agent:  OpenCode"
Write-Host "Mode:   $Mode"
Write-Host "Prompt: $PromptFile"
Write-Host "Branch: $Branch"
Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan

# Create tasks directory structure if it does not exist
if (-not (Test-Path $TasksDir)) { New-Item -ItemType Directory -Path $TasksDir | Out-Null }
if (-not (Test-Path $CompletedDir)) { New-Item -ItemType Directory -Path $CompletedDir | Out-Null }

$Iteration = 0
while ($true) {
    # Check iteration limit
    if ($MaxIterations -gt 0 -and $Iteration -ge $MaxIterations) {
        Write-Host "Reached max iteration limit: $MaxIterations" -ForegroundColor Yellow
        break
    }

    # If in building mode, verify there are tasks to work on
    if ($Mode -eq "build") {
        $OpenTasks = Get-ChildItem -Path $TasksDir -Filter "*.md" -File
        if ($OpenTasks.Count -eq 0) {
            Write-Host "🎉 No remaining open tasks in $TasksDir. Work complete!" -ForegroundColor Green
            break
        }
        Write-Host "Pending tasks: $($OpenTasks.Count)" -ForegroundColor Gray
    }

    Write-Host "`n[Iteration $($Iteration + 1)] Starting OpenCode Session..." -ForegroundColor Green
    
    # Run OpenCode CLI using the correct agent mode
    # Pipes the prompt file contents directly into the agent
    Get-Content $PromptFile | opencode --agent $Mode

    # Sync remote repository
    Write-Host "Syncing remote repository..." -ForegroundColor Gray
    git push origin $Branch

    $Iteration++
    
    # If in plan mode, we only need 1 run to generate tasks
    if ($Mode -eq "plan") {
        Write-Host "Planning phase complete. Start the build loop with: .\ai\loop.ps1 -Mode build" -ForegroundColor Green
        break
    }

    Write-Host "======================== TASK CYCLE $Iteration COMPLETE ========================" -ForegroundColor Green
}
