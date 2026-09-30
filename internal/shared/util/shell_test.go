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

func TestShellCommandWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only test")
	}
	cmd := ShellCommand(`echo hello`)
	if cmd.Path == "" && strings.ToLower(cmd.Args[0]) != "powershell.exe" && cmd.Args[0] != "powershell" {
		t.Errorf("ShellCommand first arg = %q, want powershell", cmd.Args[0])
	}
	if len(cmd.Args) < 6 {
		t.Fatalf("ShellCommand args = %v, want at least 6 (powershell -NoProfile -NoLogo -NonInteractive -EncodedCommand <base64>)", cmd.Args)
	}
	encIdx := -1
	for i, a := range cmd.Args {
		if a == "-EncodedCommand" {
			encIdx = i
			break
		}
	}
	if encIdx == -1 {
		t.Fatalf("ShellCommand args = %v, want -EncodedCommand flag", cmd.Args)
	}
	encoded := cmd.Args[encIdx+1]
	for _, bad := range []string{" ", "\"", "\\", "'", "`"} {
		if strings.Contains(encoded, bad) {
			t.Errorf("encoded command contains %q: %q", bad, encoded)
		}
	}
}

func TestShellCommandWindowsPathWithSpaces(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only test")
	}
	script := `& 'C:\Program Files\vfox\upgrade.bat'`
	cmd := ShellCommand(script)
	// The encoded argument must contain no spaces; this is what makes the
	// command safe to pass through Go's exec.Command → CreateProcess chain.
	encIdx := -1
	for i, a := range cmd.Args {
		if a == "-EncodedCommand" {
			encIdx = i
			break
		}
	}
	if encIdx == -1 {
		t.Fatalf("no -EncodedCommand flag in args: %v", cmd.Args)
	}
	if strings.Contains(cmd.Args[encIdx+1], " ") {
		t.Errorf("encoded command has spaces: %q", cmd.Args[encIdx+1])
	}
}
