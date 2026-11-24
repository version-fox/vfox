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
	"strings"

	"github.com/version-fox/vfox/internal/shared/util"
)

func buildCommandExec(execPath string, args []string) (*exec.Cmd, error) {
	if !isWindowsBatchFile(execPath) {
		return exec.Command(execPath, args...), nil
	}
	// % and ! still expand variables inside cmd.exe double quotes, while
	// quotes, CR, LF, and NUL can break out of quoting or truncate CmdLine.
	const unquotableBatchChars = "%!\"\r\n\x00"
	if strings.ContainsAny(execPath, unquotableBatchChars) {
		return nil, fmt.Errorf("refusing to pass cmd.exe metacharacters in Windows batch path %q", execPath)
	}
	for _, arg := range args {
		if strings.ContainsAny(arg, unquotableBatchChars) {
			return nil, fmt.Errorf("refusing to pass cmd.exe metacharacters in argument %q to Windows batch file %q", arg, execPath)
		}
	}
	// Wrap the inner quoted command line in one outer quote pair so cmd.exe /d /c
	// sees >=4 quotes, strips only the outer pair, and preserves quotes around both
	// execPath (e.g. C:\R&D\npm.cmd) and metacharacter args (^, &, |, <, >, ...).
	return util.ShellCommand(`"` + buildWindowsBatchCmdLine(execPath, args) + `"`), nil
}

func buildWindowsBatchCmdLine(execPath string, args []string) string {
	var b strings.Builder
	b.WriteString(quoteWindowsBatchArg(execPath, true))
	for _, arg := range args {
		b.WriteByte(' ')
		b.WriteString(quoteWindowsBatchArg(arg, arg == "" || strings.ContainsAny(arg, " \t&|<>()^,;=")))
	}
	return b.String()
}

func quoteWindowsBatchArg(arg string, forceQuote bool) string {
	if !forceQuote {
		return arg
	}
	// Trailing backslashes must be doubled before the closing quote so
	// CommandLineToArgvW does not read \" as an escaped literal quote.
	var b strings.Builder
	b.Grow(len(arg) + 4)
	b.WriteByte('"')
	b.WriteString(arg)
	for i := len(arg) - 1; i >= 0 && arg[i] == '\\'; i-- {
		b.WriteByte('\\')
	}
	b.WriteByte('"')
	return b.String()
}
