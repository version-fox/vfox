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

package sdk

import (
	"fmt"
	"github.com/aooohan/version-fox/env"
	lua "github.com/yuin/gopher-lua"
	"net/url"
	"os"
)

type Source interface {
	DownloadUrl(handler *Handler, version Version) *url.URL
	Search(handler *Handler, version Version) []Version
	FileExt(handler *Handler) string
	EnvKeys(handler *Handler, version Version) []*env.KV
	Name() string
	Close()
}

type LuaSource struct {
	state *lua.LState
}

func (l LuaSource) checkValid() error {
	if l.state == nil {
		return fmt.Errorf("lua vm is nil")
	}
	if l.state.GetGlobal("search") == lua.LNil {
		return fmt.Errorf("search function not found")
	}
	if l.state.GetGlobal("download_url") == lua.LNil {
		return fmt.Errorf("download_url function not found")
	}
	if l.state.GetGlobal("file_ext") == lua.LNil {
		return fmt.Errorf("file_ext function not found")
	}
	if l.state.GetGlobal("env_keys") == lua.LNil {
		return fmt.Errorf("env_keys function not found")
	}
	if l.state.GetGlobal("name") == lua.LNil {
		return fmt.Errorf("name function not found")
	}
	return nil
}

func (l LuaSource) Close() {
	l.state.Close()
}

func (l LuaSource) Search(handler *Handler, version Version) []Version {
	L := l.state
	ctxTable := l.convert2LTable(L, handler, version)
	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("search"),
		NRet:    1,
		Protect: true,
	}, ctxTable); err != nil {
		panic(err)
	}

	table := L.ToTable(-1) // returned value
	L.Pop(1)               // remove received value

	var result []Version
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		rV, ok := value.(lua.LString)
		if !ok {
			panic("expected a string")
		}
		result = append(result, Version(rV.String()))
	})

	return result
}

func (l LuaSource) convert2LTable(L *lua.LState, handler *Handler, version Version) *lua.LTable {
	ctxTable := L.NewTable()
	L.SetField(ctxTable, "version_path", lua.LString(handler.VersionPath(version)))
	L.SetField(ctxTable, "os_type", lua.LString(handler.Operation.OsType))
	L.SetField(ctxTable, "arch_type", lua.LString(handler.Operation.ArchType))
	L.SetField(ctxTable, "version", lua.LString(version))
	return ctxTable
}

func (l LuaSource) DownloadUrl(handler *Handler, version Version) *url.URL {
	L := l.state
	ctxTable := l.convert2LTable(L, handler, version)

	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("download_url"),
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

func (l LuaSource) FileExt(handler *Handler) string {
	L := l.state
	ctxTable := l.convert2LTable(L, handler, "-")

	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("file_ext"),
		NRet:    1,
		Protect: true,
	}, ctxTable); err != nil {
		panic(err)
	}

	ret := L.Get(-1) // returned value
	L.Pop(1)         // remove received value
	return ret.String()
}

func (l LuaSource) EnvKeys(handler *Handler, version Version) []*env.KV {
	L := l.state
	ctxTable := l.convert2LTable(L, handler, version)
	if err := L.CallByParam(lua.P{
		Fn:      L.GetGlobal("env_keys"),
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

func (l LuaSource) Name() string {
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

func NewLuaSource(path string) *LuaSource {
	file, _ := os.ReadFile(path)
	L := lua.NewState()
	if err := L.DoString(string(file)); err != nil {
		fmt.Printf("Failed to load plugin: %s\nPlugin Path:%s\n", err.Error(), path)
		return nil
	}
	source := &LuaSource{
		state: L,
	}
	if err := source.checkValid(); err != nil {
		fmt.Printf("Plugin is invalid! err:%s \nPlugin Path: %s\n", err.Error(), path)
		return nil
	}
	return source
}
