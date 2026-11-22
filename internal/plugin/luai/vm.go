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

package luai

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"

	"github.com/version-fox/vfox/internal/plugin/luai/module"
)

//go:embed fixtures/preload.lua
var preloadScript string

type LuaVM struct {
	Instance *lua.LState
}

func NewLuaVM() *LuaVM {
	instance := lua.NewState()

	return &LuaVM{
		Instance: instance,
	}
}

func (vm *LuaVM) Prepare(options *module.PreloadOptions) error {
	if err := vm.Instance.DoString(preloadScript); err != nil {
		return err
	}

	if options != nil {
		module.Preload(vm.Instance, options)
	}

	// On Windows, override os.execute and io.popen to use util.ShellCommand,
	// which invokes cmd.exe directly with a raw command line so cmd syntax
	// is preserved while paths with spaces, quotes, or backslashes survive.
	// See exec.go for details.
	installWindowsExecOverrides(vm.Instance)

	return nil
}

// LimitPackagePath limits where `require` looks for Lua modules to dirs,
// searched in order: module `a.b` is `<dir>/a/b.lua`.
//
// The search does not go through package.path. Lua reads `;` there as the
// separator between templates and `?` as the module name, and the plugin
// directories lie under HOME, so a HOME containing either character split the
// path or had the module name spliced into it. A loader in place of Lua's own
// file loader looks in each directory itself; preloaded modules come first as
// before.
func (vm *LuaVM) LimitPackagePath(dirs ...string) {
	packageModule := vm.Instance.GetGlobal("package").(*lua.LTable)
	packageModule.RawSetString("path", lua.LString(""))
	loaders, ok := packageModule.RawGetString("loaders").(*lua.LTable)
	if !ok {
		return
	}
	loaders.RawSetInt(2, vm.Instance.NewFunction(func(L *lua.LState) int {
		name := L.CheckString(1)
		file := strings.ReplaceAll(name, ".", string(os.PathSeparator)) + ".lua"
		tried := make([]string, 0, len(dirs))
		for _, dir := range dirs {
			path := filepath.Join(dir, file)
			if _, err := os.Stat(path); err != nil {
				tried = append(tried, "no file '"+path+"'")
				continue
			}
			fn, err := L.LoadFile(path)
			if err != nil {
				L.RaiseError("%s", err.Error())
			}
			L.Push(fn)
			return 1
		}
		L.Push(lua.LString("\n\t" + strings.Join(tried, "\n\t")))
		return 1
	}))
}

func (vm *LuaVM) ReturnedValue() *lua.LTable {
	table := vm.Instance.ToTable(-1) // returned value
	vm.Instance.Pop(1)               // remove received value
	return table
}

func (vm *LuaVM) CallFunction(pluginObj *lua.LTable, funcName string, _args ...lua.LValue) (lua.LValue, error) {
	function := pluginObj.RawGetString(funcName)
	fn, ok := function.(*lua.LFunction)
	if !ok {
		return nil, fmt.Errorf("[%s] must be a function, got %s", funcName, function.Type())
	}

	// In Lua, when a function is called with colon syntax (object:method()),
	// the object itself is implicitly passed as the first argument.
	// Here, pluginObj represents the Lua table instance of the plugin,
	// and it's being passed as the first argument to the Lua function to simulate this behavior.
	args := append([]lua.LValue{pluginObj}, _args...)

	if err := vm.Instance.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, args...); err != nil {
		return nil, err
	}

	value := vm.Instance.Get(-1)
	vm.Instance.Pop(1)
	return value, nil
}

func (vm *LuaVM) Close() {
	vm.Instance.Close()
}
