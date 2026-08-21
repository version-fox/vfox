#!/usr/bin/env pwsh
#
#    Copyright 2026 Han Li and contributors
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.

<#
.SYNOPSIS
    Builds vfox from source, packs it into a .msixbundle and optionally signs it.

.DESCRIPTION
    Requires Windows with the Windows SDK (MakeAppx.exe / SignTool.exe) and Go.
    Takes no parameters. The version is read from internal/version.go (the
    single source of truth) and normalized to a four-part MSIX version
    (1.2.3 -> 1.2.3.0); prerelease or build metadata segments are rejected.
    Icons are generated from the repository logo, each architecture is built
    from source and packed into its own .msix rendered from AppxManifest.xml,
    and all packages are combined into vfox_<version>_windows.msixbundle.
    The publisher is fixed to "CN=VersionFox". Signing is optional and
    configured through the environment only:
      MSIX_SIGN_PFX_PATH (falls back to <script>/signing.pfx when present),
      MSIX_SIGN_PFX_PASSWORD (falls back to MSIX_PFX_PASSWORD).

.EXAMPLE
    ./make-msix.ps1
#>

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Assert-NativeSuccess {
    param([string]$Step)
    if ($LASTEXITCODE -ne 0) {
        throw "$Step failed with exit code $LASTEXITCODE"
    }
}

function ConvertTo-MsixVersion {
    param([string]$Raw)

    # Strip a leading 'v'. Prerelease (-rc1) and metadata (+build) segments
    # cannot be represented in the four-part MSIX version and are rejected.
    $core = $Raw -replace '^[vV]', ''
    if ($core -match '[-+]') {
        throw "Unsupported version '$Raw': MSIX packages require a stable version without prerelease or build metadata."
    }
    $parts = @($core -split '\.' | ForEach-Object { [int]$_ })
    if ($parts.Count -gt 3) {
        throw "Unsupported version '$Raw': at most three numeric segments are expected"
    }
    while ($parts.Count -lt 4) {
        $parts += 0
    }
    return ($parts -join '.')
}

function Find-SdkTool {
    param([string]$ToolName)

    $kitsRootEnv = ${env:ProgramFiles(x86)}
    if ([string]::IsNullOrWhiteSpace($kitsRootEnv)) {
        throw "Environment variable 'ProgramFiles(x86)' is not set; $ToolName.exe can only be located on Windows with the Windows SDK installed."
    }

    $kitsRoot = Join-Path $kitsRootEnv "Windows Kits\10\bin"
    if (-not (Test-Path $kitsRoot)) {
        throw "Windows SDK not found under '$kitsRoot'. Install the Windows SDK to get $ToolName.exe."
    }

    $tool = Get-ChildItem -Path $kitsRoot -Directory |
        Where-Object { $_.Name -match '^\d+(\.\d+)+$' } |
        Sort-Object { [version]$_.Name } -Descending |
        ForEach-Object {
            Get-ChildItem -Path (Join-Path $_.FullName "x64") -Filter "$ToolName.exe" -ErrorAction SilentlyContinue
        } |
        Select-Object -First 1

    if ($null -eq $tool) {
        throw "$ToolName.exe not found under '$kitsRoot'. Install the Windows SDK."
    }
    return $tool.FullName
}

function Invoke-SourceBuild {
    # Builds vfox.exe for every architecture from the repository source.
    # Keep the flags aligned with .goreleaser.yaml (CGO_ENABLED=0, -trimpath).
    # The binary reports the version from internal/version.go; the script
    # never overrides it, so what you pack is what the source tree says.
    param([string]$RepoRoot, [string]$BuildDir)

    if ($null -eq (Get-Command "go" -ErrorAction SilentlyContinue)) {
        throw "'go' is required to build from source but was not found in PATH. Install Go and try again."
    }

    $archs = @(
        @{ GoArch = "386";   Name = "x86" },
        @{ GoArch = "amd64"; Name = "x64" },
        @{ GoArch = "arm64"; Name = "arm64" }
    )
    Push-Location $RepoRoot
    try {
        foreach ($a in $archs) {
            $outDir = Join-Path $BuildDir $a.Name
            New-Item -ItemType Directory -Path $outDir -Force | Out-Null
            $env:GOOS = "windows"
            $env:GOARCH = $a.GoArch
            $env:CGO_ENABLED = "0"
            go build -trimpath -ldflags "-s -w" -o (Join-Path $outDir "vfox.exe") .
            Assert-NativeSuccess -Step "go build ($($a.GoArch))"
            Write-Host "Built $($a.Name): $(Join-Path $outDir 'vfox.exe')"
        }
    }
    finally {
        Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
        Pop-Location
    }
}

# Script lives at <repo>/packaging/msix/, so the repository root is two levels up.
$repoRoot = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent

# Single source of truth: the version comes from internal/version.go and is
# never passed in, so a locally built bundle always matches the source tree.
$versionFile = Join-Path $repoRoot "internal/version.go"
$versionMatch = Select-String -Path $versionFile -Pattern '(const|var)\s+RuntimeVersion\s*=\s*"([^"]+)"' | Select-Object -First 1
if ($null -eq $versionMatch) {
    throw "RuntimeVersion not found in '$versionFile'."
}
$Version = $versionMatch.Matches[0].Groups[2].Value
$msixVersion = ConvertTo-MsixVersion -Raw $Version

$Publisher = "CN=VersionFox"
$OutputDir = Join-Path $PSScriptRoot "Output"
$WorkDir = Join-Path $PSScriptRoot "staging"

# Signing is environment-only. MSIX_SIGN_PFX_PATH falls back to a signing.pfx
# next to this script (the compile-msix workflow decodes its secret there);
# the password falls back to MSIX_PFX_PASSWORD (the secret's name).
$SignPfxPath = $env:MSIX_SIGN_PFX_PATH
if ([string]::IsNullOrWhiteSpace($SignPfxPath)) {
    $candidatePfx = Join-Path $PSScriptRoot "signing.pfx"
    if (Test-Path $candidatePfx) {
        $SignPfxPath = $candidatePfx
    }
}
$SignPfxPassword = $env:MSIX_SIGN_PFX_PASSWORD
if ([string]::IsNullOrWhiteSpace($SignPfxPassword)) {
    $SignPfxPassword = $env:MSIX_PFX_PASSWORD
}

Write-Host "Building vfox $Version from source..."
Invoke-SourceBuild -RepoRoot $repoRoot -BuildDir (Join-Path $PSScriptRoot "build")
$selected = @()
foreach ($arch in @("x86", "x64", "arm64")) {
    $selected += @{
        Architecture = $arch
        ExePath      = (Resolve-Path (Join-Path $PSScriptRoot "build/$arch/vfox.exe")).Path
    }
}

$makeAppx = Find-SdkTool -ToolName "MakeAppx"

# Generate the tile icons from the repository logo (they are not committed).
& (Join-Path $PSScriptRoot "gen-assets.ps1") `
    -LogoPath (Join-Path $repoRoot "logo.png") `
    -OutDir (Join-Path $PSScriptRoot "assets")

New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
New-Item -ItemType Directory -Path $WorkDir -Force | Out-Null

$manifestTemplate = Get-Content -Raw -Path (Join-Path $PSScriptRoot "AppxManifest.xml")
$assetsDir = Join-Path $PSScriptRoot "assets"
if (-not (Test-Path $assetsDir)) {
    throw "Assets directory not found: $assetsDir"
}

$packages = @()
foreach ($entry in $selected) {
    $arch = $entry.Architecture
    $stage = Join-Path $WorkDir $arch
    if (Test-Path $stage) {
        Remove-Item -Path $stage -Recurse -Force
    }
    New-Item -ItemType Directory -Path $stage -Force | Out-Null

    Copy-Item -Path $entry.ExePath -Destination (Join-Path $stage "vfox.exe")
    Copy-Item -Path $assetsDir -Destination (Join-Path $stage "assets") -Recurse

    $manifest = $manifestTemplate
    $manifest = $manifest.Replace("@@VERSION@@", $msixVersion)
    $manifest = $manifest.Replace("@@ARCHITECTURE@@", $arch)
    # Escape XML entities in the fixed publisher value before token replacement.
    $manifest = $manifest.Replace("@@PUBLISHER@@", [System.Security.SecurityElement]::Escape($Publisher))
    Set-Content -Path (Join-Path $stage "AppxManifest.xml") -Value $manifest -Encoding UTF8 -NoNewline

    $packagePath = Join-Path $OutputDir ("vfox_{0}_windows_{1}.msix" -f $Version, $arch)
    & $makeAppx pack /o /d $stage /p $packagePath | Out-Host
    Assert-NativeSuccess -Step "MakeAppx pack ($arch)"

    $packages += $packagePath
    Write-Host "Created package: $packagePath"
}

$bundleStage = Join-Path $WorkDir "bundle"
if (Test-Path $bundleStage) {
    Remove-Item -Path $bundleStage -Recurse -Force
}
New-Item -ItemType Directory -Path $bundleStage -Force | Out-Null
foreach ($pkg in $packages) {
    Copy-Item -Path $pkg -Destination $bundleStage
}

$bundlePath = Join-Path $OutputDir ("vfox_{0}_windows.msixbundle" -f $Version)
& $makeAppx bundle /o /d $bundleStage /p $bundlePath | Out-Host
Assert-NativeSuccess -Step "MakeAppx bundle"
Write-Host "Created bundle: $bundlePath"

if (-not [string]::IsNullOrWhiteSpace($SignPfxPath)) {
    if ([string]::IsNullOrWhiteSpace($SignPfxPassword)) {
        throw "A signing certificate was found but no password was provided. Set MSIX_SIGN_PFX_PASSWORD (or MSIX_PFX_PASSWORD)."
    }
    $signTool = Find-SdkTool -ToolName "signtool"
    & $signTool sign /fd SHA256 /f $SignPfxPath /p $SignPfxPassword $bundlePath | Out-Host
    Assert-NativeSuccess -Step "SignTool sign"
    Write-Host "Signed bundle: $bundlePath"
}
else {
    Write-Warning "No signing certificate provided; publishing UNSIGNED bundle. Windows blocks installation of unsigned packages unless they are re-signed first."
}

Write-Host "`nArtifacts:"
Get-ChildItem -Path $OutputDir | Format-Table Name, Length -AutoSize | Out-Host
