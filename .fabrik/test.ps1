$errors = $null
$tokens = $null
[System.Management.Automation.Language.Parser]::ParseFile("$PSScriptRoot/loop.ps1", [ref]$tokens, [ref]$errors)
$errors | ForEach-Object { Write-Host $_.Message "at line" $_.Extent.StartLineNumber }
