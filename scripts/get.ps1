# Installs the latest hustleX on Windows (64-bit), for the current user:
#   irm https://hustle-x.github.io/hustlex/get.ps1 | iex
$ErrorActionPreference = 'Stop'
# Windows PowerShell 5.1 may still default to TLS 1.0, which GitHub refuses.
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
$repo = 'hustle-x/hustlex'
$version = '1.3.4'
$file = "hustleX-Setup-$version-amd64.exe"
$base = "https://github.com/$repo/releases/latest/download"
if (-not [Environment]::Is64BitOperatingSystem) { throw 'Only 64-bit Windows builds exist for now.' }
$t0 = Get-Date
$mask = @'
 ###                                  #   ###             ###     ###
 ###                                ###   ###              ###   ####
 ###                                ###   ###              #### ####
 ### ####    ###    ##    ######   ###### ###    #####      #######
 #########   ###    ###  ########  ###### ###   #######      #####
 ####   ###  ###    ### ###    ##   ###   ###  ###   ###      ####
 ###    ###  ###    ###  ######     ###   ###  ###   ###     #####
 ###    ###  ###    ###  ########   ###   ###  #########    #######
 ###    ###  ###   ####     ######  ###   ###  ###          ### ####
 ###    ###  ###   #### ###    ###  ###   ###  ###   ###   ###   ###
 ###    ###  ##########  ########   ##### ###   ########  ####   ####
 ###    ###   ##### ##    ######     #### ###    ######  ####     ####
'@ -split "`r?`n"
$rnd = New-Object System.Random
Write-Host ''
foreach ($line in $mask) {
  Write-Host -NoNewline '  '
  foreach ($c in $line.ToCharArray()) { if ($c -eq '#') { Write-Host -NoNewline $rnd.Next(2) -ForegroundColor White } else { Write-Host -NoNewline $c } }
  Write-Host ''
}
Write-Host ''
Write-Host '  every agent that codes, in one place.' -ForegroundColor White
Write-Host "  v$version - Windows - x64" -ForegroundColor DarkGray
Write-Host ''
function Step($a, $b) { Write-Host -NoNewline '  '; Write-Host -NoNewline ([char]0x2713) -ForegroundColor Green; Write-Host (' {0,-24} ' -f $a) -NoNewline; Write-Host $b -ForegroundColor DarkGray }
Step 'Checking system' ((Get-CimInstance Win32_OperatingSystem).Caption)
$path = Join-Path $env:TEMP $file
$ProgressPreference = 'SilentlyContinue'
Invoke-WebRequest -UseBasicParsing "$base/$file" -OutFile $path
Step 'Downloading hustleX' ('{0:N0} MB' -f ((Get-Item $path).Length / 1MB))
# Saved to a file: GitHub serves it as application/octet-stream, so .Content would be bytes.
$sumsPath = Join-Path $env:TEMP 'hustlex-SHA256SUMS'
Invoke-WebRequest -UseBasicParsing "$base/SHA256SUMS" -OutFile $sumsPath
$sums = Get-Content $sumsPath
Remove-Item $sumsPath -ErrorAction SilentlyContinue
$want = ($sums | Where-Object { $_ -match [regex]::Escape($file) } | ForEach-Object { ($_ -split '\s+')[0] }) | Select-Object -First 1
$got = (Get-FileHash $path -Algorithm SHA256).Hash.ToLower()
if (-not $want -or $want.Trim().ToLower() -ne $got) { throw 'The download does not match SHA256SUMS.' }
Step 'Verifying installer' 'sha256 ok'
# A running hustleX, or its daemon (the same program), keeps hustleX.exe
# open, and the installer then leaves the old one in place.
$running = Get-Process -Name hustleX, hustlex-app -ErrorAction SilentlyContinue
if ($running) {
  $running | Stop-Process -Force
  Start-Sleep -Seconds 2
  Step 'Closing hustleX' "$(@($running).Count) process(es)"
}
Start-Process -FilePath $path -ArgumentList '/S' -Wait
Remove-Item $path -ErrorAction SilentlyContinue
$exe = Join-Path $env:LOCALAPPDATA 'Programs\hustleX\hustleX.exe'
$key = Get-ChildItem 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall' -ErrorAction SilentlyContinue |
  Get-ItemProperty | Where-Object { $_.DisplayName -eq 'hustleX' } | Select-Object -First 1
if (-not $key -or $key.DisplayVersion -ne $version) { throw "hustleX $version did not install (found: $($key.DisplayVersion)). Close hustleX in Task Manager and run this again." }
Step 'Installing' "v$version, for the current user"
if ($key.InstallLocation) { $exe = Join-Path $key.InstallLocation 'hustleX.exe' }
if (Test-Path $exe) { Start-Process $exe; Step 'Opening hustleX' '' }
$sec = [int]((Get-Date) - $t0).TotalSeconds
Write-Host ''
Write-Host "  Done in ${sec}s. hustleX is open; next time, find it in the Start menu."
Write-Host ''
Write-Host '  Docs   https://hustle-x.github.io/hustlex/' -ForegroundColor DarkGray
Write-Host ''
