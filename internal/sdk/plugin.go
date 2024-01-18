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

package sdk

import (
	"fmt"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/module"
	"github.com/version-fox/vfox/internal/util"
	lua "github.com/yuin/gopher-lua"
	"regexp"
)

const (
	LuaPluginObjKey = "PLUGIN"
	OsType          = "OS_TYPE"
	ArchType        = "ARCH_TYPE"
	PluginVersion   = "0.0.1"
)

type LuaPlugin struct {
	state     *lua.LState
	pluginObj *lua.LTable
	// plugin source path
	SourcePath  string
	Name        string
	Author      string
	Version     string
	Description string
	UpdateUrl   string
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
	L.SetField(ctxTable, "plugin_version", lua.LString(PluginVersion))
	if err := L.CallByParam(lua.P{
		Fn:      l.pluginObj.RawGetString("Available").(*lua.LFunction),
		NRet:    1,
		Protect: true,
	}, l.pluginObj, ctxTable); err != nil {
		return nil, err
	}

	table := L.ToTable(-1) // returned value
	L.Pop(1)               // remove received value

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
		v := kvTable.RawGetString("version").String()
		note := kvTable.RawGetString("note").String()
		mainSdk := &Info{
			Name:    l.Name,
			Version: Version(v),
			Note:    note,
		}
		var additionalArr []*Info
		additional := kvTable.RawGetString("addition")
		if tb, ok := additional.(*lua.LTable); ok && tb.Len() != 0 {
			additional.(*lua.LTable).ForEach(func(key lua.LValue, value lua.LValue) {
				itemTable, ok := value.(*lua.LTable)
				if !ok {
					err = fmt.Errorf("the return value is not a table")
					return
				}
				item := Info{
					Name:    itemTable.RawGetString("name").String(),
					Version: Version(itemTable.RawGetString("version").String()),
				}
				additionalArr = append(additionalArr, &item)
			})
		}

		result = append(result, &Package{
			Main:       mainSdk,
			Additional: additionalArr,
		})

	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (l *LuaPlugin) Checksum(table *lua.LTable) (*Checksum, error) {
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
		return NoneChecksum, nil
	}
	return checksum, nil
}

func (l *LuaPlugin) PreInstall(version Version) (*Package, error) {
	L := l.state
	ctxTable := L.NewTable()
	L.SetField(ctxTable, "version", lua.LString(version))

	if err := L.CallByParam(lua.P{
		Fn:      l.pluginObj.RawGetString("PreInstall").(*lua.LFunction),
		NRet:    1,
		Protect: true,
	}, l.pluginObj, ctxTable); err != nil {
		return nil, err
	}

	table := L.ToTable(-1) // returned value
	L.Pop(1)               // remove received value
	if table == nil || table.Type() == lua.LTNil {
		return nil, nil
	}
	v := table.RawGetString("version").String()
	muStr := table.RawGetString("url").String()

	checksum, err := l.Checksum(table)
	if err != nil {
		return nil, err
	}
	mainSdk := &Info{
		Name:     l.Name,
		Version:  Version(v),
		Path:     muStr,
		Checksum: checksum,
	}
	var additionalArr []*Info
	additional := table.RawGetString("addition")
	if tb, ok := additional.(*lua.LTable); ok && tb.Len() != 0 {
		var err error
		additional.(*lua.LTable).ForEach(func(key lua.LValue, value lua.LValue) {
			kvTable, ok := value.(*lua.LTable)
			if !ok {
				err = fmt.Errorf("the return value is not a table")
				return
			}
			s := kvTable.RawGetString("url").String()
			checksum, err = l.Checksum(kvTable)
			if err != nil {
				return
			}
			item := Info{
				Name:     kvTable.RawGetString("name").String(),
				Version:  Version(kvTable.RawGetString("version").String()),
				Path:     s,
				Checksum: checksum,
			}
			additionalArr = append(additionalArr, &item)
		})
		if err != nil {
			return nil, err
		}
	}

	return &Package{
		Main:       mainSdk,
		Additional: additionalArr,
	}, nil
}

func (l *LuaPlugin) PostInstall(rootPath string, sdks []*Info) error {
	L := l.state
	sdkArr := L.NewTable()
	for _, v := range sdks {
		sdkTable := L.NewTable()
		L.SetField(sdkTable, "name", lua.LString(v.Name))
		L.SetField(sdkTable, "version", lua.LString(v.Version))
		L.SetField(sdkTable, "path", lua.LString(v.Path))
		L.SetField(sdkArr, v.Name, sdkTable)
	}
	ctxTable := L.NewTable()
	L.SetField(ctxTable, "sdkInfo", sdkArr)
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
	ctxTable := L.NewTable()
	L.SetField(ctxTable, "path", lua.LString(sdkPackage.Main.Path))
	if len(sdkPackage.Additional) != 0 {
		additionalTable := L.NewTable()
		for _, v := range sdkPackage.Additional {
			L.SetField(additionalTable, v.Name, lua.LString(v.Path))
		}
		L.SetField(ctxTable, "additional_path", additionalTable)
	}
	if err := L.CallByParam(lua.P{
		Fn:      l.pluginObj.RawGetString("EnvKeys"),
		NRet:    1,
		Protect: true,
	}, l.pluginObj, ctxTable); err != nil {
		return nil, err
	}

	table := L.ToTable(-1) // returned value
	L.Pop(1)               // remove received value
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

func NewLuaPlugin(content, path string, osType util.OSType, archType util.ArchType) (*LuaPlugin, error) {
	luaVMInstance := lua.NewState()
	module.Preload(luaVMInstance)
	if err := luaVMInstance.DoString(content); err != nil {
		return nil, err
	}

	// set OS_TYPE and ARCH_TYPE
	luaVMInstance.SetGlobal(OsType, lua.LString(osType))
	luaVMInstance.SetGlobal(ArchType, lua.LString(archType))

	pluginObj := luaVMInstance.GetGlobal(LuaPluginObjKey)
	if pluginObj.Type() == lua.LTNil {
		return nil, fmt.Errorf("plugin object not found")
	}

	PLUGIN := pluginObj.(*lua.LTable)

	source := &LuaPlugin{
		state:      luaVMInstance,
		pluginObj:  PLUGIN,
		SourcePath: path,
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
	return source, nil
}

func isValidName(name string) bool {
	// The regular expression means: start with a letter,
	// followed by any number of letters, digits, or underscores.
	re := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	return re.MatchString(name)
}
