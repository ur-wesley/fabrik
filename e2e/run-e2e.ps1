#Requires -Version 5.1
$ErrorActionPreference = 'Stop'

$Root = Split-Path $PSScriptRoot -Parent

if (-not (Get-Command bd -ErrorAction SilentlyContinue)) {
    Write-Error 'bd not on PATH. Run .\install\setup.ps1 first.'
}

if (-not (Get-Command bun -ErrorAction SilentlyContinue)) {
    Write-Error 'bun not on PATH.'
}

Push-Location $Root
try {
    Write-Host '==> Fabrik E2E' -ForegroundColor Cyan
    bun test ./e2e/*.e2e.test.ts
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    Write-Host 'E2E passed.' -ForegroundColor Green
}
finally {
    Pop-Location
}
