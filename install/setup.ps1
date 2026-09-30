#Requires -Version 5.1
param(
    [switch]$SkipToolInstall,
    [switch]$SkipEngramSetup,
    [switch]$SkipPiPackages,
    [string]$RepoPath
)

$ErrorActionPreference = 'Stop'

$Root = $PSScriptRoot
$DepsPath = Join-Path $Root 'deps.json'
$Template = Join-Path $Root 'templates\workflow-note.md'
$BinDir = Join-Path $env:USERPROFILE '.local\bin'
# Cursor / OpenCode / Pi only — never .agents, .claude, .codex.
$GraphifySkills = @(
  (Join-Path $env:USERPROFILE '.cursor\rules\graphify.mdc'),
  (Join-Path $env:USERPROFILE '.config\opencode\skills\graphify.md'),
  (Join-Path $env:USERPROFILE '.pi\agent\skills\graphify.md')
)

if (-not (Test-Path $DepsPath)) { Write-Error "Missing deps.json at $DepsPath" }
$Deps = Get-Content -Raw -LiteralPath $DepsPath | ConvertFrom-Json

function Test-Command([string]$Name) {
    $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Write-Step([string]$Text) {
    Write-Host ""
    Write-Host "==> $Text" -ForegroundColor Cyan
}

function Get-PlatformKey {
    if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { return 'windows_arm64' }
    return 'windows_amd64'
}

function Get-BinaryName([string]$Base) { return "$Base.exe" }

function Ensure-BinDir {
    New-Item -ItemType Directory -Force -Path $BinDir | Out-Null
    if ($env:PATH -notlike "*$BinDir*") {
        $env:PATH = "$BinDir;$env:PATH"
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        if ($userPath -notlike "*$BinDir*") {
            [Environment]::SetEnvironmentVariable('Path', "$BinDir;$userPath", 'User')
            Write-Host "Added $BinDir to user PATH"
        }
    }
}

function Install-GithubArchiveBinary {
    param(
        [string]$Repo,
        [string]$Tag,
        [string]$Asset,
        [string]$BinaryName,
        [string]$Label
    )
    $dest = Join-Path $BinDir $BinaryName
    if (Test-Path $dest) {
        Write-Host "$Label already at $dest"
        return
    }
    if ($SkipToolInstall) { Write-Error "$Label missing and -SkipToolInstall set" }
    Write-Step "Installing $Label $Tag"
    $url = "https://github.com/$Repo/releases/download/$Tag/$Asset"
    $archive = Join-Path $env:TEMP "$BinaryName-$Tag"
    Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $archive
    $extract = Join-Path $env:TEMP "fabrik-$BinaryName-$Tag"
    if (Test-Path $extract) { Remove-Item -Recurse -Force $extract }
    New-Item -ItemType Directory -Force -Path $extract | Out-Null
    if ($Asset.EndsWith('.zip')) {
        Expand-Archive -LiteralPath $archive -DestinationPath $extract -Force
    } else {
        tar -xzf $archive -C $extract
    }
    $found = Get-ChildItem -Path $extract -Recurse -File | Where-Object { $_.Name -eq $BinaryName } | Select-Object -First 1
    if (-not $found) { Write-Error "Could not find $BinaryName inside $Asset" }
    Copy-Item -LiteralPath $found.FullName -Destination $dest -Force
    Remove-Item -LiteralPath $archive -Force -ErrorAction SilentlyContinue
    Remove-Item -Recurse -Force $extract -ErrorAction SilentlyContinue
    Write-Host "Installed $dest"
}

function Install-Bd {
    if (Test-Command bd) {
        Write-Host "bd already on PATH: $(bd version 2>&1 | Select-Object -First 1)"
        return
    }
    $key = Get-PlatformKey
    $asset = $Deps.beads.assets.$key
    Install-GithubArchiveBinary -Repo $Deps.beads.repo -Tag $Deps.beads.tag -Asset $asset -BinaryName (Get-BinaryName $Deps.beads.binary) -Label 'Beads'
}

function Install-Engram {
    if (Test-Command engram) {
        Write-Host "engram already on PATH: $(engram version 2>&1 | Select-Object -First 1)"
        return
    }
    if ($SkipToolInstall) { Write-Error 'engram missing and -SkipToolInstall set' }
    if (Test-Command go) {
        Write-Step "Installing Engram $($Deps.engram.tag) via go install"
        go install "$($Deps.engram.goModule)@$($Deps.engram.tag)"
        $goBin = Join-Path (go env GOPATH) 'bin'
        $goExe = Join-Path $goBin (Get-BinaryName $Deps.engram.binary)
        if (Test-Path $goExe) {
            Copy-Item -LiteralPath $goExe -Destination (Join-Path $BinDir (Get-BinaryName $Deps.engram.binary)) -Force
            Write-Host "Copied engram to $BinDir"
            return
        }
    }
    $key = Get-PlatformKey
    $asset = $Deps.engram.assets.$key
    Install-GithubArchiveBinary -Repo $Deps.engram.repo -Tag $Deps.engram.tag -Asset $asset -BinaryName (Get-BinaryName $Deps.engram.binary) -Label 'Engram'
}

function Install-Graphify {
    Write-Step "Installing Graphify skill (Cursor, OpenCode, Pi only)"
    $tmp = Join-Path $env:TEMP 'fabrik-graphify-skill.md'
    Invoke-WebRequest -UseBasicParsing -Uri $Deps.graphify.skillUrl -OutFile $tmp
    foreach ($dest in $GraphifySkills) {
      New-Item -ItemType Directory -Force -Path (Split-Path $dest) | Out-Null
      Copy-Item -LiteralPath $tmp -Destination $dest -Force
      Write-Host "Wrote $dest"
    }
    Remove-Item -LiteralPath $tmp -Force -ErrorAction SilentlyContinue
    if (Test-Command graphify) { Write-Host 'graphify CLI already on PATH'; return }
    if ($SkipToolInstall) { return }
    if (Test-Command uv) {
        Write-Step "Installing $($Deps.graphify.pypi)==$($Deps.graphify.version) with uv"
        uv tool install "$($Deps.graphify.pypi)==$($Deps.graphify.version)"
        return
    }
    Write-Host "uv not found; install graphify CLI manually:"
    Write-Host "  uv tool install $($Deps.graphify.pypi)==$($Deps.graphify.version)"
}

function Install-PiPackages {
    if ($SkipPiPackages) { return }
    if (-not (Test-Command pi)) { Write-Host 'pi not on PATH; skip pi package install'; return }
    Write-Step 'Installing Pi MCP adapter'
    pi install $Deps.pi.mcpAdapter
}

function Install-WorkflowNote {
    Write-Step 'Installing personal workflow note'
    $note = Get-Content -Raw -LiteralPath $Template
    $cursorRules = Join-Path $env:USERPROFILE '.cursor\rules\ai-workflow.mdc'
    New-Item -ItemType Directory -Force -Path (Split-Path $cursorRules) | Out-Null
    @"
---
description: Fabrik workflow with Beads, Engram, and Graphify
alwaysApply: true
---
$note
"@ | Set-Content -LiteralPath $cursorRules -Encoding utf8
    Write-Host "Wrote $cursorRules"
    $opencodeAgents = Join-Path $env:USERPROFILE '.config\opencode\AGENTS.md'
    New-Item -ItemType Directory -Force -Path (Split-Path $opencodeAgents) | Out-Null
    $note | Set-Content -LiteralPath $opencodeAgents -Encoding utf8
    Write-Host "Wrote $opencodeAgents"
    $piAgents = Join-Path $env:USERPROFILE '.pi\agent\AGENTS.md'
    New-Item -ItemType Directory -Force -Path (Split-Path $piAgents) | Out-Null
    $note | Set-Content -LiteralPath $piAgents -Encoding utf8
    Write-Host "Wrote $piAgents"
}

function Setup-EngramAgents {
    if ($SkipEngramSetup) { return }
    if (-not (Test-Command engram)) { Write-Host 'Skipping engram setup (engram not on PATH)'; return }
    Write-Step 'Running engram setup for Cursor, OpenCode, Pi'
    engram setup cursor
    engram setup opencode
    engram setup pi
}

function Setup-BeadsCursor {
    if (-not (Test-Command bd)) { return }
    Write-Step 'Running bd setup (Cursor, OpenCode only)'
    try { bd setup cursor } catch { }
    try { bd setup opencode } catch { }
}

Write-Host 'Fabrik machine setup' -ForegroundColor Green
Write-Host "Deps: beads $($Deps.beads.tag), engram $($Deps.engram.tag), graphify $($Deps.graphify.version)"

Ensure-BinDir
Install-Bd
Install-Engram
Install-Graphify
Install-PiPackages
Install-WorkflowNote
Setup-EngramAgents
Setup-BeadsCursor

if ($RepoPath) {
    Write-Step "Initializing repo: $RepoPath"
    & (Join-Path $Root 'init.ps1') -RepoPath $RepoPath
}

Write-Host ""
Write-Host 'Done.' -ForegroundColor Green
Write-Host 'Restart Cursor, OpenCode, and Pi so MCP and rules reload.'
Write-Host 'Per repo:  .\install\init.ps1 D:\path\to\your-repo'
