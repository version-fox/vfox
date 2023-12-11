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
	"github.com/aooohan/version-fox/plugin/module"
	"github.com/aooohan/version-fox/util"
	lua "github.com/yuin/gopher-lua"
	"net/url"
	"os"
)

const (
	LuaPluginObjKey = "PLUGIN"
	OsType          = "OS_TYPE"
	ArchType        = "ARCH_TYPE"
)

type LuaPlugin struct {
	state     *lua.LState
	pluginObj *lua.LTable
	path      string
	Name      string
	Author    string
	Version   string
	UpdateUrl string
}

func (l *LuaPlugin) checkValid() error {
	if l.state == nil {
		return fmt.Errorf("lua vm is nil")
	}
	obj := l.pluginObj
	if obj.RawGetString("Search") == lua.LNil {
		return fmt.Errorf("search function not found")
	}
	if obj.RawGetString("DownloadUrl") == lua.LNil {
		return fmt.Errorf("download_url function not found")
	}
	if obj.RawGetString("EnvKeys") == lua.LNil {
		return fmt.Errorf("env_keys function not found")
	}
	return nil
}

func (l *LuaPlugin) Close() {
	l.state.Close()
}

func (l *LuaPlugin) Search(version string) []string {
	L := l.state
	ctxTable := L.NewTable()
	L.SetField(ctxTable, "version", lua.LString(version))
	if err := L.CallByParam(lua.P{
		Fn:      l.pluginObj.RawGetString("Search").(*lua.LFunction),
		NRet:    1,
		Protect: true,
	}, ctxTable); err != nil {
		panic(err)
	}

	table := L.ToTable(-1) // returned value
	L.Pop(1)               // remove received value

	var result []string
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		rV, ok := value.(lua.LString)
		if !ok {
			panic("expected a string")
		}
		result = append(result, rV.String())
	})

	return result
}

func (l *LuaPlugin) DownloadUrl(version string) *url.URL {
	L := l.state
	ctxTable := L.NewTable()
	L.SetField(ctxTable, "version", lua.LString(version))

	if err := L.CallByParam(lua.P{
		Fn:      l.pluginObj.RawGetString("DownloadUrl").(*lua.LFunction),
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

func (l *LuaPlugin) EnvKeys(version, versionPath string) []*env.KV {
	L := l.state
	ctxTable := L.NewTable()
	L.SetField(ctxTable, "version", lua.LString(version))
	L.SetField(ctxTable, "version_path", lua.LString(versionPath))
	if err := L.CallByParam(lua.P{
		Fn:      l.pluginObj.RawGetString("EnvKeys"),
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

func (l *LuaPlugin) luaPrint() int {
	L := l.state
	L.SetGlobal("print", L.NewFunction(func(L *lua.LState) int {
		top := L.GetTop()
		for i := 1; i <= top; i++ {
			fmt.Print(L.ToStringMeta(L.Get(i)))
			if i != top {
				fmt.Print("\t")
			}
		}
		fmt.Println()
		return 0
	}))
	return 0
}

func (l *LuaPlugin) Label(version string) string {
	return fmt.Sprintf("%s@%s", l.Name, version)
}

func NewLuaSource(path string, osType util.OSType, archType util.ArchType) *LuaPlugin {
	file, _ := os.ReadFile(path)
	// TODO: use filename as the plugin Name
	L := lua.NewState()
	module.Preload(L)
	// set OS_TYPE and ARCH_TYPE

	L.SetGlobal(OsType, lua.LString(osType))
	L.SetGlobal(ArchType, lua.LString(archType))

	if err := L.DoString(string(file)); err != nil {
		fmt.Printf("Failed to load plugin: %s\nPlugin Path:%s\n", err.Error(), path)
		return nil
	}
	pluginOjb := L.GetGlobal(LuaPluginObjKey)
	if pluginOjb.Type() == lua.LTNil {
		fmt.Printf("Plugin is invalid! err:%s \nPlugin Path: %s\n", "plugin object not found", path)
		return nil
	}

	PLUGIN := pluginOjb.(*lua.LTable)

	source := &LuaPlugin{
		state:     L,
		pluginObj: PLUGIN,
	}

	if err := source.checkValid(); err != nil {
		fmt.Printf("Plugin is invalid! err:%s \nPlugin Path: %s\n", err.Error(), path)
		return nil
	}

	if name := PLUGIN.RawGetString("name"); name.Type() != lua.LTNil {
		source.Name = name.String()
	}
	if version := PLUGIN.RawGetString("version"); version.Type() != lua.LTNil {
		source.Version = version.String()
	}
	if updateUrl := PLUGIN.RawGetString("updateUrl"); updateUrl.Type() != lua.LTNil {
		source.UpdateUrl = updateUrl.String()
	}
	if author := PLUGIN.RawGetString("author"); author.Type() != lua.LTNil {
		source.Author = author.String()
	}
	return source
}
