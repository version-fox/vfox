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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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
	ioModule := L.GetGlobal("io").(*lua.LTable)
	stdout, stderr := ioModule.RawGetString("stdout"), ioModule.RawGetString("stderr")
	isStandard := func(value lua.LValue) bool { return value == stdout || value == stderr }
	write := func(L *lua.LState, start int) int {
		for i := start; i <= L.GetTop(); i++ {
			if _, err := io.WriteString(output, L.CheckString(i)); err != nil {
				L.RaiseError("write diagnostics: %s", err)
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
	for _, name := range []string{"write", "flush", "close", "setvbuf"} {
		original := methods.RawGetString(name)
		methods.RawSetString(name, L.NewFunction(func(L *lua.LState) int {
			if isStandard(L.Get(1)) {
				return standardResult(L, name, 2)
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
			if isStandard(file) {
				return standardResult(L, name, 1)
			}
			return forwardLua(L, original, luaArguments(L)...)
		}))
	}
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
			program, args := "/bin/sh", []string{"-c", L.CheckString(1)}
			if runtime.GOOS == "windows" {
				root := os.Getenv("SystemRoot")
				if root == "" {
					root = `C:\Windows`
				}
				program, args = filepath.Join(root, "System32", "cmd.exe"), []string{"/c", L.CheckString(1)}
			}
			// As with normal Lua os.execute, arbitrary child processes are not
			// covered by the Lua/HTTP deadline. Their output is diagnostic text.
			command := exec.Command(program, args...)
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
