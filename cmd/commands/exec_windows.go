//go:build windows

/*
 *    Copyright 2026 Han Li and contributors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package commands

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const unsafeBatchChars = "%!\"\r\n\x00"

func isWindowsBatchFile(path string) bool {
	ext := filepath.Ext(path)
	return strings.EqualFold(ext, ".cmd") || strings.EqualFold(ext, ".bat")
}

func validateWindowsBatchCommand(execPath string, args []string) error {
	if strings.ContainsAny(execPath, unsafeBatchChars) {
		return fmt.Errorf("refusing to run Windows batch file %q: path contains unsafe cmd.exe characters (%%, !, \", CR, LF, NUL)", execPath)
	}
	for _, arg := range args {
		if strings.ContainsAny(arg, unsafeBatchChars) {
			return fmt.Errorf("refusing to pass unsafe cmd.exe character in argument %q to Windows batch file %q (refused: %%, !, \", CR, LF, NUL)", arg, execPath)
		}
	}
	return nil
}

func quoteForceBatchArg(s string) string {
	backslashes := 0
	for i := len(s) - 1; i >= 0 && s[i] == '\\'; i-- {
		backslashes++
	}
	return `"` + s + strings.Repeat(`\`, backslashes) + `"`
}

func isSafeUnquotedBatchArg(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.ContainsRune("._:/\\+-", rune(c)):
		default:
			return false
		}
	}
	return true
}

func quoteBatchArg(s string) string {
	if isSafeUnquotedBatchArg(s) {
		return s
	}
	return quoteForceBatchArg(s)
}

// buildBatchCmdLine formats a CreateProcessW command line for a .cmd/.bat
// target. CreateProcessW invokes `cmd.exe /c <cmdline>`, which strips one outer
// quote pair from lines starting with `"`; wrapping the entire invocation in
// an outer quote pair preserves quoting around both execPath (argv[0]) and
// each quoted argument.
func buildBatchCmdLine(execPath string, args []string) string {
	var b strings.Builder
	b.WriteByte('"')
	b.WriteString(quoteForceBatchArg(execPath))
	for _, arg := range args {
		b.WriteByte(' ')
		b.WriteString(quoteBatchArg(arg))
	}
	b.WriteByte('"')
	return b.String()
}

func prepareExecCommand(execPath string, args []string) (*exec.Cmd, error) {
	if !isWindowsBatchFile(execPath) {
		return exec.Command(execPath, args...), nil
	}
	if err := validateWindowsBatchCommand(execPath, args); err != nil {
		return nil, err
	}
	cmd := exec.Command(execPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CmdLine: buildBatchCmdLine(execPath, args),
	}
	return cmd, nil
}
