<#
.SYNOPSIS
    Render the single SVG brand asset into Wails and Windows icon resources.
.DESCRIPTION
    Uses the pure-Go tools/icongen renderer (oksvg + rasterx), so no
    ImageMagick, Inkscape, Python or native binary is required.

    Outputs:
      build/appicon.png          512x512 application icon
      build/windows/icon.ico     16/24/32/48/64/128/256 multi-size ICO
      frontend/public/logo.svg   favicon copy, synced from the SVG source
.PARAMETER Root
    Project root. Defaults to the folder containing this script.
.EXAMPLE
    .\generate-icon.ps1
#>
[CmdletBinding()]
param(
    [string]$Root = ''
)

$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false

function Write-Step([string]$Message) {
    Write-Host "[icon] $Message" -ForegroundColor Cyan
}
function Stop-WithError([string]$Message, [int]$Code = 1) {
    Write-Host "[icon] ERROR: $Message" -ForegroundColor Red
    exit $Code
}

if ([string]::IsNullOrWhiteSpace($Root)) {
    $Root = $PSScriptRoot
}
try {
    $Root = (Resolve-Path -LiteralPath $Root).Path
} catch {
    Stop-WithError "Project root not found: $Root"
}
Set-Location -LiteralPath $Root

if (-not (Test-Path -LiteralPath (Join-Path $Root 'wails.json'))) {
    Stop-WithError "wails.json not found under $Root"
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Stop-WithError "Go toolchain not found in PATH"
}

$svgSource = Join-Path $Root 'frontend/src/assets/logo.svg'
if (-not (Test-Path -LiteralPath $svgSource)) {
    Stop-WithError "SVG source not found: $svgSource"
}

Write-Step "Project : $Root"
Write-Step "SVG     : $svgSource"
Write-Step "Go      : $(go version)"
Write-Step 'Rendering PNG frames and ICO container...'

& go run ./tools/icongen -root $Root
if ($LASTEXITCODE -ne 0) {
    Stop-WithError "icongen failed (exit code $LASTEXITCODE)" $LASTEXITCODE
}

$appIcon = Join-Path $Root 'build/appicon.png'
$icoFile = Join-Path $Root 'build/windows/icon.ico'
foreach ($file in @($appIcon, $icoFile)) {
    if (-not (Test-Path -LiteralPath $file)) {
        Stop-WithError "Missing output: $file"
    }
    $item = Get-Item -LiteralPath $file
    if ($item.Length -le 0) {
        Stop-WithError "Empty output: $file"
    }
    Write-Step ("Output  : {0} ({1:N1} KB)" -f $item.FullName, ($item.Length / 1KB))
}

Write-Host '[icon] Done.' -ForegroundColor Green
exit 0
