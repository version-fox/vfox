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
	"sync"

	"github.com/version-fox/vfox/internal/shared/util"
	lua "github.com/yuin/gopher-lua"
)

var (
	execStdioMu     sync.RWMutex
	execStdioWriter func() io.Writer
)

// RedirectStdIO overrides Lua's init-time io.stdout/io.stderr bindings for newly
// created plugins while non-nil and returns a restore function.
func RedirectStdIO(writer func() io.Writer) func() {
	execStdioMu.Lock()
	prev := execStdioWriter
	execStdioWriter = writer
	execStdioMu.Unlock()
	return func() {
		execStdioMu.Lock()
		execStdioWriter = prev
		execStdioMu.Unlock()
	}
}

func redirectLuaIO(L *lua.LState, outputFn func() io.Writer, raiseOnWriteError, interceptClose bool) {
	ioModule := L.GetGlobal("io").(*lua.LTable)
	stdout, stderr := ioModule.RawGetString("stdout"), ioModule.RawGetString("stderr")
	isStandard := func(value lua.LValue) bool { return value == stdout || value == stderr }
	write := func(L *lua.LState, start int) int {
		writer := outputFn()
		for i := start; i <= L.GetTop(); i++ {
			if _, err := io.WriteString(writer, L.CheckString(i)); err != nil {
				if raiseOnWriteError {
					L.RaiseError("write diagnostics: %s", err)
					return 0
				}
				L.Push(lua.LNil)
				L.Push(lua.LString(err.Error()))
				L.Push(lua.LNumber(1))
				return 3
			}
		}
		// Match gopher-lua's file:write/io.write result.
		L.Push(lua.LTrue)
		return 1
	}
	standardResult := func(L *lua.LState, name string, start int) int {
		switch name {
		case "write":
			return write(L, start)
		case "close":
			L.Push(lua.LNil)
			L.Push(lua.LString("cannot close standard output"))
			return 2
		default:
			L.Push(lua.LTrue)
			return 1
		}
	}
	methods := L.GetMetatable(stdout).(*lua.LTable)
	methodNames := []string{"write", "flush", "setvbuf"}
	ioNames := []string{"write", "flush"}
	if interceptClose {
		methodNames = append(methodNames, "close")
		ioNames = append(ioNames, "close")
	}
	for _, name := range methodNames {
		original := methods.RawGetString(name)
		methods.RawSetString(name, L.NewFunction(func(L *lua.LState) int {
			if isStandard(L.Get(1)) {
				return standardResult(L, name, 2)
			}
			return forwardLua(L, original, luaArguments(L)...)
		}))
	}
	originalOutput := ioModule.RawGetString("output")
	for _, name := range ioNames {
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
			if isStandard(file) {
				return standardResult(L, name, 1)
			}
			return forwardLua(L, original, luaArguments(L)...)
		}))
	}
}

func currentExecStdioWriter() func() io.Writer {
	execStdioMu.RLock()
	w := execStdioWriter
	execStdioMu.RUnlock()
	return w
}

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

// Redirect writes per VM, retaining normal Lua file handles and io.output.
// Neither the host's stdout nor its process environment is modified.
func configureDevelopmentOutput(L *lua.LState, output io.Writer, testing bool) {
	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		parts := make([]string, L.GetTop())
		for i := range parts {
			parts[i] = L.ToStringMeta(L.Get(i + 1)).String()
		}
		if _, err := fmt.Fprintln(output, strings.Join(parts, "\t")); err != nil {
			L.RaiseError("write diagnostics: %s", err)
		}
		return 0
	}))
	redirectLuaIO(L, func() io.Writer { return output }, true, true)
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
