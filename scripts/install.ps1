#Requires -Version 5.1
<#
.SYNOPSIS
  Daedalus installer for Windows.

.DESCRIPTION
  Downloads the matching release archive from GitHub Releases, verifies its
  SHA-256 checksum, extracts daedalus.exe, installs it under your user profile,
  and adds it to your user PATH.

.EXAMPLE
  # Quick install (latest release):
  irm https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.ps1 | iex

.EXAMPLE
  # Install a specific version or directory:
  & ([scriptblock]::Create((irm https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.ps1))) -Version v0.1.0
#>
[CmdletBinding()]
param(
  [string]$Version = $env:DAEDALUS_VERSION,
  [string]$BinDir = $env:DAEDALUS_INSTALL_DIR
)

$ErrorActionPreference = 'Stop'
$repo = 'Codigo-de-Altura/Daedalus'
$binary = 'daedalus'

function Info($m) { Write-Host "==> $m" -ForegroundColor Cyan }

# --- detect architecture ---
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  'AMD64' { 'amd64' }
  'ARM64' { 'arm64' }
  default { throw "unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)" }
}

# --- resolve the release tag ---
if (-not $Version) {
  Info 'Resolving latest release...'
  $rel = Invoke-RestMethod "https://api.github.com/repos/$repo/releases/latest" `
    -Headers @{ 'User-Agent' = 'daedalus-installer' }
  $Version = $rel.tag_name
  if (-not $Version) { throw 'could not find a published release yet — build from source instead (see the docs)' }
}
$ver = $Version.TrimStart('v')

$asset = "${binary}_${ver}_windows_${arch}.zip"
$checksums = "${binary}_${ver}_checksums.txt"
$base = "https://github.com/$repo/releases/download/$Version"

# --- choose an install directory ---
if (-not $BinDir) { $BinDir = Join-Path $env:LOCALAPPDATA 'Programs\Daedalus' }
New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

$tmp = Join-Path $env:TEMP ("daedalus-" + [System.Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
try {
  $zip = Join-Path $tmp $asset
  Info "Downloading $asset ($Version)..."
  Invoke-WebRequest "$base/$asset" -OutFile $zip -UseBasicParsing

  # --- verify checksum ---
  try {
    $sumFile = Join-Path $tmp $checksums
    Invoke-WebRequest "$base/$checksums" -OutFile $sumFile -UseBasicParsing
    $line = Select-String -Path $sumFile -Pattern ([regex]::Escape($asset)) | Select-Object -First 1
    $want = ($line.Line -split '\s+')[0]
    $got = (Get-FileHash $zip -Algorithm SHA256).Hash
    if ($want -and ($want -ne $got)) { throw "checksum mismatch for $asset" }
    Info 'Checksum verified.'
  } catch {
    Write-Warning "checksum not verified: $($_.Exception.Message)"
  }

  # --- extract + install ---
  Info 'Extracting...'
  Expand-Archive -Path $zip -DestinationPath (Join-Path $tmp 'out') -Force
  $exe = Get-ChildItem -Path (Join-Path $tmp 'out') -Recurse -Filter "$binary.exe" |
    Select-Object -First 1
  if (-not $exe) { throw "binary '$binary.exe' not found in archive" }
  $dest = Join-Path $BinDir "$binary.exe"
  Copy-Item $exe.FullName $dest -Force
  Info "Installed $binary $Version -> $dest"

  # --- add to user PATH ---
  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  if (($userPath -split ';') -notcontains $BinDir) {
    $newPath = ($userPath.TrimEnd(';') + ';' + $BinDir).TrimStart(';')
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Info "Added $BinDir to your user PATH. Open a new terminal to use 'daedalus'."
  }
  $env:Path = "$env:Path;$BinDir"

  & $dest --version
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
