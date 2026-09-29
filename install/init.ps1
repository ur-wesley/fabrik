#Requires -Version 5.1
param(
    [Parameter(Position = 0)]
    [string]$RepoPath = (Get-Location).Path
)

$ErrorActionPreference = 'Stop'
$InstallRoot = $PSScriptRoot
$FabrikRoot = Split-Path $InstallRoot -Parent

if (-not (Get-Command bd -ErrorAction SilentlyContinue)) {
    Write-Error "bd not found. Run .\install\setup.ps1 first."
}

$resolved = Resolve-Path -LiteralPath $RepoPath
Push-Location $resolved

try {
    if (Test-Path '.beads') {
        Write-Host "Beads already initialized in $resolved"
    } else {
        bd init --non-interactive --skip-agents --skip-hooks -q
        Write-Host "Beads initialized (embedded Dolt in .beads/)"
        Write-Host "Run bd setup cursor (and engram setup) from install/setup if needed."
    }

    $fabrikDir = Join-Path $resolved '.fabrik'
    New-Item -ItemType Directory -Force -Path $fabrikDir, (Join-Path $fabrikDir 'docs'), (Join-Path $fabrikDir 'specs'), (Join-Path $fabrikDir 'styleguide') | Out-Null

    $templateDir = Join-Path $FabrikRoot '.fabrik'
    $orchestrationFiles = @(
        'PROMPT_plan.md', 'PROMPT_build.md', 'loop.ps1', 'loop.sh',
        'fabrik.ps1', 'fabrik.sh', 'AGENTS.md', 'setup.ps1', 'setup.sh', 'CONTEXT.md'
    )
    foreach ($file in $orchestrationFiles) {
        $src = Join-Path $templateDir $file
        $dst = Join-Path $fabrikDir $file
        if ((Test-Path $src) -and -not (Test-Path $dst)) {
            Copy-Item -LiteralPath $src -Destination $dst
        }
    }
    $styleSrc = Join-Path $templateDir 'styleguide\STYLEGUIDE.md'
    $styleDst = Join-Path $fabrikDir 'styleguide\STYLEGUIDE.md'
    if ((Test-Path $styleSrc) -and -not (Test-Path $styleDst)) {
        Copy-Item -LiteralPath $styleSrc -Destination $styleDst
    }

    $configYaml = Join-Path $fabrikDir 'config.yaml'
    if (-not (Test-Path $configYaml)) {
        Copy-Item -LiteralPath (Join-Path $FabrikRoot '.fabrik\config.yaml') -Destination $configYaml -ErrorAction SilentlyContinue
        if (-not (Test-Path $configYaml)) {
            @"
agent: pi

models:
  default: inherit

session:
  auto_lock: true
  auto_commit: true

tools:
  rtk: true
  engram: true
  caveman: true
  ponytail: true

skills:
  - caveman
  - ponytail
  - grill-with-docs
  - to-prd
  - to-issues
  - tdd
"@ | Set-Content -LiteralPath $configYaml -Encoding utf8
        }
    }

    $workflowTemplate = Join-Path $InstallRoot 'templates\workflow-note.md'
    $agents = Join-Path $resolved 'AGENTS.md'
    $marker = '## Fabrik workflow'
    $workflowBlock = Get-Content -Raw -LiteralPath $workflowTemplate
    if (Test-Path $agents) {
        $body = Get-Content -Raw -LiteralPath $agents
        if ($body -notmatch 'Fabrik workflow') {
            "`n$marker`n`n$workflowBlock" | Add-Content -LiteralPath $agents -Encoding utf8
            Write-Host 'Appended Fabrik workflow to AGENTS.md'
        }
    } else {
        "# Agent instructions`n`n$marker`n`n$workflowBlock" | Set-Content -LiteralPath $agents -Encoding utf8
        Write-Host 'Created AGENTS.md'
    }

    Write-Host ''
    Write-Host 'Optional: graphify . when search is slow.'
    Write-Host 'Next: grill, bd create issues, bd ready, claim, close.'
}
finally {
    Pop-Location
}
