param (
    [switch]$Auto,
    [string]$InitialPrompt
)

$FabrikDir = $PSScriptRoot

$RequiredSetupItems = @(
    "docs",
    "styleguide",
    "CONTEXT.md"
)
$SetupIncomplete = $false
foreach ($Item in $RequiredSetupItems) {
    if (-not (Test-Path (Join-Path $FabrikDir $Item))) {
        $SetupIncomplete = $true
        break
    }
}

if (-not (Test-Path (Join-Path (Get-Location) '.beads'))) {
    $SetupIncomplete = $true
}

if ($SetupIncomplete) {
    Write-Host '[WARNING] Fabrik setup incomplete.' -ForegroundColor Red
    Write-Host 'Run: .\install\setup.ps1 then .\install\init.ps1 .' -ForegroundColor Yellow
    exit 1
}

Clear-Host
Write-Host '[FABRIK] Starting workflow...' -ForegroundColor Cyan

if ($Auto) {
    if ([string]::IsNullOrWhiteSpace($InitialPrompt)) {
        $InitialPrompt = Read-Host 'Initial prompt for PRD generation'
    }
    $promptCmd = "You are an expert product manager. Generate a detailed PRD in .fabrik/docs/PRD.md based on these requirements: $InitialPrompt"
    opencode run $promptCmd
} else {
    Write-Host 'Step 1: Interactive alignment' -ForegroundColor Cyan
    Write-Host '1. /grill-with-docs  2. /to-prd  3. /exit when done' -ForegroundColor Gray
    Read-Host 'Press enter to launch OpenCode TUI'
    opencode
}

Write-Host 'Step 2: Planning (PRD to Beads issues)...' -ForegroundColor Cyan
& "$PSScriptRoot/loop.ps1" -Mode plan

Write-Host 'Step 3: Build loop...' -ForegroundColor Cyan
& "$PSScriptRoot/loop.ps1" -Mode build
