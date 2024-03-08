package luai

import (
	_ "embed"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/module"
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
	vm.Instance.DoString(preloadScript)
	module.Preload(vm.Instance, options.Config)

	return nil
}

func (vm *LuaVM) ReturnedValue() *lua.LTable {
	table := vm.Instance.ToTable(-1) // returned value
	vm.Instance.Pop(1)               // remove received value
	return table
}

func (vm *LuaVM) CallFunction(function lua.LValue, args ...lua.LValue) error {
	logger.Debugf("CallFunction: %s", function.String())

	if err := vm.Instance.CallByParam(lua.P{
		Fn:      function.(*lua.LFunction),
		NRet:    1,
		Protect: true,
	}, args...); err != nil {
		return err
	}
	return nil
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
