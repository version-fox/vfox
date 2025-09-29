/*
 *    Copyright 2025 Han Li and contributors
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
	"strings"

	"github.com/version-fox/vfox/internal/plugin/luai/module"
	lua "github.com/yuin/gopher-lua"
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

	return nil
}

// LimitPackagePath limits the package path of the Lua VM.
func (vm *LuaVM) LimitPackagePath(packagePaths ...string) {
	packageModule := vm.Instance.GetGlobal("package").(*lua.LTable)
	packageModule.RawSetString("path", lua.LString(strings.Join(packagePaths, ";")))
}

func (vm *LuaVM) ReturnedValue() *lua.LTable {
	table := vm.Instance.ToTable(-1) // returned value
	vm.Instance.Pop(1)               // remove received value
	return table
}

func (vm *LuaVM) CallFunction(pluginObj *lua.LTable, funcName string, _args ...lua.LValue) (*lua.LTable, error) {
	function := pluginObj.RawGetString(funcName)

	// In Lua, when a function is called with colon syntax (object:method()),
	// the object itself is implicitly passed as the first argument.
	// Here, pluginObj represents the Lua table instance of the plugin,
	// and it's being passed as the first argument to the Lua function to simulate this behavior.
	args := append([]lua.LValue{pluginObj}, _args...)

	if err := vm.Instance.CallByParam(lua.P{
		Fn:      function.(*lua.LFunction),
		NRet:    1,
		Protect: true,
	}, args...); err != nil {
		return nil, err
	}

	return vm.ReturnedValue(), nil
}

func (vm *LuaVM) Close() {
	vm.Instance.Close()
}
