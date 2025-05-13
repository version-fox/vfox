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

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/luai/module"
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

type PrepareOptions struct {
	Config *config.Config
}

func (vm *LuaVM) Prepare(options *PrepareOptions) error {
	if err := vm.Instance.DoString(preloadScript); err != nil {
		return err
	}

	if options != nil {
		module.Preload(vm.Instance, options.Config)
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

func (vm *LuaVM) CallFunction(function lua.LValue, args ...lua.LValue) (*lua.LTable, error) {
	if err := vm.Instance.CallByParam(lua.P{
		Fn:      function.(*lua.LFunction),
		NRet:    1,
		Protect: true,
	}, args...); err != nil {
		return nil, err
	}

	return vm.ReturnedValue(), nil
}

func (vm *LuaVM) GetTableString(table *lua.LTable, key string) string {
	if value := table.RawGetString(key); value.Type() != lua.LTNil {
		return value.String()
	}
	return ""
}

func (vm *LuaVM) Close() {
	vm.Instance.Close()
}
