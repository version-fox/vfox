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

package luai

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	lua "github.com/yuin/gopher-lua"

	"github.com/version-fox/vfox/internal/shared/util"
)

// installWindowsExecOverrides overrides Lua's os.execute and io.popen on
// Windows to use PowerShell -EncodedCommand. Go's exec.Command escapes
// arguments with syscall.EscapeArg, which doubles backslashes and turns "
// into \, but cmd.exe does not understand \" and mangles paths containing
// spaces. Routing through PowerShell -EncodedCommand (base64 of UTF-16LE)
// bypasses the cmd-line layer entirely: the encoded blob contains only
// [A-Za-z0-9+/=], so it is immune to EscapeArg.
//
// This override applies only in production. Development mode uses its own
// override in internal/plugin/development_output.go, which already routes
// through util.ShellCommand.
func installWindowsExecOverrides(L *lua.LState) {
	if runtime.GOOS != "windows" {
		return
	}

	// Override os.execute to use PowerShell -EncodedCommand on Windows.
	// The signature matches gopher-lua's os.execute: it returns
	// (exitcode, reason, signal) on failure or (exitcode) on success.
	osTable := L.GetGlobal("os")
	if osTable == nil {
		return
	}
	osLib, ok := osTable.(*lua.LTable)
	if !ok {
		return
	}

	osLib.RawSetString("execute", L.NewFunction(func(ls *lua.LState) int {
		s := ls.CheckString(1)
		cmd := util.ShellCommand(s)
		if err := cmd.Run(); err != nil {
			var ee *exec.ExitError
			if errors.As(err, &ee) {
				ls.Push(lua.LNumber(ee.ExitCode()))
				ls.Push(lua.LString(""))
				ls.Push(lua.LString(""))
				return 3
			}
			ls.Push(lua.LNumber(0))
			ls.Push(lua.LString(err.Error()))
			ls.Push(lua.LString(""))
			return 3
		}
		ls.Push(lua.LNumber(0))
		ls.Push(lua.LString(""))
		ls.Push(lua.LString(""))
		return 3
	}))

	// Override io.popen to use PowerShell -EncodedCommand on Windows.
	ioTable := L.GetGlobal("io")
	if ioTable == nil {
		return
	}
	ioLib, ok := ioTable.(*lua.LTable)
	if !ok {
		return
	}

	ioLib.RawSetString("popen", L.NewFunction(func(ls *lua.LState) int {
		cmdStr := ls.CheckString(1)
		mode := ls.OptString(2, "r")
		if mode != "r" && mode != "w" {
			ls.RaiseError("invalid mode %q to popen", mode)
			return 0
		}

		parent, child, err := os.Pipe()
		if err != nil {
			ls.RaiseError("failed to create pipe: %s", err)
			return 0
		}

		shellCmd := util.ShellCommand(cmdStr)
		if mode == "w" {
			shellCmd.Stdin = child
		} else {
			shellCmd.Stdout = child
			shellCmd.Stderr = child
		}

		if err := shellCmd.Start(); err != nil {
			parent.Close()
			child.Close()
			ls.RaiseError("failed to start: %s", err)
			return 0
		}
		child.Close()

		h := &popenHandle{f: parent, cmd: shellCmd, mode: mode}

		// Return a Lua table with read/write/close methods.
		f := ls.NewTable()
		f.RawSetString("read", ls.NewFunction(func(ls *lua.LState) int {
			format := ls.OptString(2, "*l")
			data, err := readPopen(h, format)
			if err != nil {
				ls.Push(lua.LNil)
				ls.Push(lua.LString(err.Error()))
				return 2
			}
			ls.Push(lua.LString(data))
			return 1
		}))
		f.RawSetString("write", ls.NewFunction(func(ls *lua.LState) int {
			if h.mode != "w" {
				ls.RaiseError("cannot write to a read-only popen handle")
				return 0
			}
			s := ls.OptString(2, "")
			if _, err := h.f.Write([]byte(s)); err != nil {
				ls.RaiseError("write error: %s", err)
				return 0
			}
			return 0
		}))
		f.RawSetString("close", ls.NewFunction(func(ls *lua.LState) int {
			h.f.Close()
			_ = h.cmd.Wait()
			return 0
		}))

		ls.Push(f)
		return 1
	}))
}

// popenHandle holds the pipe and command for an io.popen call.
type popenHandle struct {
	f    *os.File
	cmd  *exec.Cmd
	mode string
}

// readPopen reads from the popen pipe according to the given format.
// Supports "*a" (read all), "*l" (read line), and numeric byte counts.
func readPopen(h *popenHandle, format string) (string, error) {
	switch format {
	case "*a":
		data, err := io.ReadAll(h.f)
		if err != nil {
			return "", err
		}
		return string(data), nil
	case "*l":
		// Read until newline or EOF.
		buf := make([]byte, 4096)
		var out []byte
		for {
			n, err := h.f.Read(buf)
			if n > 0 {
				out = append(out, buf[:n]...)
				if idx := indexOf(out, '\n'); idx >= 0 {
					return string(out), nil
				}
			}
			if err != nil {
				if len(out) > 0 {
					return string(out), nil
				}
				if err == io.EOF {
					return "", nil
				}
				return "", err
			}
			if n == 0 {
				if len(out) > 0 {
					return string(out), nil
				}
				return "", nil
			}
		}
	default:
		// Try to parse as a number of bytes.
		n, err := strconv.Atoi(format)
		if err == nil && n >= 0 {
			buf := make([]byte, n)
			total := 0
			for total < n {
				r, e := h.f.Read(buf[total:])
				total += r
				if e != nil {
					break
				}
			}
			return string(buf[:total]), nil
		}
		return "", nil
	}
}

// indexOf returns the index of the first occurrence of b in s, or -1.
func indexOf(s []byte, b byte) int {
	for i, c := range s {
		if c == b {
			return i
		}
	}
	return -1
}
