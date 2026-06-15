# Shared helpers for the CronCompose Windows installer: logging, prompts, port
# detection, and secret generation. Dot-sourced by install.ps1. Targets PowerShell 7+
# (works on Windows PowerShell 5.1 too).

function Write-Step($msg) { Write-Host "`n==> $msg" -ForegroundColor Cyan }
function Write-Info($msg) { Write-Host "    $msg" }
function Write-Ok($msg)   { Write-Host "  + $msg" -ForegroundColor Green }
function Write-Warnn($msg){ Write-Host "  ! $msg" -ForegroundColor Yellow }
function Write-Dim($msg)  { Write-Host "    $msg" -ForegroundColor DarkGray }
function Die($msg) { Write-Host "  x $msg" -ForegroundColor Red; exit 1 }

# Prompt with a default. In non-interactive mode returns the default unread.
function Read-Default($label, $default) {
  if ($script:NonInteractive) { return $default }
  $suffix = if ($default) { " [$default]" } else { "" }
  $ans = Read-Host "$label$suffix"
  if ([string]::IsNullOrWhiteSpace($ans)) { return $default }
  return $ans
}

function Read-Secret($label, $default) {
  if ($script:NonInteractive) { return $default }
  $sec = Read-Host "$label" -AsSecureString
  $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($sec)
  try { $val = [Runtime.InteropServices.Marshal]::PtrToStringAuto($bstr) }
  finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr) }
  if ([string]::IsNullOrWhiteSpace($val)) { return $default }
  return $val
}

function Confirm-Yes($question, $default = 'n') {
  if ($script:NonInteractive) { return ($default -eq 'y') }
  $hint = if ($default -eq 'y') { '[Y/n]' } else { '[y/N]' }
  $ans = Read-Host "$question $hint"
  if ([string]::IsNullOrWhiteSpace($ans)) { $ans = $default }
  return ($ans -match '^[Yy]')
}

# True if nothing is listening on the loopback port (i.e. it is free to bind).
function Test-PortFree([int]$Port) {
  try {
    $l = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, $Port)
    $l.Start(); $l.Stop(); return $true
  } catch { return $false }
}

function Find-FreePort([int]$Start) {
  $p = $Start
  for ($i = 0; $i -lt 500; $i++) {
    if (Test-PortFree $p) { return $p }
    $p++
  }
  return $Start
}

# Prompt for a port, defaulting to the first free one at/after $Start.
function Read-Port($label, [int]$Start) {
  $suggest = Find-FreePort $Start
  while ($true) {
    $chosen = Read-Default $label $suggest
    $n = 0
    if (-not [int]::TryParse($chosen, [ref]$n) -or $n -lt 1 -or $n -gt 65535) {
      Write-Warnn "not a valid port: $chosen"; if ($script:NonInteractive) { return $suggest }; continue
    }
    if (-not (Test-PortFree $n)) {
      Write-Warnn "port $n is already in use"
      if ($script:NonInteractive) { return $n }
      if (-not (Confirm-Yes "  Use it anyway?" 'n')) { continue }
    }
    return $n
  }
}

function New-HexSecret([int]$Bytes = 32) {
  $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
  $b = [byte[]]::new($Bytes)
  $rng.GetBytes($b)
  ($b | ForEach-Object { $_.ToString('x2') }) -join ''
}

function Get-HostIp {
  try {
    $ip = (Get-NetIPAddress -AddressFamily IPv4 -ErrorAction Stop |
           Where-Object { $_.IPAddress -notlike '127.*' -and $_.IPAddress -notlike '169.254.*' } |
           Select-Object -First 1).IPAddress
    if ($ip) { return $ip }
  } catch {}
  return 'localhost'
}

# Poll an HTTP endpoint until it answers (any status) or attempts run out.
function Wait-Http($url, [int]$attempts = 60, [double]$delay = 0.5) {
  for ($i = 0; $i -lt $attempts; $i++) {
    try { Invoke-WebRequest -UseBasicParsing -Uri $url -TimeoutSec 3 | Out-Null; return $true }
    catch {
      # A 3xx/4xx still means the server is up and answering.
      if ($_.Exception.Response) { return $true }
    }
    Start-Sleep -Seconds $delay
  }
  return $false
}

function Wait-Listen([int]$Port, [int]$attempts = 60, [double]$delay = 0.5) {
  for ($i = 0; $i -lt $attempts; $i++) {
    if (-not (Test-PortFree $Port)) { return $true }
    Start-Sleep -Seconds $delay
  }
  return $false
}
