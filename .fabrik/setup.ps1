Write-Host '[FABRIK] Per-repo setup...' -ForegroundColor Cyan

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot '..')
$InitScript = Join-Path $RepoRoot 'install\init.ps1'

if (-not (Test-Path $InitScript)) {
    Write-Error "install/init.ps1 not found. Clone the full Fabrik repo."
    exit 1
}

& $InitScript -RepoPath $RepoRoot

# Canonical skills/subagents are installed by install/init (Cursor, OpenCode, Pi only).

$ContextFile = Join-Path $PSScriptRoot 'CONTEXT.md'
if (-not (Test-Path $ContextFile)) {
    @(
        '# Project Context & Dictionary',
        '',
        'Domain terms and architecture rules for this project.',
        '',
        '## Domain Language',
        '*   **Term**: [Definition]',
        '',
        '## Architecture Guidelines',
        '*   Prefer deep modules over thin wrappers.'
    ) -join [Environment]::NewLine | Set-Content -LiteralPath $ContextFile -Encoding utf8
}

$Styleguide = Join-Path $PSScriptRoot 'styleguide\STYLEGUIDE.md'
if (-not (Test-Path $Styleguide)) {
    New-Item -ItemType Directory -Force -Path (Split-Path $Styleguide) | Out-Null
    '# Styleguide' | Set-Content -LiteralPath $Styleguide -Encoding utf8
}

Write-Host '[DONE] Repo ready. Run .\.fabrik\fabrik.ps1 or bd ready.' -ForegroundColor Green
