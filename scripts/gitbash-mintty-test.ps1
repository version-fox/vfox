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

Add-Type -Namespace Vfox -Name Win32 -MemberDefinition @'
[DllImport("user32.dll", CharSet = CharSet.Auto)]
public static extern bool PostMessage(IntPtr hWnd, uint msg, IntPtr wParam, IntPtr lParam);
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

$deadline = (Get-Date).AddSeconds($TimeoutSeconds)
while ($window.MainWindowHandle -eq [IntPtr]::Zero) {
    if ((Get-Date) -gt $deadline) {
        Write-Host "UserInteractive: $([Environment]::UserInteractive)"
        throw 'mintty opened no window: is there a desktop session?'
    }
    Start-Sleep -Milliseconds 300
    $window.Refresh()
}
$handle = $window.MainWindowHandle
Write-Host "mintty window $handle, process $($window.Id)"

function Send-Line([IntPtr] $target, [string] $text) {
    foreach ($char in $text.ToCharArray()) {
        [void][Vfox.Win32]::PostMessage($target, $WM_CHAR, [IntPtr][int][char]$char, [IntPtr]1)
        Start-Sleep -Milliseconds 100
    }
    [void][Vfox.Win32]::PostMessage($target, $WM_CHAR, [IntPtr]13, [IntPtr]1)
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
