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
	"os"
	"os/exec"
	"syscall"
)

// resolveWindowsCmd returns the cmd.exe path, honoring ComSpec with the same
// SystemRoot fallback previously used by development shell launches.
func resolveWindowsCmd() string {
	if comSpec := os.Getenv("ComSpec"); comSpec != "" {
		return comSpec
	}
	if root := os.Getenv("SystemRoot"); root != "" {
		return root + `\System32\cmd.exe`
	}
	return `C:\Windows\System32\cmd.exe`
}

// windowsShellCommand builds an *exec.Cmd that runs command via cmd.exe with
// an explicit raw command line. Go's exec.Command would otherwise escape the
// command argument with syscall.EscapeArg (doubling backslashes and turning
// " into \"), which cmd.exe misinterprets and which mangles paths containing
// spaces. Setting SysProcAttr.CmdLine passes windowsCmdLine verbatim to
// CreateProcess instead.
func windowsShellCommand(command string) *exec.Cmd {
	comSpec := resolveWindowsCmd()
	cmd := exec.Command(comSpec, "/d", "/c", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: windowsCmdLine(comSpec, command)}
	return cmd
}
