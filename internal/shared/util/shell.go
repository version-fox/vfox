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
// script itself contains spaces, quotes, or backslashes. This is the core
// of the Windows path-with-spaces fix: Go's exec.Command escapes arguments
// with syscall.EscapeArg (doubling backslashes and turning " into \"),
// which cmd.exe misinterprets, but a base64 blob has none of those
// characters.
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

// ShellCommand builds an *exec.Cmd that runs the given shell command string
// with correct quoting. On Windows it invokes PowerShell with
// -EncodedCommand so paths containing spaces, quotes, or backslashes survive
// the cmd-line layer. On Unix it uses /bin/sh -c.
func ShellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("powershell.exe", powershellArgs(command)...)
	}
	return exec.Command("/bin/sh", "-c", command)
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
