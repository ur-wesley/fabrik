param (
    [ValidateSet("plan", "build")]
    [string]$Mode = "build",
    [int]$MaxIterations = 0
)

$FabrikDir = $PSScriptRoot
$PromptFile = Join-Path $FabrikDir "PROMPT_$Mode.md"
$StateFile = Join-Path $FabrikDir "state.md"

if (-not (Test-Path $PromptFile)) {
    Write-Error "Prompt file $PromptFile not found."
    exit 1
}

if (-not (Get-Command bd -ErrorAction SilentlyContinue)) {
    Write-Error "bd not found. Run install/setup first."
    exit 1
}

$Branch = (git branch --show-current 2>$null).Trim()
if (-not $Branch) { $Branch = 'main' }

$StartTimeStr = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
@"
# Fabrik Loop Status: INITIALIZING

*   **Started At:** $StartTimeStr
*   **Mode:** $Mode
*   **Branch:** $Branch
*   **Status:** Running...
"@ | Set-Content -LiteralPath $StateFile -Encoding utf8

$Iteration = 0
while ($true) {
    if ($MaxIterations -gt 0 -and $Iteration -ge $MaxIterations) {
        Write-Host "Reached max iteration limit: $MaxIterations" -ForegroundColor Yellow
        break
    }

    $openJson = bd list --status=open --json 2>$null
    $openIssues = @()
    if ($openJson) { $openIssues = $openJson | ConvertFrom-Json }
    $closedJson = bd list --status=closed --json --limit 100 2>$null
    $closedCount = 0
    if ($closedJson) { $closedCount = @($closedJson | ConvertFrom-Json).Count }

    $readyJson = bd ready --json 2>$null
    $readyIssues = @()
    if ($readyJson) { $readyIssues = $readyJson | ConvertFrom-Json }

    $NextTaskId = 'plan'
    $NextTaskDesc = 'PRD gap analysis and bd create'
    if ($Mode -eq 'build') {
        if ($openIssues.Count -eq 0) {
            Write-Host '[DONE] No open Beads issues.' -ForegroundColor Green
            break
        }
        if ($readyIssues.Count -eq 0) {
            Write-Host '[BLOCKED] Open issues exist but none are ready.' -ForegroundColor Yellow
            break
        }
        $NextTaskId = $readyIssues[0].id
        $NextTaskDesc = $readyIssues[0].title
    }

    Clear-Host
    Write-Host '====================================================' -ForegroundColor Cyan
    Write-Host "[FABRIK] $Mode (iteration $($Iteration + 1))" -ForegroundColor Cyan
    Write-Host "Open: $($openIssues.Count) | Closed: $closedCount | Ready: $($readyIssues.Count)" -ForegroundColor Gray
    Write-Host "Target: $NextTaskId - $NextTaskDesc" -ForegroundColor Yellow
    Write-Host '====================================================' -ForegroundColor Cyan

    $PromptText = Get-Content -Raw -Path $PromptFile
    opencode run $PromptText

    git push origin $Branch 2>$null

    $Iteration++
    if ($Mode -eq 'plan') {
        Write-Host 'Planning complete. Run: .\.fabrik\loop.ps1 -Mode build' -ForegroundColor Green
        break
    }
    Start-Sleep -Seconds 3
}
