/*
 *    Copyright 2023 [lihan aooohan@gmail.com]
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
	"github.com/aooohan/version-fox/env"
	"github.com/aooohan/version-fox/lua_module"
	lua "github.com/yuin/gopher-lua"
	"net/url"
	"os"
)

const (
	LuaPluginObjKey = "PLUGIN"
)

type LuaPlugin struct {
	state     *lua.LState
	pluginObj lua.LValue
	name      string
}

func (l *LuaPlugin) checkValid() error {
	if l.state == nil {
		return fmt.Errorf("lua_module vm is nil")
	}
	if l.state.GetGlobal("search") == lua.LNil {
		return fmt.Errorf("search function not found")
	}
	if l.state.GetGlobal("download_url") == lua.LNil {
		return fmt.Errorf("download_url function not found")
	}
	if l.state.GetGlobal("env_keys") == lua.LNil {
		return fmt.Errorf("env_keys function not found")
	}
	if l.state.GetGlobal("name") == lua.LNil {
		return fmt.Errorf("name function not found")
	}
	return nil
}

func (l *LuaPlugin) Close() {
	l.state.Close()
}

func (l *LuaPlugin) Search(ctx *Context) []SearchResult {
	L := l.state
	ctxTable := l.convert2LTable(L, ctx)
	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("Search"),
		NRet:    1,
		Protect: true,
	}, ctxTable); err != nil {
		panic(err)
	}

	table := L.ToTable(-1) // returned value
	L.Pop(1)               // remove received value

	var result []SearchResult
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		rV, ok := value.(lua.LString)
		if !ok {
			panic("expected a string")
		}
		result = append(result, SearchResult(rV.String()))
	})

	return result
}

func (l *LuaPlugin) convert2LTable(L *lua.LState, ctx *Context) *lua.LTable {
	//handler := ctx.Handler
	//version := ctx.Version
	ctxTable := L.NewTable()
	//L.SetField(ctxTable, "version_path", lua.LString(handler.VersionPath(version)))
	//L.SetField(ctxTable, "os_type", lua.LString(handler.sdkManager.osType))
	//L.SetField(ctxTable, "arch_type", lua.LString(handler.sdkManager.osType))
	//L.SetField(ctxTable, "version", lua.LString(version))
	return ctxTable
}

func (l *LuaPlugin) DownloadUrl(ctx *Context) *url.URL {
	L := l.state
	ctxTable := l.convert2LTable(L, ctx)

	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("DownloadUrl"),
		NRet:    1,
		Protect: true,
	}, ctxTable); err != nil {
		panic(err)
	}

	ret := L.Get(-1) // returned value
	L.Pop(1)         // remove received value

	u, _ := url.Parse(ret.String())
	return u
}

func (l *LuaPlugin) EnvKeys(ctx *Context) []*env.KV {
	L := l.state
	ctxTable := l.convert2LTable(L, ctx)
	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("EnvKeys"),
		NRet:    1,
		Protect: true,
	}, ctxTable); err != nil {
		panic(err)
	}

	table := L.ToTable(-1) // returned value
	L.Pop(1)               // remove received value

	var envKeys []*env.KV
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		kvTable, ok := value.(*lua.LTable)
		if !ok {
			panic("expected a table")
		}
		key = kvTable.RawGetString("key")
		value = kvTable.RawGetString("value")
		envKeys = append(envKeys, &env.KV{Key: key.String(), Value: value.String()})
	})

	return envKeys
}

func (l *LuaPlugin) Name() string {
	L := l.state

	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("name"),
		NRet:    1,
		Protect: true,
	}); err != nil {
		panic(err)
	}

	ret := L.Get(-1) // returned value
	L.Pop(1)         // remove received value
	return ret.String()
}

func NewLuaSource(path string) *LuaPlugin {
	file, _ := os.ReadFile(path)
	// TODO: use filename as the plugin name
	L := lua.NewState()
	lua_module.Preload(L)
	if err := L.DoString(string(file)); err != nil {
		fmt.Printf("Failed to load plugin: %s\nPlugin Path:%s\n", err.Error(), path)
		return nil
	}
	pluginOjb := L.GetGlobal(LuaPluginObjKey)
	if pluginOjb.Type() == lua.LTNil {
		fmt.Printf("Plugin is invalid! err:%s \nPlugin Path: %s\n", "plugin object not found", path)
		return nil
	}
	source := &LuaPlugin{
		state:     L,
		pluginObj: pluginOjb,
	}
	if err := source.checkValid(); err != nil {
		fmt.Printf("Plugin is invalid! err:%s \nPlugin Path: %s\n", err.Error(), path)
		return nil
	}
	return source
}
