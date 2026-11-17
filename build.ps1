<#
.SYNOPSIS
    Build Cine Sort into a single portable Windows executable.
.DESCRIPTION
    Pipeline:
      1. Verify go / node / pnpm toolchain and Wails CLI v2.17.1
         (installs the CLI with `go install` when missing).
      2. Install frontend dependencies and build Vue/TS assets with pnpm.
      3. Run `wails build -clean -s` to produce build/bin/go-cine-sort.exe.

    No NSIS installer and no zip archive are produced - the deliverable is
    the single exe. Icons are generated separately by generate-icon.ps1.

    The caller's working directory is always restored, even on failure.
.PARAMETER SkipInstall
    Reuse frontend/node_modules (still runs the frontend build).
.EXAMPLE
    .\build.ps1
.EXAMPLE
    .\build.ps1 -SkipInstall
#>
[CmdletBinding()]
param(
    [switch]$SkipInstall
)

$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false

$Root = $PSScriptRoot
$WailsVersion = 'v2.17.1'
$VersionTag = '2\.17\.1'
$OutputExe = Join-Path $Root 'build/bin/go-cine-sort.exe'

function Write-Step([string]$Message) {
    Write-Host "`n==> $Message" -ForegroundColor Cyan
}
function Stop-WithError([string]$Message, [int]$Code = 1) {
    Write-Host "`n[build] ERROR: $Message" -ForegroundColor Red
    exit $Code
}
function Assert-Exit([string]$Name) {
    if ($LASTEXITCODE -ne 0) {
        Stop-WithError "$Name failed (exit code $LASTEXITCODE)" $LASTEXITCODE
    }
}
function Resolve-WailsCli {
    $fromPath = Get-Command wails -ErrorAction SilentlyContinue
    if ($fromPath) {
        $ver = (& $fromPath.Source version 2>&1 | Out-String)
        if ($ver -match $VersionTag) { return $fromPath.Source }
        Write-Host "[build] PATH has an unexpected Wails version, installing $WailsVersion instead." -ForegroundColor Yellow
    }

    $goPath = (& go env GOPATH).Trim()
    $localCli = Join-Path $goPath 'bin/wails.exe'
    if (Test-Path -LiteralPath $localCli) {
        $ver = (& $localCli version 2>&1 | Out-String)
        if ($ver -match $VersionTag) { return $localCli }
    }

    Write-Step "Installing Wails CLI $WailsVersion ..."
    & go install "github.com/wailsapp/wails/v2/cmd/wails@$WailsVersion"
    Assert-Exit 'go install wails'
    if (-not (Test-Path -LiteralPath $localCli)) {
        Stop-WithError "wails.exe not found after install: $localCli"
    }
    return $localCli
}

# Push/Pop so the caller's working directory survives every exit path,
# including Stop-WithError (exit) inside the pipeline below.
Push-Location
try {
    Set-Location -LiteralPath $Root

    # ---------------------------------------------------------------- 1. checks
    Write-Step 'Checking project and toolchain'
    foreach ($required in @('wails.json', 'go.mod', 'frontend/package.json')) {
        if (-not (Test-Path -LiteralPath (Join-Path $Root $required))) {
            Stop-WithError "Missing project file: $required"
        }
    }
    foreach ($tool in @('go', 'node', 'pnpm')) {
        if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
            Stop-WithError "$tool not found in PATH"
        }
    }
    Write-Host ("  Go   : {0}" -f (go version))
    Write-Host ("  Node : {0}" -f (node -v))
    Write-Host ("  pnpm : {0}" -f (pnpm -v))

    # ---------------------------------------------------------------- 2. wails cli
    $wails = Resolve-WailsCli
    Write-Host "  Wails: $wails ($WailsVersion)"

    # ---------------------------------------------------------------- 3. icons
    foreach ($iconFile in @((Join-Path $Root 'build/appicon.png'), (Join-Path $Root 'build/windows/icon.ico'))) {
        if (-not (Test-Path -LiteralPath $iconFile)) {
            Stop-WithError "Icon resource missing: $iconFile (run .\generate-icon.ps1 first)"
        }
    }

    # ---------------------------------------------------------------- 4. frontend
    Set-Location -LiteralPath (Join-Path $Root 'frontend')
    if ($SkipInstall) {
        Write-Step 'Skipping pnpm install (-SkipInstall)'
    } else {
        Write-Step 'Installing frontend dependencies (pnpm install --frozen-lockfile)'
        # Call the .cmd shim directly: the pnpm.ps1 shim writes script echoes to
        # stderr, which PowerShell 5.1 reports as a NativeCommandError.
        & pnpm.cmd install --frozen-lockfile
        Assert-Exit 'pnpm install'
    }

    Write-Step 'Building Vue + TypeScript frontend'
    & pnpm.cmd run build
    Assert-Exit 'pnpm run build'
    if (-not (Test-Path -LiteralPath (Join-Path $Root 'frontend/dist/index.html'))) {
        Stop-WithError 'frontend/dist/index.html was not produced'
    }

    # ---------------------------------------------------------------- 5. wails build
    Set-Location -LiteralPath $Root
    Write-Step 'Building single-file Windows executable (wails build -clean -s)'
    & $wails build -clean -s
    Assert-Exit 'wails build'

    if (-not (Test-Path -LiteralPath $OutputExe)) {
        Stop-WithError "Executable not found: $OutputExe"
    }
    $exe = Get-Item -LiteralPath $OutputExe
    Write-Host ''
    Write-Host '[build] Build succeeded.' -ForegroundColor Green
    Write-Host ("  Executable : {0}" -f $exe.FullName)
    Write-Host ("  Size       : {0:N2} MB" -f ($exe.Length / 1MB))
    Write-Host ("  Built at   : {0}" -f $exe.LastWriteTime)
    Write-Host ("  Icon       : {0}" -f (Join-Path $Root 'build/windows/icon.ico'))
    exit 0
} finally {
    Pop-Location
}
