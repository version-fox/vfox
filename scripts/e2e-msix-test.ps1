#!/usr/bin/env pwsh

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
    Integration tests for the vfox MSIX packaging pipeline.

.DESCRIPTION
    Must run on Windows (Windows 10 1809+ / Windows Server 2025 or newer).
    Builds vfox.exe for x86/x64/arm64, packs them into an .msixbundle via
    msix/make-msix.ps1, signs it with an ephemeral self-signed certificate,
    installs the bundle through the AppX deployment service, verifies the
    `vfox` CLI through its app execution alias, then uninstalls it.

.PARAMETER OutputDir
    Directory for artifacts (packages, logs). Defaults to <repo>/msix-e2e-out.
#>

param(
    [string]$OutputDir = (Join-Path (Split-Path $PSScriptRoot -Parent) "msix-e2e-out")
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$GREEN = "`e[32m"
$RED = "`e[31m"
$YELLOW = "`e[33m"
$NC = "`e[0m"

$RepoRoot = Split-Path $PSScriptRoot -Parent
$TestVersion = "9.9.9"
$PfxPassword = "vfox-e2e-password"
$PublisherSubject = "CN=VersionFox"
$PackageIdentityName = "VersionFox.vfox"

$TEST_COUNT = 0
$PASSED = 0
$FAILED = 0

$InstalledPackage = $null
$E2ECert = $null
$pfxPath = $null

# make-msix.ps1 takes no parameters and reads its version from
# internal/version.go, so the e2e run temporarily points the source tree at
# the throwaway test version. The original content is restored in finally.
$versionFile = Join-Path $RepoRoot "internal/version.go"
$originalVersionGo = $null
$versionPatched = $false
$msixOutDir = Join-Path $RepoRoot "packaging/msix/Output"

function Write-Banner {
    param([string]$Title, [ConsoleColor]$Color = [ConsoleColor]::White)
    $bar = "=" * 42
    Write-Host ""
    Write-Host $bar
    Write-Host $Title -ForegroundColor $Color
    Write-Host $bar
}

Write-Banner "vFox MSIX Packaging E2E Test" Cyan

function Run-Test {
    param(
        [string]$TestName,
        [scriptblock]$TestScript,
        [string]$ExpectedOutput
    )

    $script:TEST_COUNT++

    Write-Host "`nTest $($script:TEST_COUNT): $TestName" -ForegroundColor Yellow
    Write-Host "Running test..."

    try {
        $result = & $TestScript 2>&1 | Out-String

        if ($result -match [regex]::Escape($ExpectedOutput)) {
            Write-Host "$GREEN✓ PASSED$NC" -ForegroundColor Green
            $script:PASSED++
        }
        else {
            Write-Host "$RED✗ FAILED$NC" -ForegroundColor Red
            Write-Host "Expected: $ExpectedOutput"
            Write-Host "Got: $result"
            $script:FAILED++
        }
    }
    catch {
        Write-Host "$RED✗ FAILED$NC" -ForegroundColor Red
        Write-Host "Error: $_"
        $script:FAILED++
    }
}

try {
    # Export-PfxCertificate requires the target directory to exist.
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null

    # Fail fast instead of clobbering an existing installation: cleanup only
    # removes what this run installs.
    if ($null -ne (Get-AppxPackage -Name $PackageIdentityName -ErrorAction SilentlyContinue)) {
        throw "A package with identity '$PackageIdentityName' is already installed. Uninstall it first to avoid clobbering your installation."
    }

    # ------------------------------------------------------------------
    # Step 1: Create ephemeral code-signing certificate
    # ------------------------------------------------------------------
    Write-Banner "Step 1: Creating self-signed certificate"

    $E2ECert = New-SelfSignedCertificate -Type Custom `
        -Subject $PublisherSubject `
        -KeyUsage DigitalSignature `
        -FriendlyName "vfox MSIX E2E (ephemeral)" `
        -CertStoreLocation "Cert:\CurrentUser\My" `
        -NotAfter (Get-Date).AddDays(3) `
        -TextExtension @("2.5.29.37={text}1.3.6.1.5.5.7.3.3", "2.5.29.19={text}")

    # Keep the PFX (private key) outside of $OutputDir: that directory is
    # uploaded as a CI artifact and must never contain signing material.
    $tempRoot = if ($env:RUNNER_TEMP) { $env:RUNNER_TEMP } else { [System.IO.Path]::GetTempPath() }
    $pfxPath = Join-Path $tempRoot "vfox-e2e-$PID.pfx"
    $securePassword = ConvertTo-SecureString -String $PfxPassword -Force -AsPlainText
    Export-PfxCertificate -Cert $E2ECert -FilePath $pfxPath -Password $securePassword | Out-Null
    Write-Host "Certificate created: $($E2ECert.Thumbprint)" -ForegroundColor Green

    # ------------------------------------------------------------------
    # Step 2: Build + pack + sign the bundle
    # ------------------------------------------------------------------
    Write-Banner "Step 2: Building and packing MSIX bundle"

    $packageDir = Join-Path $OutputDir "packages"
    New-Item -ItemType Directory -Path $packageDir -Force | Out-Null

    # Point the source tree at the throwaway test version so the bundle (and
    # the binary inside it) reports 9.9.9 instead of the real release version.
    # This also keeps the test isolated from any vfox already installed.
    $originalVersionGo = Get-Content -Raw -Path $versionFile
    if ($originalVersionGo -notmatch '\b(var|const)\s+RuntimeVersion\s*=') {
        throw "RuntimeVersion not found in $versionFile"
    }
    $patchedVersionGo = $originalVersionGo -replace '\b(var|const)\s+RuntimeVersion\s*=\s*"[^"]*"', ('var RuntimeVersion = "{0}"' -f $TestVersion)
    Set-Content -Path $versionFile -Value $patchedVersionGo -Encoding UTF8 -NoNewline
    $versionPatched = $true
    Write-Host "Patched $versionFile to test version $TestVersion"

    # make-msix.ps1 takes no parameters: version comes from the patched source
    # above, signing inputs come from the environment.
    $env:MSIX_SIGN_PFX_PATH = $pfxPath
    $env:MSIX_SIGN_PFX_PASSWORD = $PfxPassword
    try {
        & (Join-Path $RepoRoot "packaging/msix/make-msix.ps1")
        if ($LASTEXITCODE -ne 0) {
            throw "make-msix.ps1 failed with exit code $LASTEXITCODE"
        }
    }
    finally {
        Remove-Item Env:MSIX_SIGN_PFX_PATH -ErrorAction SilentlyContinue
        Remove-Item Env:MSIX_SIGN_PFX_PASSWORD -ErrorAction SilentlyContinue
    }

    $bundlePath = Join-Path $msixOutDir ("vfox_{0}_windows.msixbundle" -f $TestVersion)
    if (-not (Test-Path $bundlePath)) {
        throw "Bundle not found: $bundlePath"
    }
    # Copy the bundle into the e2e output directory: $OutputDir is uploaded as
    # a CI artifact, while packaging/msix/Output is a gitignored scratch dir.
    Copy-Item -Path $bundlePath -Destination (Join-Path $packageDir (Split-Path $bundlePath -Leaf))

    # ------------------------------------------------------------------
    # Step 3: Trust certificate and install the bundle
    # ------------------------------------------------------------------
    Write-Banner "Step 3: Installing MSIX bundle"

    $trustedPeople = "Cert:\LocalMachine\TrustedPeople"
    Import-PfxCertificate -FilePath $pfxPath -CertStoreLocation $trustedPeople -Password $securePassword | Out-Null
    Write-Host "Certificate imported into TrustedPeople"

    Add-AppxPackage -Path $bundlePath
    $InstalledPackage = Get-AppxPackage -Name $PackageIdentityName
    if ($null -eq $InstalledPackage) {
        throw "Package was not registered after Add-AppxPackage"
    }
    Write-Host "Installed: $($InstalledPackage.PackageFullName)" -ForegroundColor Green

    # ------------------------------------------------------------------
    # Step 4: Assertions
    # ------------------------------------------------------------------
    $aliasPath = Join-Path $env:LOCALAPPDATA "Microsoft\WindowsApps\vfox.exe"

    Run-Test "Per-architecture packages were produced" `
        {
            $missing = @(@("x86", "x64", "arm64") |
                ForEach-Object { Join-Path $msixOutDir "vfox_${TestVersion}_windows_$_.msix" } |
                Where-Object { -not (Test-Path $_) })
            if ($missing.Count -gt 0) {
                "MISSING: $($missing -join ', ')"
            }
            elseif (-not (Test-Path $bundlePath)) { "MISSING_BUNDLE" }
            else { "ALL_PACKAGES_PRESENT" }
        } `
        "ALL_PACKAGES_PRESENT"

    Run-Test "Package registered with normalized version" `
        {
            $pkg = Get-AppxPackage -Name $PackageIdentityName
            if ($null -eq $pkg) { "PACKAGE_NOT_FOUND" }
            elseif ($pkg.Version -eq "$TestVersion.0" -and $pkg.Publisher -eq $PublisherSubject) {
                "VERSION_OK"
            }
            else { "UNEXPECTED: $($pkg.Version) / $($pkg.Publisher)" }
        } `
        "VERSION_OK"

    Run-Test "App execution alias exists" `
        {
            $deadline = (Get-Date).AddSeconds(30)
            while (-not (Test-Path $aliasPath) -and (Get-Date) -lt $deadline) {
                Start-Sleep -Milliseconds 500
            }
            if (Test-Path $aliasPath) { "ALIAS_EXISTS" } else { "ALIAS_MISSING" }
        } `
        "ALIAS_EXISTS"

    Run-Test "vfox runs through execution alias" `
        {
            $versionOutput = & $aliasPath --version 2>&1 | Out-String
            if ($versionOutput -match '\d+\.\d+\.\d+') { "CLI_OK: $($versionOutput.Trim())" } else { "BAD_OUTPUT: $versionOutput" }
        } `
        "CLI_OK"

    Run-Test "vfox --help through execution alias" `
        {
            $helpOutput = & $aliasPath --help 2>&1 | Out-String
            if ($helpOutput -match 'Usage') { "HELP_OK" } else { "BAD_OUTPUT: $helpOutput" }
        } `
        "HELP_OK"

    Run-Test "Packaged binary keeps user data outside WindowsApps" `
        {
            $tempRoot = if ($env:RUNNER_TEMP) { $env:RUNNER_TEMP } else { [System.IO.Path]::GetTempPath() }
            $probeHome = Join-Path $tempRoot "vfox-msix-probe-home"
            New-Item -ItemType Directory -Path $probeHome -Force | Out-Null
            $originalProfile = $env:USERPROFILE
            $env:USERPROFILE = $probeHome
            try {
                # `list` initializes SdkManager/pathmeta (unlike --version/--help,
                # which are handled at CLI routing), so ~/.vfox gets created.
                $null = & $aliasPath list 2>&1
                if (Test-Path (Join-Path $probeHome ".vfox")) { "USER_DATA_OK" } else { "NO_USER_DATA" }
            }
            finally {
                $env:USERPROFILE = $originalProfile
                Remove-Item -Path $probeHome -Recurse -Force -ErrorAction SilentlyContinue
            }
        } `
        "USER_DATA_OK"

    # ------------------------------------------------------------------
    # Step 5: Uninstall
    # ------------------------------------------------------------------
    Write-Banner "Step 5: Uninstalling MSIX bundle"

    Run-Test "Uninstall removes the package and alias" `
        {
            $pkg = Get-AppxPackage -Name $PackageIdentityName
            if ($null -eq $pkg) { return "PACKAGE_NOT_FOUND_BEFORE_UNINSTALL" }
            Remove-AppxPackage -Package $pkg.PackageFullName
            $script:InstalledPackage = $null
            $deadline = (Get-Date).AddSeconds(30)
            while ((Get-AppxPackage -Name $PackageIdentityName) -and (Get-Date) -lt $deadline) {
                Start-Sleep -Milliseconds 500
            }
            if ((Get-AppxPackage -Name $PackageIdentityName)) { return "PACKAGE_STILL_REGISTERED" }
            if (Test-Path $aliasPath) { return "ALIAS_STILL_EXISTS" }
            return "UNINSTALL_OK"
        } `
        "UNINSTALL_OK"
}
catch {
    Write-Host "`n$RED✗ Fatal error: $_$NC" -ForegroundColor Red
    $script:FAILED++
}
finally {
    Write-Banner "Cleanup"

    # Restore the source tree first so a failed run never leaves the
    # throwaway test version behind.
    if ($versionPatched -and $null -ne $originalVersionGo) {
        Set-Content -Path $versionFile -Value $originalVersionGo -Encoding UTF8 -NoNewline
        Write-Host "Restored $versionFile" -ForegroundColor Yellow
    }
    Remove-Item Env:MSIX_SIGN_PFX_PATH -ErrorAction SilentlyContinue
    Remove-Item Env:MSIX_SIGN_PFX_PASSWORD -ErrorAction SilentlyContinue
    # Remove the throwaway test artifacts from the packaging scratch dir
    # (the copies under $OutputDir are kept for the CI artifact upload).
    if ($null -ne $msixOutDir -and (Test-Path $msixOutDir)) {
        Get-ChildItem -Path $msixOutDir -Filter ("vfox_{0}_windows*" -f $TestVersion) -ErrorAction SilentlyContinue |
            Remove-Item -Force -ErrorAction SilentlyContinue
    }

    # Only remove the package installed by this run (exact identity), never
    # an unrelated package that happens to match a wildcard.
    if ($null -ne $InstalledPackage) {
        Remove-AppxPackage -Package $InstalledPackage.PackageFullName -ErrorAction SilentlyContinue
        Write-Host "Removed leftover AppX package" -ForegroundColor Yellow
    }
    if ($null -ne $E2ECert) {
        Remove-Item -Path "Cert:\CurrentUser\My\$($E2ECert.Thumbprint)" -Force -ErrorAction SilentlyContinue
        Remove-Item -Path "Cert:\LocalMachine\TrustedPeople\$($E2ECert.Thumbprint)" -Force -ErrorAction SilentlyContinue
        Write-Host "Removed ephemeral certificate" -ForegroundColor Yellow
    }
    if ($null -ne $pfxPath -and (Test-Path $pfxPath)) {
        Remove-Item -Path $pfxPath -Force -ErrorAction SilentlyContinue
    }
}

# Print summary
Write-Banner "Test Summary"
Write-Host "Total tests: $($script:TEST_COUNT)"
Write-Host "Passed: $($script:PASSED)" -ForegroundColor Green
Write-Host "Failed: $($script:FAILED)" -ForegroundColor Red

if ($script:FAILED -eq 0) {
    Write-Host "`nAll tests passed! ✓" -ForegroundColor Green
    exit 0
}
else {
    Write-Host "`nSome tests failed! ✗" -ForegroundColor Red
    exit 1
}
