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
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/luai"
	lua "github.com/yuin/gopher-lua"
)

const (
	LuaPluginObjKey = "PLUGIN"
	OsType          = "OS_TYPE"
	ArchType        = "ARCH_TYPE"
)

type LuaPlugin struct {
	vm        *luai.LuaVM
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
	if l.vm == nil || l.vm.Instance == nil {
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
	l.vm.Close()
}

func (l *LuaPlugin) Available() ([]*Package, error) {
	L := l.vm.Instance
	ctxTable, err := luai.Marshal(L, AvailableHookCtx{
		RuntimeVersion: RuntimeVersion,
	})

	if err != nil {
		return nil, err
	}

	if err := l.vm.CallFunction(l.pluginObj.RawGetString("Available"), l.pluginObj, ctxTable); err != nil {
		return nil, err
	}

	table := l.vm.ReturnedValue()

	if table == nil || table.Type() == lua.LTNil {
		return []*Package{}, nil
	}

	hookResult := []AvailableHookResultItem{}
	err = luai.Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}

	var result []*Package

	for _, item := range hookResult {
		fmt.Printf("item: %+v\n", item)

		mainSdk := &Info{
			Name:    l.Name,
			Version: Version(item.Version),
			Note:    item.Note,
		}

		var additionalArr []*Info

		for i, addition := range item.Addition {
			if addition.Name == "" {
				logger.Errorf("[Available] additional file %d no name provided", i+1)
			}

			additionalArr = append(additionalArr, &Info{
				Name:    addition.Name,
				Version: Version(addition.Version),
				Path:    addition.Path,
				Note:    addition.Note,
			})
		}

		result = append(result, &Package{
			Main:      mainSdk,
			Additions: additionalArr,
		})
	}

	return result, nil
}

func (l *LuaPlugin) PreInstall(version Version) (*Package, error) {
	L := l.vm.Instance
	ctxTable, err := luai.Marshal(L, PreInstallHookCtx{
		Version:        string(version),
		RuntimeVersion: RuntimeVersion,
	})

	if err != nil {
		return nil, err
	}

	if err := l.vm.CallFunction(l.pluginObj.RawGetString("PreInstall"), l.pluginObj, ctxTable); err != nil {
		return nil, err
	}

	table := l.vm.ReturnedValue()
	if table == nil || table.Type() == lua.LTNil {
		return nil, nil
	}

	result := PreInstallHookResult{}

	err = luai.Unmarshal(table, &result)
	if err != nil {
		return nil, err
	}

	mainSdk, err := result.Info()
	if err != nil {
		return nil, err
	}
	mainSdk.Name = l.Name

	var additionalArr []*Info

	for i, addition := range result.Addition {
		if addition.Name == "" {
			return nil, fmt.Errorf("[PreInstall] additional file %d no name provided", i+1)
		}

		additionalArr = append(additionalArr, addition.Info())
	}

	return &Package{
		Main:      mainSdk,
		Additions: additionalArr,
	}, nil
}

func (l *LuaPlugin) PostInstall(rootPath string, sdks []*Info) error {
	L := l.vm.Instance

	function := l.pluginObj.RawGetString("PostInstall")
	if function.Type() == lua.LTNil {
		return nil
	}

	ctx := &PostInstallHookCtx{
		RuntimeVersion: RuntimeVersion,
		RootPath:       rootPath,
		SdkInfo:        make(map[string]*Info),
	}

	for _, v := range sdks {
		ctx.SdkInfo[v.Name] = v
	}

	ctxTable, err := luai.Marshal(L, ctx)
	if err != nil {
		return err
	}

	if err := l.vm.CallFunction(function, l.pluginObj, ctxTable); err != nil {
		return err
	}

	return nil
}

func (l *LuaPlugin) EnvKeys(sdkPackage *Package) (env.Envs, error) {
	L := l.vm.Instance
	mainInfo := sdkPackage.Main

	ctx := &EnvKeysHookCtx{
		// TODO Will be deprecated in future versions
		Path:           mainInfo.Path,
		RuntimeVersion: RuntimeVersion,
		Main:           (mainInfo),
		SdkInfo:        make(map[string]*Info),
	}

	for _, v := range sdkPackage.Additions {
		ctx.SdkInfo[v.Name] = v
	}

	ctxTable, err := luai.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}

	if err = l.vm.CallFunction(l.pluginObj.RawGetString("EnvKeys"), l.pluginObj, ctxTable); err != nil {
		return nil, err
	}

	table := l.vm.ReturnedValue()

	if table == nil || table.Type() == lua.LTNil || table.Len() == 0 {
		return nil, fmt.Errorf("no environment variables provided")
	}

	envKeys := make(env.Envs)

	items := []*EnvKeysHookResultItem{}
	err = luai.Unmarshal(table, &items)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		envKeys[item.Key] = &item.Value
	}

	return envKeys, nil
}

func (l *LuaPlugin) Label(version string) string {
	return fmt.Sprintf("%s@%s", l.Name, version)
}

func (l *LuaPlugin) HasFunction(name string) bool {
	return l.pluginObj.RawGetString(name) != lua.LNil
}

func (l *LuaPlugin) PreUse(version Version, previousVersion Version, scope UseScope, cwd string, installedSdks []*Package) (Version, error) {
	L := l.vm.Instance

	ctx := PreUseHookCtx{
		RuntimeVersion:  RuntimeVersion,
		Cwd:             cwd,
		Scope:           scope.String(),
		Version:         string(version),
		PreviousVersion: string(previousVersion),
		InstalledSdks:   make(map[string]*Info),
	}

	for _, v := range installedSdks {
		lSdk := v.Main
		ctx.InstalledSdks[string(lSdk.Version)] = lSdk
	}

	logger.Debugf("PreUseHookCtx: %+v", ctx)

	ctxTable, err := luai.Marshal(L, ctx)
	if err != nil {
		return "", err
	}

	function := l.pluginObj.RawGetString("PreUse")
	if function.Type() == lua.LTNil {
		return "", nil
	}

	if err := l.vm.CallFunction(function, l.pluginObj, ctxTable); err != nil {
		return "", err
	}

	table := l.vm.ReturnedValue()
	if table == nil || table.Type() == lua.LTNil {
		return "", nil
	}

	result := &PreUseHookResult{}

	if err := luai.Unmarshal(table, result); err != nil {
		return "", err
	}

	return Version(result.Version), nil
}

func NewLuaPlugin(content, path string, manager *Manager) (*LuaPlugin, error) {
	vm := luai.NewLuaVM()

	vm.Prepare(&luai.PrepareOptions{
		Config: manager.Config,
	})

	// set OS_TYPE and ARCH_TYPE
	vm.Instance.SetGlobal(OsType, lua.LString(manager.osType))
	vm.Instance.SetGlobal(ArchType, lua.LString(manager.archType))

	if err := vm.Instance.DoString(content); err != nil {
		return nil, err
	}

	pluginObj := vm.Instance.GetGlobal(LuaPluginObjKey)
	if pluginObj.Type() == lua.LTNil {
		return nil, fmt.Errorf("plugin object not found")
	}

	PLUGIN := pluginObj.(*lua.LTable)

	source := &LuaPlugin{
		vm:        vm,
		pluginObj: PLUGIN,
		Filepath:  path,
		Filename:  strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	}

	if err := source.checkValid(); err != nil {
		return nil, err
	}

	if name := vm.GetTableString(PLUGIN, "name"); name != "" {
		source.Name = name
		if !isValidName(source.Name) {
			return nil, fmt.Errorf("invalid plugin name")
		}
	} else {
		return nil, fmt.Errorf("no plugin name provided")
	}
	if version := vm.GetTableString(PLUGIN, "version"); version != "" {
		source.Version = version
	}
	if description := vm.GetTableString(PLUGIN, "description"); description != "" {
		source.Description = description
	}
	if updateUrl := vm.GetTableString(PLUGIN, "updateUrl"); updateUrl != "" {
		source.UpdateUrl = updateUrl
	}
	if author := vm.GetTableString(PLUGIN, "author"); author != "" {
		source.Author = author
	}
	if minRuntimeVersion := vm.GetTableString(PLUGIN, "minRuntimeVersion"); minRuntimeVersion != "" {
		source.MinRuntimeVersion = minRuntimeVersion
	}
	return source, nil
}

func isValidName(name string) bool {
	// The regular expression means: start with a letter,
	// followed by any number of letters, digits, or underscores.
	re := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	return re.MatchString(name)
}
