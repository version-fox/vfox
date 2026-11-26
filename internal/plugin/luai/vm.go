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
// package.path is set to the templates for dirs (`<dir>/?.lua`, joined with
// `;`), as before, so plugin code that reads it, for example to find its own
// directory, keeps working. But Lua reads `;` there as the separator between
// templates and `?` as the module name, and the plugin directories lie under
// HOME, so a HOME containing either character split the path or had the
// module name spliced into it. So a loader in place of Lua's own file loader
// searches dirs themselves at the position of vfox's templates in the current
// package.path, and the templates a plugin added before or after them with
// Lua's `?` rule. If the plugin replaced package.path, dirs come first.
// Preloaded modules come first as before.
func (vm *LuaVM) LimitPackagePath(dirs ...string) {
	templates := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		templates = append(templates, filepath.Join(dir, "?.lua"))
	}
	own := strings.Join(templates, ";")

	packageModule := vm.Instance.GetGlobal("package").(*lua.LTable)
	packageModule.RawSetString("path", lua.LString(own))
	// gopher-lua's OpenPackage always creates package.loaders, so this
	// assertion only fails if that changes, and then loudly.
	loaders := packageModule.RawGetString("loaders").(*lua.LTable)
	loaders.RawSetInt(2, vm.Instance.NewFunction(func(L *lua.LState) int {
		modulePath := strings.ReplaceAll(L.CheckString(1), ".", string(os.PathSeparator))
		fromTemplates := func(path string) []string {
			var candidates []string
			for _, template := range strings.Split(path, ";") {
				if template != "" {
					candidates = append(candidates, strings.ReplaceAll(template, "?", modulePath))
				}
			}
			return candidates
		}
		fromDirs := make([]string, 0, len(dirs))
		for _, dir := range dirs {
			fromDirs = append(fromDirs, filepath.Join(dir, modulePath+".lua"))
		}

		var candidates []string
		path := lua.LVAsString(packageModule.RawGetString("path"))
		if before, after, found := strings.Cut(path, own); found {
			candidates = append(fromTemplates(before), fromDirs...)
			candidates = append(candidates, fromTemplates(after)...)
		} else {
			candidates = append(fromDirs, fromTemplates(path)...)
		}

		tried := make([]string, 0, len(candidates))
		for _, path := range candidates {
			if _, err := os.Stat(path); err != nil {
				if os.IsNotExist(err) {
					tried = append(tried, "no file '"+path+"'")
				} else {
					// Keep the reason, for example "permission denied", rather
					// than "no file", which would send the reader looking for a
					// file that may be there. As with Lua's own loader, the
					// search goes on to the next candidate.
					tried = append(tried, err.Error())
				}
				continue
			}
			// As with Lua's own loader, a directory with the module's file
			// name is an error here, not skipped.
			fn, err := L.LoadFile(path)
			if err != nil {
				L.RaiseError("%s", err.Error())
			}
			L.Push(fn)
			return 1
		}
		// gopher-lua puts "\n\t" in front of each loader's message itself.
		L.Push(lua.LString(strings.Join(tried, "\n\t")))
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
