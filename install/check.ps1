#Requires -Version 5.1
param([switch]$Json)

# Fabrik tool check (Windows). Exit 0 = ok, 1 = missing.

function Test-Cmd([string]$Name) { $null -ne (Get-Command $Name -ErrorAction SilentlyContinue) }

$tools = @('bd', 'engram', 'graphify', 'bun', 'pi', 'uv')
$result = @{}
$fail = $false
foreach ($t in $tools) {
  $ok = Test-Cmd $t
  $result[$t] = if ($ok) { 'ok' } else { 'missing' }
  if (-not $ok) { $fail = $true }
}

if ($Json) {
  $result['ok'] = (-not $fail)
  $result | ConvertTo-Json -Compress
} else {
  foreach ($t in $tools) { Write-Host "$t : $($result[$t])" }
  if ($fail) {
    Write-Host ''
    Write-Host 'Missing tools. Run .\install\setup.ps1.'
  }
}

if ($fail) { exit 1 }
exit 0
