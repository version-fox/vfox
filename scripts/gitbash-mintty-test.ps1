<#
.SYNOPSIS
    Runs the terminal code in a real mintty window and types the answers into it.

.DESCRIPTION
    Starts Git for Windows' mintty.exe running Git for Windows' bash.exe, and in
    that bash the test binary built from cmd/commands: TestGitBashMinttySelection.
    The test creates a marker file in -WorkDir when it waits for a choice. This
    script then types the choice into the window with WM_CHAR messages, each
    followed by Enter: "2" for the selection, then "y" for the confirmation. It
    fails unless the test passes.

    It needs a desktop session, which GitHub-hosted Windows runners have, and it
    types into a terminal window. Run it on a disposable runner or VM only.
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string] $TestBinary,

    [Parameter(Mandatory)]
    [string] $WorkDir,

    [ValidateRange(10, 1800)]
    [int] $TimeoutSeconds = 180
)

$ErrorActionPreference = 'Stop'

Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Runtime.InteropServices;
using System.Text;

public static class VfoxWindow
{
    public delegate bool EnumProc(IntPtr hWnd, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool EnumWindows(EnumProc callback, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId);

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    public static extern int GetClassName(IntPtr hWnd, StringBuilder className, int maxCount);

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    public static extern int GetWindowText(IntPtr hWnd, StringBuilder text, int maxCount);

    [DllImport("user32.dll")]
    public static extern bool PostMessage(IntPtr hWnd, uint msg, IntPtr wParam, IntPtr lParam);

    public static List<IntPtr> TopLevel()
    {
        List<IntPtr> found = new List<IntPtr>();
        EnumWindows(delegate (IntPtr hWnd, IntPtr lParam) { found.Add(hWnd); return true; }, IntPtr.Zero);
        return found;
    }

    public static string ClassOf(IntPtr hWnd)
    {
        StringBuilder text = new StringBuilder(256);
        GetClassName(hWnd, text, text.Capacity);
        return text.ToString();
    }

    public static string TitleOf(IntPtr hWnd)
    {
        StringBuilder text = new StringBuilder(512);
        GetWindowText(hWnd, text, text.Capacity);
        return text.ToString();
    }

    public static uint ProcessOf(IntPtr hWnd)
    {
        uint processId;
        GetWindowThreadProcessId(hWnd, out processId);
        return processId;
    }
}
'@

$WM_CHAR = 0x0102
$git = 'C:\Program Files\Git'
$bash = Join-Path $git 'bin\bash.exe'
$mintty = Join-Path $git 'usr\bin\mintty.exe'
foreach ($tool in $bash, $mintty, $TestBinary) {
    if (-not (Test-Path -LiteralPath $tool -PathType Leaf)) { throw "missing $tool" }
}

New-Item -ItemType Directory -Force -Path $WorkDir | Out-Null
$WorkDir = (Resolve-Path -LiteralPath $WorkDir).Path
Get-ChildItem -LiteralPath $WorkDir | Remove-Item -Force
$unixWork = (& $bash --noprofile --norc -c "cygpath -u '$WorkDir'").Trim()
$unixTest = (& $bash --noprofile --norc -c "cygpath -u '$TestBinary'").Trim()
$unixScript = "$unixWork/run.sh"

# The script runs in Git Bash inside mintty. Its exit status goes to a file, and
# the window closes when the script ends.
$runScript = @'
#!/usr/bin/env bash
export VFOX_TEST_GITBASH_MINTTY_DIR='__WORK__'
'__TEST__' -test.run '^TestGitBashMinttySelection$' -test.v
echo $? > '__UNIXWORK__/exit.txt'
'@
$runScript = $runScript.Replace('__WORK__', $WorkDir).Replace('__TEST__', $unixTest).Replace('__UNIXWORK__', $unixWork)
[System.IO.File]::WriteAllText((Join-Path $WorkDir 'run.sh'), $runScript.Replace("`r`n", "`n") + "`n")

$window = Start-Process -FilePath $mintty -ArgumentList "-e `"$bash`" --noprofile --norc `"$unixScript`"" -PassThru
Write-Host "started $mintty as process $($window.Id)"

function Find-MinttyWindow {
    foreach ($candidate in [VfoxWindow]::TopLevel()) {
        if ([VfoxWindow]::ClassOf($candidate) -eq 'mintty') { return $candidate }
    }
    return [IntPtr]::Zero
}

$deadline = (Get-Date).AddSeconds($TimeoutSeconds)
$handle = Find-MinttyWindow
while ($handle -eq [IntPtr]::Zero) {
    if ($window.HasExited) { throw "mintty exited with code $($window.ExitCode) before it showed a window" }
    if ((Get-Date) -gt $deadline) {
        Write-Host "UserInteractive: $([Environment]::UserInteractive)"
        foreach ($candidate in [VfoxWindow]::TopLevel()) {
            $title = [VfoxWindow]::TitleOf($candidate)
            if ($title) {
                Write-Host ("window {0} class '{1}' title '{2}' process {3}" -f $candidate, [VfoxWindow]::ClassOf($candidate), $title, [VfoxWindow]::ProcessOf($candidate))
            }
        }
        Get-CimInstance Win32_Process -Filter "Name = 'mintty.exe'" | ForEach-Object { Write-Host "mintty.exe $($_.ProcessId): $($_.CommandLine)" }
        throw 'no mintty window appeared'
    }
    Start-Sleep -Milliseconds 300
    $handle = Find-MinttyWindow
}
Write-Host "mintty window $handle, process $([VfoxWindow]::ProcessOf($handle))"

function Send-Line([IntPtr] $target, [string] $text) {
    foreach ($char in $text.ToCharArray()) {
        [void][VfoxWindow]::PostMessage($target, $WM_CHAR, [IntPtr][int][char]$char, [IntPtr]1)
        Start-Sleep -Milliseconds 100
    }
    [void][VfoxWindow]::PostMessage($target, $WM_CHAR, [IntPtr]13, [IntPtr]1)
}

function Wait-Marker([string] $name) {
    $path = Join-Path $WorkDir $name
    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while (-not (Test-Path -LiteralPath $path)) {
        if ((Get-Date) -gt $deadline) { throw "timed out waiting for $name" }
        Start-Sleep -Milliseconds 200
    }
}

try {
    Wait-Marker 'select.ready'
    Start-Sleep -Seconds 1
    Send-Line $handle '2'
    Wait-Marker 'confirm.ready'
    Start-Sleep -Seconds 1
    Send-Line $handle 'y'
    Wait-Marker 'exit.txt'
}
finally {
    if (-not $window.HasExited -and -not $window.WaitForExit(10000)) {
        Stop-Process -Id $window.Id -Force
    }
    Write-Host "files in ${WorkDir}:"
    Get-ChildItem -LiteralPath $WorkDir | ForEach-Object { Write-Host "  $($_.Name)" }
}

$exitCode = (Get-Content -LiteralPath (Join-Path $WorkDir 'exit.txt') -Raw).Trim()
Write-Host '--- result.txt'
Get-Content -LiteralPath (Join-Path $WorkDir 'result.txt') -ErrorAction SilentlyContinue | Write-Host
if ($exitCode -ne '0') { throw "the test failed in mintty with exit code $exitCode" }
Write-Host 'Git Bash in mintty: selection and confirmation passed.'
