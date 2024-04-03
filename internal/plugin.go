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
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/luai"
	"github.com/version-fox/vfox/internal/util"
	lua "github.com/yuin/gopher-lua"
	"path/filepath"
	"regexp"
)

const (
	luaPluginObjKey = "PLUGIN"
	osType          = "OS_TYPE"
	archType        = "ARCH_TYPE"
	runtime         = "RUNTIME"
)

type HookFunc struct {
	Name     string
	Required bool
	Filename string
}

var (
	// HookFuncMap is a map of built-in hook functions.
	HookFuncMap = map[string]HookFunc{
		"Available":   {Name: "Available", Required: true, Filename: "available"},
		"PreInstall":  {Name: "PreInstall", Required: true, Filename: "pre_install"},
		"EnvKeys":     {Name: "EnvKeys", Required: true, Filename: "env_keys"},
		"PostInstall": {Name: "PostInstall", Required: false, Filename: "post_install"},
		"PreUse":      {Name: "PreUse", Required: false, Filename: "pre_use"},
	}
)

type LuaPlugin struct {
	vm        *luai.LuaVM
	pluginObj *lua.LTable
	// plugin source path
	Path string
	// plugin filename, this is also alias name, sdk-name
	SdkName string
	*LuaPluginInfo
}

func (l *LuaPlugin) Validate() error {
	for _, hf := range HookFuncMap {
		if hf.Required {
			if !l.HasFunction(hf.Name) {
				return fmt.Errorf("[%s] function not found", hf.Name)
			}
		}
	}
	return nil
}

func (l *LuaPlugin) Close() {
	l.vm.Close()
}

func (l *LuaPlugin) Available(args []string) ([]*Package, error) {
	L := l.vm.Instance
	ctxTable, err := luai.Marshal(L, AvailableHookCtx{
		Args: args,
	})

	if err != nil {
		return nil, err
	}

	if err = l.CallFunction("Available", ctxTable); err != nil {
		return nil, err
	}

	table := l.vm.ReturnedValue()

	if table == nil || table.Type() == lua.LTNil {
		return []*Package{}, nil
	}

	hookResult := AvailableHookResult{}
	err = luai.Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}

	var result []*Package

	for _, item := range hookResult {
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
		Version: string(version),
	})

	if err != nil {
		return nil, err
	}

	if err = l.CallFunction("PreInstall", ctxTable); err != nil {
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

	if !l.HasFunction("PostInstall") {
		return nil
	}

	ctx := &PostInstallHookCtx{
		RootPath: rootPath,
		SdkInfo:  make(map[string]*Info),
	}

	for _, v := range sdks {
		ctx.SdkInfo[v.Name] = v
	}

	ctxTable, err := luai.Marshal(L, ctx)
	if err != nil {
		return err
	}

	if err = l.CallFunction("PostInstall", ctxTable); err != nil {
		return err
	}

	return nil
}

func (l *LuaPlugin) EnvKeys(sdkPackage *Package) (*env.Envs, error) {
	L := l.vm.Instance
	mainInfo := sdkPackage.Main

	ctx := &EnvKeysHookCtx{
		// TODO Will be deprecated in future versions
		Path:    mainInfo.Path,
		Main:    mainInfo,
		SdkInfo: make(map[string]*Info),
	}

	for _, v := range sdkPackage.Additions {
		ctx.SdkInfo[v.Name] = v
	}

	ctxTable, err := luai.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}

	if err = l.CallFunction("EnvKeys", ctxTable); err != nil {
		return nil, err
	}

	table := l.vm.ReturnedValue()

	if table == nil || table.Type() == lua.LTNil || table.Len() == 0 {
		return nil, fmt.Errorf("no environment variables provided")
	}

	envKeys := &env.Envs{
		Variables: make(env.Vars),
	}

	var items []*EnvKeysHookResultItem
	err = luai.Unmarshal(table, &items)
	if err != nil {
		return nil, err
	}

	pathSet := env.NewPaths(env.EmptyPaths)
	for _, item := range items {
		if item.Key == "PATH" {
			pathSet.Add(item.Value)
		} else {
			envKeys.Variables[item.Key] = &item.Value
		}
	}

	envKeys.Paths = pathSet

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

	if !l.HasFunction("PreUse") {
		return "", nil
	}

	if err = l.CallFunction("PreUse", ctxTable); err != nil {
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

func (l *LuaPlugin) CallFunction(funcName string, args ...lua.LValue) error {
	logger.Debugf("CallFunction: %s\n", funcName)
	if err := l.vm.CallFunction(l.pluginObj.RawGetString(funcName), append([]lua.LValue{l.pluginObj}, args...)...); err != nil {
		return err
	}
	return nil
}

// NewLuaPlugin creates a new LuaPlugin instance from the specified directory path.
// The plugin directory must meet one of the following conditions:
// - The directory must contain a metadata.lua file and a hooks directory that includes all must be implemented hook functions.
// - The directory contain a main.lua file that defines the plugin object and all hook functions.
func NewLuaPlugin(pluginDirPath string, manager *Manager) (*LuaPlugin, error) {
	vm := luai.NewLuaVM()

	if err := vm.Prepare(&luai.PrepareOptions{
		Config: manager.Config,
	}); err != nil {
		return nil, err
	}

	mainPath := filepath.Join(pluginDirPath, "main.lua")
	// main.lua first
	if util.FileExists(mainPath) {
		vm.LimitPackagePath(filepath.Join(pluginDirPath, "?.lua"))
		if err := vm.Instance.DoFile(mainPath); err != nil {
			return nil, err
		}
	} else {
		// Limit package search scope, hooks directory search priority is higher than lib directory
		hookPath := filepath.Join(pluginDirPath, "hooks", "?.lua")
		libPath := filepath.Join(pluginDirPath, "lib", "?.lua")
		vm.LimitPackagePath(hookPath, libPath)

		// load metadata file
		metadataPath := filepath.Join(pluginDirPath, "metadata.lua")
		if !util.FileExists(metadataPath) {
			return nil, fmt.Errorf("plugin invalid, metadata file not found")
		}

		if err := vm.Instance.DoFile(metadataPath); err != nil {
			return nil, fmt.Errorf("failed to load meatadata file, %w", err)
		}

		// load hook func files
		for _, hf := range HookFuncMap {
			hp := filepath.Join(pluginDirPath, "hooks", hf.Filename+".lua")

			if !hf.Required && !util.FileExists(hp) {
				continue
			}
			if err := vm.Instance.DoFile(hp); err != nil {
				return nil, fmt.Errorf("failed to load [%s] hook function: %s", hf.Name, err.Error())
			}
		}
	}

	// !!!! Must be set after loading the script to prevent overwriting!
	// set OS_TYPE and ARCH_TYPE
	vm.Instance.SetGlobal(osType, lua.LString(util.GetOSType()))
	vm.Instance.SetGlobal(archType, lua.LString(util.GetArchType()))

	r, err := luai.Marshal(vm.Instance, LuaRuntime{
		OsType:        string(util.GetOSType()),
		ArchType:      string(util.GetArchType()),
		Version:       RuntimeVersion,
		PluginDirPath: pluginDirPath,
	})
	if err != nil {
		return nil, err
	}
	vm.Instance.SetGlobal(runtime, r)

	pluginObj := vm.Instance.GetGlobal(luaPluginObjKey)
	if pluginObj.Type() == lua.LTNil {
		return nil, fmt.Errorf("plugin object not found")
	}

	PLUGIN := pluginObj.(*lua.LTable)

	source := &LuaPlugin{
		vm:        vm,
		pluginObj: PLUGIN,
		Path:      pluginDirPath,
		SdkName:   filepath.Base(pluginDirPath),
	}

	if err = source.Validate(); err != nil {
		return nil, err
	}

	pluginInfo := &LuaPluginInfo{}
	if err = luai.Unmarshal(PLUGIN, pluginInfo); err != nil {
		return nil, err
	}

	source.LuaPluginInfo = pluginInfo

	if !isValidName(source.Name) {
		return nil, fmt.Errorf("invalid plugin name")
	}

	if source.Name == "" {
		return nil, fmt.Errorf("no plugin name provided")
	}
	return source, nil
}

func isValidName(name string) bool {
	// The regular expression means: start with a letter,
	// followed by any number of letters, digits, or underscores.
	re := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	return re.MatchString(name)
}
