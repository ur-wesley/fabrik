#Requires -Version 5.1
param(
  [Parameter(Position = 0)]
  [string]$RepoPath = (Get-Location).Path,
  [switch]$SkipChecks
)

# Fabrik repo init (Windows). Parity with install/init.sh.
# Sets up .fabrik overview hub + Cursor/Pi/OpenCode only. Never other agents.

$ErrorActionPreference = 'Stop'
$InstallRoot = $PSScriptRoot
$FabrikRoot = Split-Path $InstallRoot -Parent

if (-not $SkipChecks) {
  $check = Join-Path $InstallRoot 'check.ps1'
  try { & $check } catch { }
  Write-Host '(advisory: optional tools may be missing — continuing, bd is required)'
  $global:LASTEXITCODE = 0
  if (-not (Get-Command bd -ErrorAction SilentlyContinue)) {
    Write-Error 'bd not found. Run .\install\setup.ps1 first.'
  }
}

if (-not (Get-Command bd -ErrorAction SilentlyContinue)) {
  Write-Error 'bd not found. Run .\install\setup.ps1 first.'
}

$resolved = Resolve-Path -LiteralPath $RepoPath
Push-Location $resolved
try {
  if (Test-Path '.beads') {
    Write-Host "Beads already initialized in $resolved"
  } else {
    bd init --non-interactive --skip-agents --skip-hooks -q
    Write-Host 'Beads initialized'
  }

  $fabrikDir = Join-Path $resolved '.fabrik'
  New-Item -ItemType Directory -Force -Path $fabrikDir, (Join-Path $fabrikDir 'docs'), (Join-Path $fabrikDir 'specs'), (Join-Path $fabrikDir 'styleguide'), (Join-Path $fabrikDir 'agents') | Out-Null

  function Copy-Missing([string]$Src, [string]$Dst) {
    if ((Test-Path $Src) -and -not (Test-Path $Dst)) { Copy-Item -LiteralPath $Src -Destination $Dst }
  }

  foreach ($f in @('PROMPT_plan.md', 'PROMPT_build.md', 'loop.ps1', 'fabrik.ps1', 'AGENTS.md')) {
    Copy-Missing (Join-Path $FabrikRoot ".fabrik\$f") (Join-Path $fabrikDir $f)
  }
  Copy-Missing (Join-Path $InstallRoot 'templates\fabrik\README.md') (Join-Path $fabrikDir 'README.md')
  Copy-Missing (Join-Path $InstallRoot 'templates\fabrik\CONTEXT.md') (Join-Path $fabrikDir 'CONTEXT.md')
  Copy-Missing (Join-Path $InstallRoot 'templates\fabrik\config.yaml') (Join-Path $fabrikDir 'config.yaml')
  Copy-Missing (Join-Path $InstallRoot 'templates\fabrik\skills.md') (Join-Path $fabrikDir 'skills.md')
  if (-not (Test-Path (Join-Path $fabrikDir '.gitignore'))) {
    Copy-Missing (Join-Path $InstallRoot 'templates\fabrik\gitignore') (Join-Path $fabrikDir '.gitignore')
  }
  Copy-Missing (Join-Path $FabrikRoot '.fabrik\styleguide\STYLEGUIDE.md') (Join-Path $fabrikDir 'styleguide\STYLEGUIDE.md')
  foreach ($a in @('orchestrator', 'explore', 'researcher', 'briefer', 'planner', 'builder', 'tester', 'reviewer', 'style-smells', 'security')) {
    Copy-Missing (Join-Path $InstallRoot "templates\agents\$a.md") (Join-Path $fabrikDir "agents\$a.md")
  }

  Copy-Missing (Join-Path $InstallRoot 'templates\opencode.json') (Join-Path $resolved 'opencode.json')  # replaces built-in build with Fabrik roster

  $skills = @('i-have-adhd', 'caveman', 'ponytail', 'rtk-usage', 'tdd', 'diagnose', 'guardrails')
  New-Item -ItemType Directory -Force -Path (Join-Path $resolved '.cursor\rules'), (Join-Path $resolved '.opencode\skills'), (Join-Path $resolved '.pi\skills') | Out-Null
  foreach ($s in $skills) {
    $src = Join-Path $FabrikRoot "skills\$s.md"
    if (-not (Test-Path $src)) { continue }
    $body = Get-Content -Raw -LiteralPath $src
    $mdc = Join-Path $resolved ".cursor\rules\$s.mdc"
    if (-not (Test-Path $mdc)) {
      "---`ndescription: Fabrik skill $s`nalwaysApply: false`n---`n`n$body" | Set-Content -LiteralPath $mdc -Encoding utf8
    }
    Copy-Missing $src (Join-Path $resolved ".opencode\skills\$s.md")
    Copy-Missing $src (Join-Path $resolved ".pi\skills\$s.md")
  }

  New-Item -ItemType Directory -Force -Path (Join-Path $resolved '.pi\agent\agents'), (Join-Path $resolved '.cursor\rules\agents') | Out-Null
  foreach ($a in @('orchestrator', 'explore', 'researcher', 'briefer', 'planner', 'builder', 'tester', 'reviewer', 'style-smells', 'security')) {
    $src = Join-Path $InstallRoot "templates\agents\$a.md"
    Copy-Missing $src (Join-Path $resolved ".pi\agent\agents\$a.md")
    $mdc = Join-Path $resolved ".cursor\rules\agents\$a.mdc"
    if (-not (Test-Path $mdc)) {
      $body = Get-Content -Raw -LiteralPath $src
      "---`ndescription: Fabrik subagent $a`nalwaysApply: false`n---`n`n$body" | Set-Content -LiteralPath $mdc -Encoding utf8
    }
  }

  $hadAgents = Test-Path (Join-Path $resolved '.agents')
  $hadClaude = Test-Path (Join-Path $resolved '.claude')
  $hadCodex = Test-Path (Join-Path $resolved '.codex')
  if (Get-Command bd -ErrorAction SilentlyContinue) {
    try { bd setup cursor 2>$null | Out-Null } catch { }
    try { bd setup opencode 2>$null | Out-Null } catch { }
  }
  if (Get-Command engram -ErrorAction SilentlyContinue) {
    foreach ($app in @('cursor', 'opencode', 'pi')) {
      try { engram setup $app 2>$null | Out-Null } catch { }
    }
  }
  # bd/engram may side-effect generic dirs; remove only if we didn't have them (3 apps only)
  if ((-not $hadAgents) -and (Test-Path (Join-Path $resolved '.agents'))) { Remove-Item -Recurse -Force (Join-Path $resolved '.agents') }
  if ((-not $hadClaude) -and (Test-Path (Join-Path $resolved '.claude'))) { Remove-Item -Recurse -Force (Join-Path $resolved '.claude') }
  if ((-not $hadCodex) -and (Test-Path (Join-Path $resolved '.codex'))) { Remove-Item -Recurse -Force (Join-Path $resolved '.codex') }
  $global:LASTEXITCODE = 0

  $workflowTemplate = Join-Path $InstallRoot 'templates\workflow-note.md'
  $agents = Join-Path $resolved 'AGENTS.md'
  $workflowBlock = Get-Content -Raw -LiteralPath $workflowTemplate
  if (Test-Path $agents) {
    $body = Get-Content -Raw -LiteralPath $agents
    if ($body -notmatch 'Fabrik workflow') {
      "`n## Fabrik workflow`n`n$workflowBlock" | Add-Content -LiteralPath $agents -Encoding utf8
      Write-Host 'Appended Fabrik workflow to AGENTS.md'
    }
  } else {
    "# Agent instructions`n`n## Fabrik workflow`n`n$workflowBlock" | Set-Content -LiteralPath $agents -Encoding utf8
    Write-Host 'Created AGENTS.md'
  }

  Write-Host ''
  Write-Host 'Done. Apps: Cursor, OpenCode, Pi. Hub: .fabrik/README.md'
  Write-Host 'Next: grill, bd create issues, bd ready, claim, close.'
}
finally { Pop-Location }
