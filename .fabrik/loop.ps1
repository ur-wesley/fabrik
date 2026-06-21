# loop.ps1
# Native Windows PowerShell runner for the AI workflow loop using OpenCode.
# Includes real-time execution tracking, runtime timers, and a live state dashboard.
# Usage: .\.fabrik\loop.ps1 -Mode "build" -MaxIterations 20

param (
    [ValidateSet("plan", "build")]
    [string]$Mode = "build",

    [int]$MaxIterations = 0
)

# Resolve paths relative to the script's directory (.fabrik/)
$FabrikDir = $PSScriptRoot
$TasksDir = Join-Path $FabrikDir ".tasks"
$CompletedDir = Join-Path $TasksDir "completed"
$PromptFile = Join-Path $FabrikDir "PROMPT_$Mode.md"
$StateFile = Join-Path $FabrikDir "state.md"

if (-not (Test-Path $PromptFile)) {
    Write-Error "Prompt file $PromptFile not found."
    exit 1
}

$Branch = (git branch --show-current).Trim()

# Initialize task folders
if (-not (Test-Path $TasksDir)) { New-Item -ItemType Directory -Path $TasksDir | Out-Null }
if (-not (Test-Path $CompletedDir)) { New-Item -ItemType Directory -Path $CompletedDir | Out-Null }

# Clear/Initialize the live state file
$StartTimeStr = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
# Note: Here-string closing delimiter MUST start at column 1 (no leading spaces)
@"
# Fabrik Loop Status: INITIALIZING

*   **Started At:** $StartTimeStr
*   **Mode:** `$Mode`
*   **Branch:** `$Branch`
*   **Status:** `Running Setup...`
"@ | Out-File -FilePath $StateFile -Encoding utf8

$Iteration = 0
while ($true) {
    # Check iteration limit
    if ($MaxIterations -gt 0 -and $Iteration -ge $MaxIterations) {
        Write-Host "Reached max iteration limit: $MaxIterations" -ForegroundColor Yellow
        break
    }

    # Fetch pending tasks
    $OpenTasks = Get-ChildItem -Path $TasksDir -Filter "*.md" -File | Sort-Object Name
    $NextTaskName = "None"
    $NextTaskDesc = "No description available."

    # If in building mode, verify there are tasks to work on
    if ($Mode -eq "build") {
        if ($OpenTasks.Count -eq 0) {
            Write-Host "🎉 No remaining open tasks. Work complete!" -ForegroundColor Green
            # Update state file to finished
            $EndTimeStr = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
# Note: Here-string closing delimiter MUST start at column 1 (no leading spaces)
@"
# Fabrik Loop Status: FINISHED / IDLE

*   **Last Run Complete:** $EndTimeStr
*   **Outcome:** All tasks executed successfully!
*   **Status:** `IDLE`
"@ | Out-File -FilePath $StateFile -Encoding utf8
            break
        }
        $NextTask = $OpenTasks[0]
        $NextTaskName = $NextTask.Name
        # Read the first few lines to get the task title
        $NextTaskLines = Get-Content -Path $NextTask.FullName -TotalCount 5
        foreach ($Line in $NextTaskLines) {
            if ($Line.Trim() -and -not $Line.StartsWith("#")) {
                $NextTaskDesc = $Line.Trim()
                break
            }
        }
    } else {
        $NextTaskName = "Planning Phase (Gap Analysis & PRD Triage)"
        $NextTaskDesc = "Comparing docs/PRD.md against the src/ directory to generate task issues."
    }

    $CycleStart = Get-Date
    $CycleStartStr = $CycleStart.Format("HH:mm:ss")

    # Update terminal screen dashboard
    Clear-Host
    Write-Host "====================================================" -ForegroundColor Cyan
    Write-Host "🏭 FABRIK DASHBOARD (Iteration: $($Iteration + 1))" -ForegroundColor Cyan
    Write-Host "====================================================" -ForegroundColor Cyan
    Write-Host "Mode:         $Mode" -ForegroundColor Gray
    Write-Host "Branch:       $Branch" -ForegroundColor Gray
    Write-Host "Target Task:  $NextTaskName" -ForegroundColor Yellow
    Write-Host "Description:  $NextTaskDesc" -ForegroundColor Gray
    Write-Host "Started At:   $CycleStartStr" -ForegroundColor Gray
    Write-Host "Remaining:    $($OpenTasks.Count) tasks" -ForegroundColor Gray
    Write-Host "====================================================" -ForegroundColor Cyan
    Write-Host "Status: Running OpenCode agent run..." -ForegroundColor DarkYellow

    # Update the live state file
# Note: Here-string closing delimiter MUST start at column 1 (no leading spaces)
@"
# Fabrik Loop Status: RUNNING 🔄

*   **Last Update:** $($CycleStart.Format("yyyy-MM-dd HH:mm:ss"))
*   **Iteration:** $($Iteration + 1)
*   **Current Task:** `$NextTaskName`
*   **Description:** $NextTaskDesc
*   **Cycle Started At:** $CycleStartStr
*   **Pending Queue:** $($OpenTasks.Count) tasks
*   **Status:** `Active - Executing OpenCode Run`
"@ | Out-File -FilePath $StateFile -Encoding utf8

    # Run OpenCode CLI using the correct agent mode
    $PromptText = Get-Content -Raw -Path $PromptFile
    opencode run --agent $Mode $PromptText

    # Calculate runtime duration
    $CycleEnd = Get-Date
    $Duration = $CycleEnd - $CycleStart
    $DurationStr = "{0:mm}m {0:ss}s" -f $Duration

    Write-Host "`nSyncing remote branch..." -ForegroundColor Gray
    git push origin $Branch

    # Write completion details to screen
    Write-Host "Task complete! Duration: $DurationStr" -ForegroundColor Green

    # Log task completion in state file
    $LogEntry = "*   [$($CycleEnd.Format("HH:mm:ss"))] Completed `$NextTaskName` in $DurationStr"
    if (-not (Test-Path $StateFile)) { "" | Out-File -FilePath $StateFile }
    $StateContent = Get-Content -Path $StateFile
    
    # Append log entry to state file
    $NewState = @()
    $HasLogHeader = $false
    foreach ($Line in $StateContent) {
        if ($Line.StartsWith("# Fabrik Loop Status:")) {
            $NewState += "# Fabrik Loop Status: SLEEP / SYNCING 😴"
        }
        elseif ($Line.StartsWith("*   **Status:**")) {
            $NewState += "*   **Status:** `Waiting for next iteration (Last cycle took $DurationStr)`"
        }
        else {
            $NewState += $Line
        }
        if ($Line -eq "## Recent Run Log") { $HasLogHeader = $true }
    }
    if (-not $HasLogHeader) {
        $NewState += ""
        $NewState += "## Recent Run Log"
    }
    $NewState += $LogEntry
    $NewState | Out-File -FilePath $StateFile -Encoding utf8

    $Iteration++
    
    # If in plan mode, we only need 1 run to generate tasks
    if ($Mode -eq "plan") {
        Write-Host "Planning phase complete. Start the build loop with: .\.fabrik\loop.ps1 -Mode build" -ForegroundColor Green
        break
    }

    Write-Host "======================== TASK CYCLE $Iteration COMPLETE ========================" -ForegroundColor Green
    Start-Sleep -Seconds 3 # Short buffer between iterations
}
