#Requires -Version 5.1
$ErrorActionPreference = 'Stop'

$Root = $PSScriptRoot
$DepsPath = Join-Path $Root 'deps.json'
$deps = Get-Content -Raw -LiteralPath $DepsPath | ConvertFrom-Json

function Latest-GithubTag([string]$Repo) {
    (Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest").tag_name
}

function Latest-PypiVersion([string]$Package) {
    (Invoke-RestMethod "https://pypi.org/pypi/$Package/json").info.version
}

$beadsTag = Latest-GithubTag $deps.beads.repo
$engramTag = Latest-GithubTag $deps.engram.repo
$graphifyVersion = Latest-PypiVersion $deps.graphify.pypi
$beadsVer = $beadsTag.TrimStart('v')
$engramVer = $engramTag.TrimStart('v')

$deps.beads.version = $beadsVer
$deps.beads.tag = $beadsTag
$deps.beads.assets.windows_amd64 = "beads_${beadsVer}_windows_amd64.zip"
$deps.beads.assets.windows_arm64 = "beads_${beadsVer}_windows_arm64.zip"
$deps.beads.assets.linux_amd64 = "beads_${beadsVer}_linux_amd64.tar.gz"
$deps.beads.assets.linux_arm64 = "beads_${beadsVer}_linux_arm64.tar.gz"
$deps.beads.assets.darwin_amd64 = "beads_${beadsVer}_darwin_amd64.tar.gz"
$deps.beads.assets.darwin_arm64 = "beads_${beadsVer}_darwin_arm64.tar.gz"

$deps.engram.version = $engramVer
$deps.engram.tag = $engramTag
$deps.engram.assets.windows_amd64 = "engram_${engramVer}_windows_amd64.zip"
$deps.engram.assets.windows_arm64 = "engram_${engramVer}_windows_arm64.zip"
$deps.engram.assets.linux_amd64 = "engram_${engramVer}_linux_amd64.tar.gz"
$deps.engram.assets.linux_arm64 = "engram_${engramVer}_linux_arm64.tar.gz"
$deps.engram.assets.darwin_amd64 = "engram_${engramVer}_darwin_amd64.tar.gz"
$deps.engram.assets.darwin_arm64 = "engram_${engramVer}_darwin_arm64.tar.gz"

$deps.graphify.version = $graphifyVersion

$deps | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $DepsPath -Encoding utf8

Write-Host "Updated deps.json:"
Write-Host "  beads    $($deps.beads.tag)"
Write-Host "  engram   $($deps.engram.tag)"
Write-Host "  graphify $($deps.graphify.version)"
Write-Host ""
Write-Host 'Run .\install\setup.ps1 to install the new pins.'
