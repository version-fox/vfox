#!/usr/bin/env pwsh

# E2E Test Script for vFox - Windows PowerShell Version
# Tests vfox functionality directly

param(
    [switch]$Verbose = $false
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$GREEN = "`e[32m"
$RED = "`e[31m"
$YELLOW = "`e[33m"
$NC = "`e[0m"

$OriginalUserProfile = $env:USERPROFILE
$VfoxHomeDir = if ($env:VFOX_HOME) { $env:VFOX_HOME } else { Join-Path $OriginalUserProfile ".vfox" }
$UserHomeRoot = if ($VfoxHomeDir -match "\\.vfox$") { Split-Path $VfoxHomeDir -Parent } else { $OriginalUserProfile }
$env:VFOX_HOME = $VfoxHomeDir
$env:USERPROFILE = $UserHomeRoot

$TEST_COUNT = 0
$PASSED = 0
$FAILED = 0

Write-Host "=========================================="
Write-Host "vFox E2E Test - Windows PowerShell"
Write-Host "==========================================" -ForegroundColor Cyan

function Cleanup {
    Write-Host "`n=========================================="
    Write-Host "Cleanup"
    Write-Host "==========================================" -ForegroundColor White

    Remove-Item -Path (Join-Path $VfoxHomeDir "cache/nodejs") -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item -Path (Join-Path $VfoxHomeDir "plugin/nodejs") -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item -Path (Join-Path $VfoxHomeDir "sdks") -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item -Path (Join-Path $VfoxHomeDir "tmp") -Recurse -Force -ErrorAction SilentlyContinue

    Write-Host "Cleanup completed" -ForegroundColor Green
}

$null = Register-EngineEvent -SourceIdentifier PowerShell.Exiting -Action { Cleanup }

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

# Build vfox
Write-Host "`n=========================================="
Write-Host "Building vFox"
Write-Host "==========================================" -ForegroundColor White

try {
    $buildOutput = & go build -o vfox.exe . 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Build completed" -ForegroundColor Green
    }
    else {
        Write-Host "Build failed: $buildOutput" -ForegroundColor Red
        exit 1
    }
}
catch {
    Write-Host "Build failed: $_" -ForegroundColor Red
    exit 1
}

# Define vfox executable path
$vfoxExe = Join-Path (Get-Location) "vfox.exe"

function Activate-Vfox {
    $activationScript = (& $vfoxExe activate pwsh 2>&1) -join "`n"
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to activate vfox: $activationScript"
    }
    Invoke-Expression $activationScript
}

# Setup
Write-Host "`n=========================================="
Write-Host "Setup: Activating vfox"
Write-Host "==========================================" -ForegroundColor White

try {
    $pathsToEnsure = @()
    $pathsToEnsure += $VfoxHomeDir
    $pathsToEnsure += Join-Path $VfoxHomeDir "plugin"
    $pathsToEnsure += Join-Path $VfoxHomeDir "cache"
    $pathsToEnsure += Join-Path $VfoxHomeDir "sdks"
    $pathsToEnsure += Join-Path $VfoxHomeDir "tmp"
    foreach ($path in $pathsToEnsure) {
        New-Item -ItemType Directory -Path $path -Force | Out-Null
    }

    Activate-Vfox
    try {
        & $vfoxExe add nodejs 2>&1 | Out-Null
    }
    catch {
        Write-Host "Add plugin warning: $_" -ForegroundColor Yellow
    }

    $installOutput = & $vfoxExe install -y nodejs@18.19.0 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Failed to install nodejs@18.19.0:`n$installOutput" -ForegroundColor Red
        exit 1
    }

    Write-Host "Setup completed" -ForegroundColor Green
}
catch {
    Write-Host "Setup warning: $_" -ForegroundColor Yellow
}

# Run tests
Write-Host "`n=========================================="
Write-Host "Running E2E Tests"
Write-Host "==========================================" -ForegroundColor White

$LegacyVfoxDir = Join-Path $UserHomeRoot ".version-fox"
$GlobalUserVfoxDir = if (Test-Path $LegacyVfoxDir) { $LegacyVfoxDir } else { $VfoxHomeDir }

Run-Test "Check vfox command available" `
    { Get-Command $vfoxExe -ErrorAction Stop | Select-Object -ExpandProperty Source } `
    "vfox.exe"

Run-Test "Use nodejs in session scope" `
    {
        & $vfoxExe use -s nodejs@18.19.0 2>&1 | Out-Null
        Activate-Vfox
        & $vfoxExe current nodejs 2>&1
    } `
    "18.19"

Run-Test "Check current version" `
    { & $vfoxExe current nodejs 2>&1 } `
    "18.19"

Run-Test "List available versions" `
    { & $vfoxExe available 2>&1 | Select-String "nodejs" } `
    "nodejs"

Run-Test "Use nodejs in global scope" `
    {
        & $vfoxExe use -g nodejs@18.19.0 2>&1 | Out-Null
        Activate-Vfox
        & $vfoxExe current nodejs 2>&1
    } `
    "18.19"

Run-Test "Verify global symlink exists" `
    {
        $sdkPath = Join-Path $GlobalUserVfoxDir "sdks/nodejs"
        if (Test-Path $sdkPath) { "SYMLINK_EXISTS" } else { "SYMLINK_MISSING" }
    } `
    "SYMLINK_EXISTS"

Run-Test "Verify global config file location" `
    {
        $configPath = Join-Path $GlobalUserVfoxDir "config.yaml"
        if (Test-Path $configPath) { "CONFIG_EXISTS" } else { "CONFIG_NOT_FOUND" }
    } `
    "CONFIG_EXISTS"

Run-Test "Verify global SDK installation path" `
    {
        $installPath = Join-Path $VfoxHomeDir "cache/nodejs/v-18.19.0"
        if (Test-Path $installPath) { "INSTALL_PATH_OK" } else { "PATH_NOT_FOUND" }
    } `
    "INSTALL_PATH_OK"

Run-Test "Project scope creates .vfox.toml" `
    {
        $testDir = Join-Path ([System.IO.Path]::GetTempPath()) "vfox-test-proj"
        Remove-Item -Path $testDir -Recurse -Force -ErrorAction SilentlyContinue
        New-Item -ItemType Directory -Path $testDir -Force | Out-Null
        Push-Location $testDir
        try {
            & $vfoxExe use -p nodejs@18.19.0 2>&1 | Out-Null
            if (Test-Path ".vfox.toml") { "TOML_CREATED" } else { "TOML_NOT_FOUND" }
        }
        finally {
            Pop-Location
        }
    } `
    "TOML_CREATED"

Run-Test "Session scope does not create .vfox.toml" `
    {
        $testDir = Join-Path ([System.IO.Path]::GetTempPath()) "vfox-session-test"
        Remove-Item -Path $testDir -Recurse -Force -ErrorAction SilentlyContinue
        New-Item -ItemType Directory -Path $testDir -Force | Out-Null
        Push-Location $testDir
        try {
            & $vfoxExe use -s nodejs@18.19.0 2>&1 | Out-Null
            if (-not (Test-Path ".vfox.toml")) { "NO_TOML_CREATED" } else { "TOML_FOUND" }
        }
        finally {
            Pop-Location
        }
    } `
    "NO_TOML_CREATED"

Run-Test "Multiple versions can be installed" `
    {
        & $vfoxExe install -y nodejs@20.11.0 2>&1 | Out-Null
        $list = & $vfoxExe list nodejs 2>&1
        if (($list -match "18.19") -and ($list -match "20.11")) {
            "MULTI_VERSION_OK"
        }
        else {
            "VERSIONS_NOT_FOUND"
        }
    } `
    "MULTI_VERSION_OK"

Run-Test "Use different version in same session" `
    {
        & $vfoxExe use -s nodejs@20.11.0 2>&1 | Out-Null
        $current = & $vfoxExe current nodejs 2>&1
        if ($current -match "20.11") { "VERSION_SWITCH_OK" } else { "VERSION_NOT_CORRECT" }
    } `
    "VERSION_SWITCH_OK"

Run-Test "Uninstall removes version" `
    {
        & $vfoxExe uninstall nodejs@20.11.0 2>&1 | Out-Null
        $list = & $vfoxExe list nodejs 2>&1
        if (-not ($list -match "20.11")) { "UNINSTALL_OK" } else { "VERSION_STILL_EXISTS" }
    } `
    "UNINSTALL_OK"

Run-Test "Unuse global removes symlink" `
    {
        & $vfoxExe unuse -g nodejs 2>&1 | Out-Null
        $sdkPath = Join-Path $VfoxHomeDir "sdks/nodejs"
        if (-not (Test-Path $sdkPath)) { "UNLINK_OK" } else { "SYMLINK_STILL_EXISTS" }
    } `
    "UNLINK_OK"

# Print summary
Write-Host "`n=========================================="
Write-Host "Test Summary"
Write-Host "==========================================" -ForegroundColor White
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

