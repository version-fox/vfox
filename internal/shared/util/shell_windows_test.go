//go:build windows

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
	"strings"
	"testing"
)

func TestShellCommandWindows(t *testing.T) {
	cmd := ShellCommand(`echo hello`)
	if len(cmd.Args) == 0 {
		t.Fatalf("ShellCommand args is empty")
	}
	// ShellCommand must stay on cmd.exe: plugin commands use cmd syntax
	// (e.g. `dir /b /ad`), which PowerShell would reinterpret.
	if got := strings.ToLower(cmd.Args[0]); !strings.HasSuffix(got, "cmd.exe") && got != "cmd" {
		t.Errorf("ShellCommand first arg = %q, want cmd.exe", cmd.Args[0])
	}
	if len(cmd.Args) != 4 || cmd.Args[1] != "/d" || cmd.Args[2] != "/c" || cmd.Args[3] != "echo hello" {
		t.Errorf("ShellCommand args = %v, want [cmd.exe /d /c 'echo hello']", cmd.Args)
	}
	if cmd.SysProcAttr == nil {
		t.Fatalf("ShellCommand SysProcAttr is nil, want raw CmdLine")
	}
	// The raw command line carries the command verbatim so spaces, quotes,
	// and backslashes survive Go's EscapeArg mangling.
	if !strings.HasSuffix(cmd.SysProcAttr.CmdLine, " /d /c echo hello") {
		t.Errorf("ShellCommand CmdLine = %q, want suffix %q", cmd.SysProcAttr.CmdLine, " /d /c echo hello")
	}
}

func TestShellCommandWindowsPathWithSpaces(t *testing.T) {
	// cmd syntax with spaces and quotes must be preserved verbatim in the
	// raw command line; this is what makes the command safe to pass through
	// Go's exec.Command → CreateProcess chain.
	script := `mklink /j "C:\Program Files\vfox\link" "C:\Program Files\vfox\target"`
	cmd := ShellCommand(script)
	if cmd.SysProcAttr == nil {
		t.Fatalf("ShellCommand SysProcAttr is nil, want raw CmdLine")
	}
	if !strings.HasSuffix(cmd.SysProcAttr.CmdLine, " /d /c "+script) {
		t.Errorf("ShellCommand CmdLine = %q, want suffix %q", cmd.SysProcAttr.CmdLine, " /d /c "+script)
	}
}
