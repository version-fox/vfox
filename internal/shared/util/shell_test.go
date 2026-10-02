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
	"runtime"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestPsSingleQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/tmp/path", "'/tmp/path'"},
		{`C:\Program Files\vfox`, `'C:\Program Files\vfox'`},
		{`path with spaces`, `'path with spaces'`},
		{`path with "quotes"`, `'path with "quotes"'`},
		{`it's a path`, `'it''s a path'`},
		{`C:\Users\Bob\it's vfox`, `'C:\Users\Bob\it''s vfox'`},
		{"", "''"},
	}
	for _, tc := range tests {
		got := psSingleQuote(tc.input)
		if got != tc.expected {
			t.Errorf("psSingleQuote(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestPowerShellEncodedCommand(t *testing.T) {
	tests := []struct {
		script string
		// encoded must not contain spaces, double quotes, backslashes, or
		// single quotes — those are the characters that Go's EscapeArg and
		// cmd.exe mangle. The base64 alphabet is [A-Za-z0-9+/=].
		mustNotContain []string
	}{
		{
			script:         `echo "hello world"`,
			mustNotContain: []string{" ", "\"", "\\"},
		},
		{
			script:         `New-Item -ItemType Junction -Path 'C:\Program Files\vfox' -Target 'C:\Tools\vfox'`,
			mustNotContain: []string{" ", "\\", "'"},
		},
		{
			script:         `& 'C:\Program Files\vfox\upgrade.bat'`,
			mustNotContain: []string{" ", "\\", "'", "`"},
		},
	}
	for _, tc := range tests {
		encoded := PowerShellEncodedCommand(tc.script)
		for _, bad := range tc.mustNotContain {
			if strings.Contains(encoded, bad) {
				t.Errorf("PowerShellEncodedCommand(%q) contains %q: %q", tc.script, bad, encoded)
			}
		}
		// Round-trip: base64-decode, then UTF-16LE-decode, must give back
		// the original script.
		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			t.Fatalf("base64 decode failed: %v", err)
		}
		if len(raw)%2 != 0 {
			t.Fatalf("decoded length %d is not even (not UTF-16LE)", len(raw))
		}
		decoded := make([]uint16, len(raw)/2)
		for i := range decoded {
			decoded[i] = binary.LittleEndian.Uint16(raw[i*2:])
		}
		if got := string(utf16.Decode(decoded)); got != tc.script {
			t.Errorf("round-trip mismatch:\n  want %q\n  got  %q", tc.script, got)
		}
	}
}

func TestPowerShellEncodedCommandEmpty(t *testing.T) {
	// An empty script produces an empty (but valid) base64 string.
	encoded := PowerShellEncodedCommand("")
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	if len(raw) != 0 {
		t.Errorf("empty script should decode to empty bytes, got %d bytes", len(raw))
	}
}

func TestShellCommandUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}
	cmd := ShellCommand(`echo hello`)
	if cmd.Path != "/bin/sh" {
		t.Errorf("ShellCommand path = %q, want /bin/sh", cmd.Path)
	}
	if len(cmd.Args) != 3 || cmd.Args[1] != "-c" || cmd.Args[2] != "echo hello" {
		t.Errorf("ShellCommand args = %v, want [/bin/sh -c 'echo hello']", cmd.Args)
	}
}

func TestWindowsCmdLine(t *testing.T) {
	cases := []struct {
		comSpec string
		command string
		want    string
	}{
		// The command text is appended verbatim so cmd builtins keep their
		// cmd meaning (dir stays cmd's dir, not PowerShell's Get-ChildItem
		// alias) and paths with spaces, quotes, or backslashes survive.
		{`C:\Windows\System32\cmd.exe`, `dir /b /ad`, `C:\Windows\System32\cmd.exe /d /c dir /b /ad`},
		{`C:\Windows\System32\cmd.exe`, `mklink /j "C:\a b" "C:\c d"`, `C:\Windows\System32\cmd.exe /d /c mklink /j "C:\a b" "C:\c d"`},
		{`C:\Windows\System32\cmd.exe`, `"C:\Program Files\vfox\vfox.exe" --version`, `C:\Windows\System32\cmd.exe /d /c "C:\Program Files\vfox\vfox.exe" --version`},
		{`C:\Windows\System32\cmd.exe`, `echo hello`, `C:\Windows\System32\cmd.exe /d /c echo hello`},
		{`C:\Windows\System32\cmd.exe`, ``, `C:\Windows\System32\cmd.exe /d /c `},
		// An interpreter path containing spaces is quoted.
		{`C:\My Tools\cmd.exe`, `echo hello`, `"C:\My Tools\cmd.exe" /d /c echo hello`},
	}
	for _, tc := range cases {
		if got := windowsCmdLine(tc.comSpec, tc.command); got != tc.want {
			t.Errorf("windowsCmdLine(%q, %q) = %q, want %q", tc.comSpec, tc.command, got, tc.want)
		}
	}
}

// decodePowerShell decodes a PowerShell -EncodedCommand blob back to the
// original script for assertions.
func decodePowerShell(t *testing.T, encoded string) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode failed: %v", err)
	}
	if len(raw)%2 != 0 {
		t.Fatalf("decoded length %d is not even (not UTF-16LE)", len(raw))
	}
	decoded := make([]uint16, len(raw)/2)
	for i := range decoded {
		decoded[i] = binary.LittleEndian.Uint16(raw[i*2:])
	}
	return string(utf16.Decode(decoded))
}

func TestPowerShellCommandUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}
	// On Unix PowerShellCommand falls back to /bin/sh -c so callers stay
	// testable; Windows-only call sites guard with IsWindows.
	cmd := PowerShellCommand(`echo hello`)
	if cmd.Path != "/bin/sh" {
		t.Errorf("PowerShellCommand path = %q, want /bin/sh", cmd.Path)
	}
}

func TestPowerShellArgsContainEncodedCommand(t *testing.T) {
	args := powershellArgs(`echo hello`)
	if len(args) != 5 {
		t.Fatalf("powershellArgs = %v, want 5 args", args)
	}
	if args[3] != "-EncodedCommand" {
		t.Errorf("powershellArgs[3] = %q, want -EncodedCommand", args[3])
	}
	if got := decodePowerShell(t, args[4]); got != "echo hello" {
		t.Errorf("decoded powershellArgs = %q, want %q", got, "echo hello")
	}
}
