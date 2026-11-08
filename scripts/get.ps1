# Installs the latest hustleX on Windows (64-bit), for the current user:
#   irm https://github.com/hustle-x/hustlex/releases/latest/download/get.ps1 | iex
$ErrorActionPreference = 'Stop'
$repo = 'hustle-x/hustlex'
if (-not [Environment]::Is64BitOperatingSystem) { throw 'Only 64-bit Windows builds exist for now.' }
$file = Join-Path $env:TEMP 'hustleX-Setup.exe'
Write-Host 'Downloading hustleX...'
Invoke-WebRequest -UseBasicParsing "https://github.com/$repo/releases/latest/download/hustleX-Setup-1.0.0-amd64.exe" -OutFile $file
Write-Host 'Installing...'
Start-Process -FilePath $file -ArgumentList '/S' -Wait
Remove-Item $file -ErrorAction SilentlyContinue
Write-Host 'Done. Open hustleX from the Start menu.'
