package commands

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/shared/util"
)

// powerShellProbe runs in PowerShell with its console reading native output as
// code page 936, the one Simplified Chinese Windows uses. It activates vfox the
// way the README does, then prints the session as base64, so that the report
// does not depend on the code page either. The errors come last: they may span
// lines.
const powerShellProbe = `
$ErrorActionPreference = 'Continue'
try { [System.Text.Encoding]::RegisterProvider([System.Text.CodePagesEncodingProvider]::Instance) } catch { }
[Console]::OutputEncoding = [System.Text.Encoding]::GetEncoding(936)
$codePage = [Console]::OutputEncoding.CodePage
$Error.Clear()
Invoke-Expression "$(& $env:VFOX_TEST_VFOX activate pwsh)"
$nl = [char]10
$hookErrors = ($Error | ForEach-Object { $_.ToString() }) -join $nl
$report = 'CP=' + $codePage + $nl + 'PATH=' + $env:PATH + $nl + 'INITIALIZED=' + $env:__VFOX_INITIALIZED + $nl + 'ERRORS=' + $hookErrors
[Console]::Out.Write([System.Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes($report)))
`

// powerShells returns the PowerShells that run the session test: Windows
// PowerShell and PowerShell 7, whichever is installed.
func powerShells(t *testing.T) []string {
	t.Helper()
	var found []string
	for _, name := range []string{"powershell", "pwsh"} {
		if path, err := exec.LookPath(name); err == nil {
			found = append(found, path)
		}
	}
	if len(found) == 0 {
		t.Skip("neither powershell nor pwsh is installed")
	}
	return found
}

// probeFields decodes the report that powerShellProbe prints.
func probeFields(t *testing.T, r cliResult) (codePage, path, initialized, hookErrors string) {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(r.stdout))
	if err != nil {
		t.Fatalf("the probe printed no base64 report (%v)\nstdout: %q\nstderr: %s", err, r.stdout, r.stderr)
	}
	lines := strings.SplitN(string(raw), "\n", 4)
	if len(lines) != 4 {
		t.Fatalf("the probe report has %d lines, want 4:\n%s\nstderr: %s", len(lines), raw, r.stderr)
	}
	return strings.TrimPrefix(lines[0], "CP="), strings.TrimPrefix(lines[1], "PATH="),
		strings.TrimPrefix(lines[2], "INITIALIZED="), strings.TrimPrefix(lines[3], "ERRORS=")
}

// TestPowerShellSessionKeepsNonASCIIPath activates vfox in PowerShell whose
// console reads native output with a Chinese code page, and checks that the
// session starts with the non-ASCII PATH entry it was given. The vfox executable
// lives below a non-ASCII directory too, as it does for a user whose name is not
// ASCII. Windows only: the console code page is a Windows setting.
func TestPowerShellSessionKeepsNonASCIIPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the console code page that garbles the output is a Windows setting")
	}
	f := newCLIFixture(t, false)
	vfox := buildVfox(t)
	vfoxDir := filepath.Join(f.root, "中文", "bin")
	must(t, os.MkdirAll(vfoxDir, 0755))
	vfoxExe := filepath.Join(vfoxDir, "vfox.exe")
	must(t, os.Rename(vfox, vfoxExe))
	entry := filepath.Join(f.root, "用户工具", "bin")
	must(t, os.MkdirAll(entry, 0755))
	f.env["PATH"] = pathList(entry, f.system)
	f.env["VFOX_TEST_VFOX"] = vfoxExe

	for _, shell := range powerShells(t) {
		t.Run(filepath.Base(shell), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			r := runCommand(t, f.command(ctx, shell, "-NoProfile", "-NonInteractive", "-EncodedCommand", util.PowerShellEncodedCommand(powerShellProbe)))
			codePage, path, initialized, hookErrors := probeFields(t, r)
			if codePage != "936" {
				t.Fatalf("the console took code page %q, not 936, so the output is not read the way a Chinese Windows reads it\nstderr: %s", codePage, r.stderr)
			}
			if !slices.Contains(strings.Split(path, string(os.PathListSeparator)), entry) {
				t.Errorf("the session starts with PATH %q, which lost the entry %q", path, entry)
			}
			if initialized != "1" {
				t.Errorf("the hook did not finish (__VFOX_INITIALIZED = %q)", initialized)
			}
			if hookErrors != "" {
				t.Errorf("activating vfox reported errors:\n%s", hookErrors)
			}
		})
	}
}
