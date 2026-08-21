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
    Icons are generated from the repository logo, each architecture is built
    from source and packed into its own .msix rendered from AppxManifest.xml,
    and all packages are combined into vfox_<version>_windows.msixbundle.
    Signing is optional: when -SignPfxPath is provided the bundle is signed
    with SignTool; the certificate subject must exactly match -Publisher.

.PARAMETER Version
    Stable vfox version to encode in the manifest, e.g. "1.2.3". Prerelease
    or build metadata segments are rejected; the value is normalized to a
    four-part MSIX version (1.2.3 -> 1.2.3.0).

.PARAMETER OutputDir / WorkDir
    Artifact and scratch directories. Default to <script>/Output and <script>/staging.

.PARAMETER Publisher
    Identity publisher written into the manifest; must match the signing
    certificate subject when signing. Defaults to "CN=VersionFox".

.PARAMETER SignPfxPath / SignPfxPassword
    Optional PFX certificate used to sign the bundle. Falls back to the
    MSIX_SIGN_PFX_PATH / MSIX_SIGN_PFX_PASSWORD environment variables.

.EXAMPLE
    ./make-msix.ps1 -Version 1.2.3
#>

param(
    [Parameter(Mandatory = $true)]
    [string]$Version,

    [string]$OutputDir = (Join-Path $PSScriptRoot "Output"),
    [string]$WorkDir = (Join-Path $PSScriptRoot "staging"),
    [string]$Publisher = "CN=VersionFox",

    [string]$SignPfxPath = $env:MSIX_SIGN_PFX_PATH,
    [string]$SignPfxPassword = $env:MSIX_SIGN_PFX_PASSWORD,

    [switch]$SkipAssets
)

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
    param([string]$Version, [string]$RepoRoot, [string]$BuildDir)

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
            go build -trimpath -ldflags "-s -w -X github.com/version-fox/vfox/internal.RuntimeVersion=$Version" -o (Join-Path $outDir "vfox.exe") .
            Assert-NativeSuccess -Step "go build ($($a.GoArch))"
            Write-Host "Built $($a.Name): $(Join-Path $outDir 'vfox.exe')"
        }
    }
    finally {
        Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
        Pop-Location
    }
}

$msixVersion = ConvertTo-MsixVersion -Raw $Version

# Script lives at <repo>/packaging/msix/, so the repository root is two levels up.
$repoRoot = Split-Path (Split-Path $PSScriptRoot -Parent) -Parent

Write-Host "Building vfox from source..."
Invoke-SourceBuild -Version $Version -RepoRoot $repoRoot -BuildDir (Join-Path $PSScriptRoot "build")
$selected = @()
foreach ($arch in @("x86", "x64", "arm64")) {
    $selected += @{
        Architecture = $arch
        ExePath      = (Resolve-Path (Join-Path $PSScriptRoot "build/$arch/vfox.exe")).Path
    }
}

$makeAppx = Find-SdkTool -ToolName "MakeAppx"

# Generate the tile icons from the repository logo (they are not committed).
if (-not $SkipAssets) {
    & (Join-Path $PSScriptRoot "gen-assets.ps1") `
        -LogoPath (Join-Path $repoRoot "logo.png") `
        -OutDir (Join-Path $PSScriptRoot "assets")
}

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
    # Escape XML entities: the publisher comes from a certificate subject and
    # may contain &, <, > or quotes.
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
        throw "MSIX_SIGN_PFX_PASSWORD (or -SignPfxPassword) is required when signing."
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
