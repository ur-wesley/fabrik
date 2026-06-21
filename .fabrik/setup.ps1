# setup.ps1
# Automates the setup of the Fabrik workflow framework.
# Installs battle-tested skills for OpenCode and initializes directories.

Write-Host "🏭 Initializing the Fabrik Method Setup..." -ForegroundColor Cyan

# 1. Check Node.js / NPM dependency
if (-not (Get-Command "node" -ErrorAction SilentlyContinue) -or -not (Get-Command "npx" -ErrorAction SilentlyContinue)) {
    Write-Error "Error: Node.js and NPX are required to install agent skills. Please install Node.js first."
    exit 1
}

# 2. Check git initialization
if (-not (Test-Path ".git")) {
    Write-Host "Initializing git repository..." -ForegroundColor Gray
    git init | Out-Null
}

# 3. Create required directories
Write-Host "Creating directory layout..." -ForegroundColor Gray
$Dirs = @(
    ".fabrik",
    ".fabrik/.tasks",
    ".fabrik/.tasks/completed",
    ".fabrik/docs",
    ".fabrik/specs",
    ".fabrik/styleguide",
    "src"
)
foreach ($Dir in $Dirs) {
    $FullPath = Join-Path $PSScriptRoot "../$Dir"
    if (-not (Test-Path $FullPath)) {
        New-Item -ItemType Directory -Path $FullPath | Out-Null
    }
}

# 4. Install mattpocock/skills using npx
Write-Host "Installing battle-tested community skills from mattpocock/skills..." -ForegroundColor Yellow
# Run skills.sh installer for OpenCode
npx -y skills@latest add mattpocock/skills --agent opencode

# 5. Check if CONTEXT.md exists, if not create a default one
$ContextFile = Join-Path $PSScriptRoot "CONTEXT.md"
if (-not (Test-Path $ContextFile)) {
# Note: Here-string closing delimiter MUST start at column 1 (no leading spaces)
@"
# Project Context & Dictionary

This document defines the domain terms and architecture rules for this project.

## Domain Language
*   **Term**: [Definition of specific project jargon to reduce agent verbosity]

## Architecture Guidelines
*   Standard imports should reference interfaces.
*   Prefer deep modules over thin helper wrappers.
"@ | Out-File -FilePath $ContextFile -Encoding utf8
}

Write-Host "`n🎉 Setup Complete!" -ForegroundColor Green
Write-Host "Next steps:" -ForegroundColor Gray
Write-Host "1. Set D:/projects/fabrik as your active workspace." -ForegroundColor Gray
Write-Host "2. Trigger OpenCode and run: /setup-matt-pocock-skills" -ForegroundColor Gray
Write-Host "3. Start planning by running: .\.fabrik\loop.ps1 -Mode plan" -ForegroundColor Gray
