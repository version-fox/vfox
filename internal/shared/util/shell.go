// Copyright 2026 vfox project contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"encoding/base64"
	"encoding/binary"
	"os/exec"
	"runtime"
	"strings"
	"unicode/utf16"
)

// psSingleQuote returns s wrapped in a PowerShell single-quoted string
// literal. Inside a single-quoted literal the only special character is
// the single quote itself, which is escaped by doubling it. Spaces, double
// quotes, backslashes and angle brackets are all preserved literally, so
// this is safe for embedding Windows paths.
func psSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// PowerShellEncodedCommand encodes script as a UTF-16LE base64 string
// suitable for powershell.exe -EncodedCommand. The result consists only of
// [A-Za-z0-9+/=], so it survives the cmd-line layer unchanged even when the
// script itself contains spaces, quotes, or backslashes. PowerShell-only
// call sites (junction creation, upgrade launcher) use this transport via
// PowerShellCommand.
func PowerShellEncodedCommand(script string) string {
	utf16CodeUnits := utf16.Encode([]rune(script))
	buf := make([]byte, len(utf16CodeUnits)*2)
	for i, u := range utf16CodeUnits {
		binary.LittleEndian.PutUint16(buf[i*2:], u)
	}
	return base64.StdEncoding.EncodeToString(buf)
}

// powershellArgs returns the argument slice that runs the given PowerShell
// script without any shell interpretation. The script is passed as a base64
// blob via -EncodedCommand, so spaces, quotes, and backslashes in the
// script are immune to Go's EscapeArg mangling.
func powershellArgs(script string) []string {
	return []string{
		"-NoProfile",
		"-NoLogo",
		"-NonInteractive",
		"-EncodedCommand",
		PowerShellEncodedCommand(script),
	}
}

// windowsCmdLine builds the raw Windows command line that runs command via
// cmd.exe: `<comSpec> /d /c <command>`. The command text is appended
// verbatim so cmd builtins (dir, mklink), switches (/b, /ad), quotes, and
// operators (&&, |, >) keep their exact cmd meaning, identical to typing the
// command in a terminal. `/d` skips AutoRun so launches stay hermetic.
// Kept as a pure cross-platform function so it is unit-testable on any OS;
// windowsShellCommand (in shell_windows.go) passes its result to the OS via
// SysProcAttr.CmdLine, bypassing Go's EscapeArg mangling entirely.
func windowsCmdLine(comSpec, command string) string {
	if strings.Contains(comSpec, " ") && !(strings.HasPrefix(comSpec, `"`) && strings.HasSuffix(comSpec, `"`)) {
		comSpec = `"` + comSpec + `"`
	}
	return comSpec + " /d /c " + command
}

// PowerShellCommand builds an *exec.Cmd that runs the given PowerShell
// script without any shell interpretation. Use this for PowerShell syntax
// (for example New-Item or `& 'path'`). On Unix it falls back to
// /bin/sh -c so callers remain testable; the Windows-only call sites guard
// with IsWindows or runtime checks.
func PowerShellCommand(script string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("powershell.exe", powershellArgs(script)...)
	}
	return exec.Command("/bin/sh", "-c", script)
}

// ShellCommand builds an *exec.Cmd that runs the given shell command string
// with the platform's default shell: cmd.exe on Windows, /bin/sh on Unix.
// On Windows it invokes cmd.exe directly with an explicit raw command line
// (see windowsCmdLine), so paths containing spaces, quotes, or backslashes
// survive Go's EscapeArg mangling without changing the shell language.
// cmd.exe is used instead of powershell.exe because powershell.exe creates
// per-user profile directories under %USERPROFILE% on startup, which breaks
// hermetic runs (e.g. plugin development commands must not touch the user's
// home directory).
func ShellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return windowsShellCommand(command)
	}
	return exec.Command("/bin/sh", "-c", command)
}

// RunPowerShellScript runs a PowerShell script via PowerShellCommand and
// returns combined stdout and stderr together with any error.
func RunPowerShellScript(script string) (output string, err error) {
	out, err := PowerShellCommand(script).CombinedOutput()
	return string(out), err
}

// RunShellScript runs command via ShellCommand and returns combined stdout
// and stderr together with any error. The error's message preserves the
// command output when the command exits non-zero.
func RunShellScript(command string) (output string, err error) {
	out, err := ShellCommand(command).CombinedOutput()
	return string(out), err
}

// IsWindows reports whether the current platform is Windows. Kept as a tiny
// helper so callers do not have to import runtime just for a GOOS check.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}
