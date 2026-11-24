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

package plugin

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/version-fox/vfox/internal/shared/util"
	lua "github.com/yuin/gopher-lua"
)

// forwardLua preserves all results when wrapping a standard library function.
func forwardLua(L *lua.LState, fn lua.LValue, args ...lua.LValue) int {
	top := L.GetTop()
	if err := L.CallByParam(lua.P{Fn: fn, NRet: lua.MultRet, Protect: true}, args...); err != nil {
		L.RaiseError("%s", err)
	}
	return L.GetTop() - top
}

func luaArguments(L *lua.LState) []lua.LValue {
	args := make([]lua.LValue, L.GetTop())
	for i := range args {
		args[i] = L.Get(i + 1)
	}
	return args
}

// redirectLuaOutput routes Lua print and standard stream writes per VM through
// caller-supplied writers so gopher-lua's startup stdout/stderr handles cannot go stale.
// When raiseOnWriteError is true (development mode), write errors abort the hook;
// otherwise production plugins preserve gopher-lua's write error contract (print drops
// write errors, and io.write/file:write return nil, message, 1) while intentionally
// refusing io.stdin:close()/io.stdout:close()/io.stderr:close() so plugins cannot close host stdio.
func redirectLuaOutput(L *lua.LState, stdoutWriter, stderrWriter func() io.Writer, raiseOnWriteError bool) {
	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		parts := make([]string, L.GetTop())
		for i := range parts {
			parts[i] = L.ToStringMeta(L.Get(i + 1)).String()
		}
		writer := stdoutWriter()
		if writer == nil {
			if raiseOnWriteError {
				L.RaiseError("write output: %s", os.ErrInvalid)
			}
			return 0
		}
		if _, err := fmt.Fprintln(writer, strings.Join(parts, "\t")); err != nil && raiseOnWriteError {
			L.RaiseError("write output: %s", err)
		}
		return 0
	}))
	ioModule := L.GetGlobal("io").(*lua.LTable)
	stdin, stdout, stderr := ioModule.RawGetString("stdin"), ioModule.RawGetString("stdout"), ioModule.RawGetString("stderr")
	writerFor := func(value lua.LValue) (func() io.Writer, string, bool) {
		switch value {
		case stdout:
			return stdoutWriter, "standard output", true
		case stderr:
			return stderrWriter, "standard error", true
		default:
			return nil, "", false
		}
	}
	writeError := func(L *lua.LState, err error) int {
		if raiseOnWriteError {
			L.RaiseError("write output: %s", err)
			return 0
		}
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		L.Push(lua.LNumber(1))
		return 3
	}
	write := func(L *lua.LState, writer io.Writer, start int) int {
		if writer == nil {
			return writeError(L, os.ErrInvalid)
		}
		for i := start; i <= L.GetTop(); i++ {
			if _, err := io.WriteString(writer, L.CheckString(i)); err != nil {
				return writeError(L, err)
			}
		}
		// Match gopher-lua's file:write/io.write result.
		L.Push(lua.LTrue)
		return 1
	}
	refuseClose := func(L *lua.LState, streamName string) int {
		L.Push(lua.LNil)
		L.Push(lua.LString("cannot close " + streamName))
		return 2
	}
	standardResult := func(L *lua.LState, name, streamName string, start int, writer io.Writer) int {
		switch name {
		case "write":
			return write(L, writer, start)
		case "close":
			return refuseClose(L, streamName)
		default:
			L.Push(lua.LTrue)
			return 1
		}
	}
	methods := L.GetMetatable(stdout).(*lua.LTable)
	for _, name := range []string{"write", "flush", "close", "setvbuf"} {
		original := methods.RawGetString(name)
		methods.RawSetString(name, L.NewFunction(func(L *lua.LState) int {
			file := L.Get(1)
			if name == "close" && file == stdin {
				return refuseClose(L, "standard input")
			}
			if writer, streamName, ok := writerFor(file); ok {
				return standardResult(L, name, streamName, 2, writer())
			}
			return forwardLua(L, original, luaArguments(L)...)
		}))
	}
	originalOutput := ioModule.RawGetString("output")
	for _, name := range []string{"write", "flush", "close"} {
		original := ioModule.RawGetString(name)
		ioModule.RawSetString(name, L.NewFunction(func(L *lua.LState) int {
			file := L.Get(1)
			if name != "close" || L.GetTop() == 0 {
				if err := L.CallByParam(lua.P{Fn: originalOutput, NRet: 1, Protect: true}); err != nil {
					L.RaiseError("%s", err)
				}
				file = L.Get(-1)
				L.Pop(1)
			}
			if name == "close" && file == stdin {
				return refuseClose(L, "standard input")
			}
			if writer, streamName, ok := writerFor(file); ok {
				return standardResult(L, name, streamName, 1, writer())
			}
			return forwardLua(L, original, luaArguments(L)...)
		}))
	}
}

// Redirect writes per VM, retaining normal Lua file handles and io.output.
// Neither the host's stdout nor its process environment is modified.
func configureDevelopmentOutput(L *lua.LState, output io.Writer, testing bool) {
	writer := func() io.Writer { return output }
	redirectLuaOutput(L, writer, writer, true)
	ioModule := L.GetGlobal("io").(*lua.LTable)
	osModule := L.GetGlobal("os").(*lua.LTable)
	osModule.RawSetString("exit", L.NewFunction(func(L *lua.LState) int {
		L.RaiseError("os.exit(%d) called", L.OptInt(1, 0))
		return 0
	}))
	if testing {
		for _, entry := range []struct {
			table       *lua.LTable
			name, label string
		}{
			{osModule, "execute", "os.execute"}, {ioModule, "popen", "io.popen"},
		} {
			entry.table.RawSetString(entry.name, L.NewFunction(func(L *lua.LState) int {
				L.RaiseError("%s is disabled in plugin tests; replace it with a Lua stub or use disposable CI", entry.label)
				return 0
			}))
		}
	} else {
		osModule.RawSetString("execute", L.NewFunction(func(L *lua.LState) int {
			// util.ShellCommand invokes cmd.exe directly with a raw command
			// line on Windows so paths with spaces, quotes, or backslashes
			// survive Go's EscapeArg mangling. On Unix it falls back to
			// /bin/sh -c.
			command := util.ShellCommand(L.CheckString(1))
			// As with normal Lua os.execute, arbitrary child processes are not
			// covered by the Lua/HTTP deadline. Their output is diagnostic text.
			command.Stdin, command.Stdout, command.Stderr = os.Stdin, output, output
			if err := command.Run(); err != nil {
				L.Push(lua.LNumber(1))
			} else {
				L.Push(lua.LNumber(0))
			}
			return 1
		}))
	}
}
