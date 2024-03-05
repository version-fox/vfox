/*
 *    Copyright 2024 Han Li and contributors
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

package internal

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/module"
	lua "github.com/yuin/gopher-lua"
)

//go:embed fixtures/preload.lua
var preloadScript string

const (
	LuaPluginObjKey = "PLUGIN"
	OsType          = "OS_TYPE"
	ArchType        = "ARCH_TYPE"
)

type LuaPlugin struct {
	state     *lua.LState
	pluginObj *lua.LTable
	// plugin source path
	Filepath string
	// plugin filename, this is also alias name, sdk-name
	Filename string
	// The name defined inside the plugin
	Name              string
	Author            string
	Version           string
	Description       string
	UpdateUrl         string
	MinRuntimeVersion string
}

func (l *LuaPlugin) checkValid() error {
	if l.state == nil {
		return fmt.Errorf("lua vm is nil")
	}
	obj := l.pluginObj
	if obj.RawGetString("Available") == lua.LNil {
		return fmt.Errorf("[Available] function not found")
	}
	if obj.RawGetString("PreInstall") == lua.LNil {
		return fmt.Errorf("[PreInstall] function not found")
	}
	if obj.RawGetString("EnvKeys") == lua.LNil {
		return fmt.Errorf("[EnvKeys] function not found")
	}
	return nil
}

func (l *LuaPlugin) Close() {
	l.state.Close()
}

func (l *LuaPlugin) Available() ([]*Package, error) {
	L := l.state
	ctxTable := L.NewTable()
	L.SetField(ctxTable, "runtimeVersion", lua.LString(RuntimeVersion))
	if err := L.CallByParam(lua.P{
		Fn:      l.pluginObj.RawGetString("Available").(*lua.LFunction),
		NRet:    1,
		Protect: true,
	}, l.pluginObj, ctxTable); err != nil {
		return nil, err
	}

	table := l.returnedValue()

	if table == nil || table.Type() == lua.LTNil {
		return []*Package{}, nil
	}
	var err error
	var result []*Package
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		kvTable, ok := value.(*lua.LTable)
		if !ok {
			err = fmt.Errorf("the return value is not a table")
			return
		}
		mainSdk, err := l.parseInfo(kvTable)
		if err != nil {
			return
		}
		mainSdk.Name = l.Name
		var additionalArr []*Info
		additional := kvTable.RawGetString("addition")
		if tb, ok := additional.(*lua.LTable); ok && tb.Len() != 0 {
			additional.(*lua.LTable).ForEach(func(key lua.LValue, value lua.LValue) {
				itemTable, ok := value.(*lua.LTable)
				if !ok {
					err = fmt.Errorf("the return value is not a table")
					return
				}
				item, err := l.parseInfo(itemTable)
				if err != nil {
					return
				}
				if item.Name == "" {
					err = fmt.Errorf("additional file no name provided")
					return
				}
				additionalArr = append(additionalArr, item)
			})
		}

		result = append(result, &Package{
			Main:      mainSdk,
			Additions: additionalArr,
		})

	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (l *LuaPlugin) Checksum(table *lua.LTable) *Checksum {
	checksum := &Checksum{}
	sha256 := table.RawGetString("sha256")
	md5 := table.RawGetString("md5")
	sha512 := table.RawGetString("sha512")
	sha1 := table.RawGetString("sha1")
	if sha256.Type() != lua.LTNil {
		checksum.Value = sha256.String()
		checksum.Type = "sha256"
	} else if md5.Type() != lua.LTNil {
		checksum.Value = md5.String()
		checksum.Type = "md5"
	} else if sha1.Type() != lua.LTNil {
		checksum.Value = sha1.String()
		checksum.Type = "sha1"
	} else if sha512.Type() != lua.LTNil {
		checksum.Value = sha512.String()
		checksum.Type = "sha512"
	} else {
		return NoneChecksum
	}
	return checksum
}

func (l *LuaPlugin) PreInstall(version Version) (*Package, error) {
	L := l.state
	ctxTable := L.NewTable()
	L.SetField(ctxTable, "version", lua.LString(version))
	L.SetField(ctxTable, "runtimeVersion", lua.LString(RuntimeVersion))

	if err := L.CallByParam(lua.P{
		Fn:      l.pluginObj.RawGetString("PreInstall").(*lua.LFunction),
		NRet:    1,
		Protect: true,
	}, l.pluginObj, ctxTable); err != nil {
		return nil, err
	}

	table := l.returnedValue()
	if table == nil || table.Type() == lua.LTNil {
		return nil, nil
	}
	mainSdk, err := l.parseInfo(table)
	if err != nil {
		return nil, err
	}
	mainSdk.Name = l.Name
	var additionalArr []*Info
	additions := table.RawGetString("addition")
	if tb, ok := additions.(*lua.LTable); ok && tb.Len() != 0 {
		var err error
		additions.(*lua.LTable).ForEach(func(key lua.LValue, value lua.LValue) {
			kvTable, ok := value.(*lua.LTable)
			if !ok {
				err = fmt.Errorf("the return value is not a table")
				return
			}
			info, err := l.parseInfo(kvTable)
			if err != nil {
				return
			}
			if info.Name == "" {
				err = fmt.Errorf("additional file no name provided")
				return
			}
			additionalArr = append(additionalArr, info)
		})
		if err != nil {
			return nil, err
		}
	}

	return &Package{
		Main:      mainSdk,
		Additions: additionalArr,
	}, nil
}

func (l *LuaPlugin) parseInfo(table *lua.LTable) (*Info, error) {
	versionLua := table.RawGetString("version")
	if versionLua == lua.LNil {
		return nil, fmt.Errorf("no version number provided")
	}
	var (
		path    string
		note    string
		name    string
		version string
	)
	version = versionLua.String()

	if urlLua := table.RawGetString("url"); urlLua != lua.LNil {
		path = urlLua.String()
	}
	if noteLua := table.RawGetString("note"); noteLua != lua.LNil {
		note = noteLua.String()
	}
	if nameLua := table.RawGetString("name"); nameLua != lua.LNil {
		name = nameLua.String()
	}
	checksum := l.Checksum(table)
	return &Info{
		Name:     name,
		Version:  Version(version),
		Path:     path,
		Note:     note,
		Checksum: checksum,
	}, nil
}

func (l *LuaPlugin) PostInstall(rootPath string, sdks []*Info) error {
	L := l.state
	sdkArr := L.NewTable()
	for _, v := range sdks {
		sdkTable := l.createSdkInfoTable(v)
		L.SetField(sdkArr, v.Name, sdkTable)
	}
	ctxTable := L.NewTable()
	L.SetField(ctxTable, "sdkInfo", sdkArr)
	L.SetField(ctxTable, "runtimeVersion", lua.LString(RuntimeVersion))
	L.SetField(ctxTable, "rootPath", lua.LString(rootPath))

	function := l.pluginObj.RawGetString("PostInstall")
	if function.Type() == lua.LTNil {
		return nil
	}
	if err := L.CallByParam(lua.P{
		Fn:      function.(*lua.LFunction),
		NRet:    1,
		Protect: true,
	}, l.pluginObj, ctxTable); err != nil {
		return err
	}

	return nil
}

func (l *LuaPlugin) EnvKeys(sdkPackage *Package) (env.Envs, error) {
	L := l.state
	mainInfo := sdkPackage.Main
	sdkArr := L.NewTable()
	for _, v := range sdkPackage.Additions {
		sdkTable := l.createSdkInfoTable(v)
		L.SetField(sdkArr, v.Name, sdkTable)
	}
	ctxTable := L.NewTable()
	L.SetField(ctxTable, "sdkInfo", sdkArr)
	L.SetField(ctxTable, "runtimeVersion", lua.LString(RuntimeVersion))
	// TODO Will be deprecated in future versions
	L.SetField(ctxTable, "path", lua.LString(mainInfo.Path))
	if err := L.CallByParam(lua.P{
		Fn:      l.pluginObj.RawGetString("EnvKeys"),
		NRet:    1,
		Protect: true,
	}, l.pluginObj, ctxTable); err != nil {
		return nil, err
	}

	table := l.returnedValue()
	if table == nil || table.Type() == lua.LTNil || table.Len() == 0 {
		return nil, fmt.Errorf("no environment variables provided")
	}
	var err error
	envKeys := make(env.Envs)
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		kvTable, ok := value.(*lua.LTable)
		if !ok {
			err = fmt.Errorf("the return value is not a table")
			return
		}
		key = kvTable.RawGetString("key")
		value = kvTable.RawGetString("value")
		s := value.String()
		envKeys[key.String()] = &s
	})
	if err != nil {
		return nil, err
	}

	return envKeys, nil
}

func (l *LuaPlugin) getTableField(table *lua.LTable, fieldName string) (lua.LValue, error) {
	value := table.RawGetString(fieldName)
	if value.Type() == lua.LTNil {
		return nil, fmt.Errorf("field '%s' not found", fieldName)
	}
	return value, nil
}

func (l *LuaPlugin) returnedValue() *lua.LTable {
	table := l.state.ToTable(-1) // returned value
	l.state.Pop(1)               // remove received value
	return table
}

func (l *LuaPlugin) Label(version string) string {
	return fmt.Sprintf("%s@%s", l.Name, version)
}

func (l *LuaPlugin) createSdkInfoTable(info *Info) *lua.LTable {
	L := l.state
	sdkTable := L.NewTable()
	L.SetField(sdkTable, "name", lua.LString(info.Name))
	L.SetField(sdkTable, "version", lua.LString(info.Version))
	L.SetField(sdkTable, "path", lua.LString(info.Path))
	L.SetField(sdkTable, "note", lua.LString(info.Note))
	return sdkTable
}

func (l *LuaPlugin) HasFunction(name string) bool {
	return l.pluginObj.RawGetString(name) != lua.LNil
}

func (l *LuaPlugin) PreUse(version Version, previousVersion Version, scope UseScope, cwd string, installedSdks []*Package) (Version, error) {
	L := l.state
	lInstalledSdks := L.NewTable()
	for _, v := range installedSdks {
		sdkTable := l.createSdkInfoTable(v.Main)
		L.SetField(lInstalledSdks, string(v.Main.Version), sdkTable)
	}
	ctxTable := L.NewTable()

	L.SetField(ctxTable, "installedSdks", lInstalledSdks)
	L.SetField(ctxTable, "runtimeVersion", lua.LString(RuntimeVersion))
	L.SetField(ctxTable, "cwd", lua.LString(cwd))
	L.SetField(ctxTable, "scope", lua.LString(scope.String()))
	L.SetField(ctxTable, "version", lua.LString(version))
	L.SetField(ctxTable, "previousVersion", lua.LString(previousVersion))

	function := l.pluginObj.RawGetString("PreUse")
	if function.Type() == lua.LTNil {
		return "", nil
	}
	if err := L.CallByParam(lua.P{
		Fn:      function.(*lua.LFunction),
		NRet:    1,
		Protect: true,
	}, l.pluginObj, ctxTable); err != nil {
		return "", err
	}

	table := l.returnedValue()
	if table == nil || table.Type() == lua.LTNil {
		return "", nil
	}

	luaVer, err := l.getTableField(table, "version")
	if err != nil {
		// ignore version field not found
		return "", nil
	}

	return Version(luaVer.String()), nil
}

func NewLuaPlugin(content, path string, manager *Manager) (*LuaPlugin, error) {
	luaVMInstance := lua.NewState()
	module.Preload(luaVMInstance, manager.Config)

	if err := luaVMInstance.DoString(preloadScript); err != nil {
		return nil, err
	}

	if err := luaVMInstance.DoString(content); err != nil {
		return nil, err
	}

	// set OS_TYPE and ARCH_TYPE
	luaVMInstance.SetGlobal(OsType, lua.LString(manager.osType))
	luaVMInstance.SetGlobal(ArchType, lua.LString(manager.archType))

	pluginObj := luaVMInstance.GetGlobal(LuaPluginObjKey)
	if pluginObj.Type() == lua.LTNil {
		return nil, fmt.Errorf("plugin object not found")
	}

	PLUGIN := pluginObj.(*lua.LTable)

	source := &LuaPlugin{
		state:     luaVMInstance,
		pluginObj: PLUGIN,
		Filepath:  path,
		Filename:  strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	if err := source.checkValid(); err != nil {
		return nil, err
	}

	if name := PLUGIN.RawGetString("name"); name.Type() == lua.LTNil {
		return nil, fmt.Errorf("no plugin name provided")
	} else {
		source.Name = name.String()
		if !isValidName(source.Name) {
			return nil, fmt.Errorf("invalid plugin name")
		}
	}
	if version := PLUGIN.RawGetString("version"); version.Type() != lua.LTNil {
		source.Version = version.String()
	}
	if description := PLUGIN.RawGetString("description"); description.Type() != lua.LTNil {
		source.Description = description.String()
	}
	if updateUrl := PLUGIN.RawGetString("updateUrl"); updateUrl.Type() != lua.LTNil {
		source.UpdateUrl = updateUrl.String()
	}
	if author := PLUGIN.RawGetString("author"); author.Type() != lua.LTNil {
		source.Author = author.String()
	}
	if minRuntimeVersion := PLUGIN.RawGetString("minRuntimeVersion"); minRuntimeVersion.Type() != lua.LTNil {
		source.MinRuntimeVersion = minRuntimeVersion.String()
	}
	return source, nil
}

func isValidName(name string) bool {
	// The regular expression means: start with a letter,
	// followed by any number of letters, digits, or underscores.
	re := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	return re.MatchString(name)
}
