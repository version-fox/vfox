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
	"bufio"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"github.com/version-fox/vfox/internal/shared/util"
)

// installWindowsExecOverrides overrides Lua's os.execute and io.popen on
// Windows to use util.ShellCommand. ShellCommand invokes cmd.exe directly
// with an explicit raw command line, so existing plugin commands such as
// `dir /b /ad` keep their cmd meaning while paths containing spaces,
// quotes, or backslashes survive Go's EscapeArg mangling.
//
// This override applies only in production. Development mode uses its own
// override in internal/plugin/development_output.go, which already routes
// through util.ShellCommand.
func installWindowsExecOverrides(L *lua.LState) {
	if runtime.GOOS != "windows" {
		return
	}

	// Override os.execute to preserve gopher-lua's contract: a single return
	// value (0 on success, 1 on failure) with the child sharing the parent's
	// standard streams so installer diagnostics stay visible.
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
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			ls.Push(lua.LNumber(1))
			return 1
		}
		ls.Push(lua.LNumber(0))
		return 1
	}))

	// Override io.popen to preserve gopher-lua's contract while using the
	// same cmd-preserving transport.
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

		// os.Pipe returns (reader, writer). For mode "r" the child writes
		// and Lua reads; for mode "w" Lua writes and the child reads.
		reader, writer, err := os.Pipe()
		if err != nil {
			ls.RaiseError("failed to create pipe: %s", err)
			return 0
		}

		shellCmd := util.ShellCommand(cmdStr)
		var luaFile *os.File
		if mode == "r" {
			shellCmd.Stdout = writer
			shellCmd.Stdin = os.Stdin
			shellCmd.Stderr = os.Stderr
			if err := shellCmd.Start(); err != nil {
				reader.Close()
				writer.Close()
				ls.RaiseError("failed to start: %s", err)
				return 0
			}
			// Close the writer in the parent; the child keeps its copy.
			// Lua retains the reader.
			writer.Close()
			luaFile = reader
		} else {
			shellCmd.Stdin = reader
			shellCmd.Stdout = os.Stdout
			shellCmd.Stderr = os.Stderr
			if err := shellCmd.Start(); err != nil {
				reader.Close()
				writer.Close()
				ls.RaiseError("failed to start: %s", err)
				return 0
			}
			// Close the reader in the parent; the child keeps its copy.
			// Lua retains the writer.
			reader.Close()
			luaFile = writer
		}

		h := &popenHandle{f: luaFile, reader: bufio.NewReader(luaFile), cmd: shellCmd, mode: mode}

		// Return a Lua table with read/write/close methods.
		f := ls.NewTable()
		f.RawSetString("read", ls.NewFunction(func(ls *lua.LState) int {
			if h.mode != "r" {
				ls.RaiseError("cannot read from a write-only popen handle")
				return 0
			}
			format := ls.OptString(2, "*l")
			value, err := readPopen(h, format)
			if err != nil {
				ls.Push(lua.LNil)
				ls.Push(lua.LString(err.Error()))
				return 2
			}
			ls.Push(value)
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
			ls.Push(lua.LTrue)
			return 1
		}))
		f.RawSetString("close", ls.NewFunction(func(ls *lua.LState) int {
			h.f.Close()
			err := h.cmd.Wait()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					ls.Push(lua.LNumber(ee.ExitCode()))
					return 1
				}
				ls.RaiseError("close error: %s", err)
				return 0
			}
			ls.Push(lua.LNumber(0))
			return 1
		}))

		ls.Push(f)
		return 1
	}))
}

// popenHandle holds the pipe and command for an io.popen call.
type popenHandle struct {
	f      *os.File
	reader *bufio.Reader
	cmd    *exec.Cmd
	mode   string
}

// readPopen reads from the popen pipe according to the given format.
// Supports "*a" (read all), "*l" (read line without the trailing newline),
// and numeric byte counts. On EOF with no data it returns LNil so Lua loops
// like `while line do` terminate, matching gopher-lua semantics.
func readPopen(h *popenHandle, format string) (lua.LValue, error) {
	switch format {
	case "*a":
		data, err := io.ReadAll(h.reader)
		if err != nil {
			return nil, err
		}
		return lua.LString(string(data)), nil
	case "*l":
		line, err := h.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if len(line) == 0 {
					return lua.LNil, nil
				}
				return lua.LString(strings.TrimRight(line, "\r\n")), nil
			}
			return nil, err
		}
		return lua.LString(strings.TrimRight(line, "\r\n")), nil
	default:
		return readPopenBytes(h, format)
	}
}

// readPopenBytes handles numeric byte-count formats. A leading "*" (such as
// "*5" accepted by some callers) is stripped. On EOF with no data it returns
// LNil; partial data is returned as a string.
func readPopenBytes(h *popenHandle, format string) (lua.LValue, error) {
	countStr := strings.TrimPrefix(format, "*")
	n, err := parseByteCount(countStr)
	if err != nil {
		// Match gopher-lua: unknown formats yield an empty result.
		return lua.LString(""), nil
	}
	if n == 0 {
		return lua.LString(""), nil
	}
	buf := make([]byte, n)
	total := 0
	for total < n {
		r, e := h.reader.Read(buf[total:])
		total += r
		if e != nil {
			if e == io.EOF {
				break
			}
			return nil, e
		}
		if r == 0 {
			break
		}
	}
	if total == 0 {
		return lua.LNil, nil
	}
	return lua.LString(string(buf[:total])), nil
}

// parseByteCount parses a non-negative decimal byte count.
func parseByteCount(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, io.ErrUnexpectedEOF
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, io.ErrUnexpectedEOF
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
