/*
 *
 *    Copyright 2026 Han Li and contributors
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
 *
 */

package plugin

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/plugin/luai"
	"github.com/version-fox/vfox/internal/plugin/luai/codec"
	"github.com/version-fox/vfox/internal/plugin/luai/module"
	"github.com/version-fox/vfox/internal/shared/logger"
	"github.com/version-fox/vfox/internal/shared/util"
	lua "github.com/yuin/gopher-lua"
)

type LuaPlugin struct {
	vm        *luai.LuaVM
	pluginObj *lua.LTable
}

func (l *LuaPlugin) HasFunction(name string) bool {
	return l.pluginObj.RawGetString(name) != lua.LNil
}

func (l *LuaPlugin) Close() {
	l.vm.Close()
}

func (l *LuaPlugin) Available(ctx *AvailableHookCtx) ([]*AvailableHookResultItem, error) {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("Available", ctxTable)
	if err != nil {
		return nil, err
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, ErrNoResultProvide
	}

	var hookResult []*AvailableHookResultItem
	err = codec.Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}

	return hookResult, nil
}
func (l *LuaPlugin) PreInstall(ctx *PreInstallHookCtx) (*PreInstallHookResult, error) {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("PreInstall", ctxTable)
	if err != nil {
		return nil, err
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, ErrNoResultProvide
	}
	hookResult := PreInstallHookResult{}
	err = codec.Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}
	return &hookResult, nil
}

func (l *LuaPlugin) EnvKeys(ctx *EnvKeysHookCtx) ([]*EnvKeysHookResultItem, error) {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("EnvKeys", ctxTable)
	if err != nil {
		return nil, err
	}
	if table == nil || table.Type() == lua.LTNil || table.Len() == 0 {
		return nil, ErrNoResultProvide
	}

	var hookResult []*EnvKeysHookResultItem
	err = codec.Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}
	return hookResult, nil
}

func (l *LuaPlugin) PreUse(ctx *PreUseHookCtx) (*PreUseHookResult, error) {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("PreUse", ctxTable)
	if err != nil {
		return nil, err
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, ErrNoResultProvide
	}
	hookResult := PreUseHookResult{}
	err = codec.Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}
	return &hookResult, nil
}

func (l *LuaPlugin) PreUninstall(ctx *PreUninstallHookCtx) error {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return err
	}
	_, err = l.CallFunction("PreUninstall", ctxTable)
	return err
}

func (l *LuaPlugin) PostInstall(ctx *PostInstallHookCtx) error {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return err
	}
	_, err = l.CallFunction("PostInstall", ctxTable)
	return err
}

func (l *LuaPlugin) ParseLegacyFile(ctx *ParseLegacyFileHookCtx) (*ParseLegacyFileResult, error) {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("ParseLegacyFile", ctxTable)
	if err != nil {
		return nil, err
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, ErrNoResultProvide
	}
	hookResult := ParseLegacyFileResult{}
	err = codec.Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}
	return &hookResult, nil
}

func (l *LuaPlugin) CallFunction(funcName string, args ...lua.LValue) (*lua.LTable, error) {
	logger.Debugf("CallFunction: %s\n", funcName)

	table, err := l.vm.CallFunction(l.pluginObj, funcName, args...)

	return table, err
}

func CreateLuaPlugin(pluginDirPath string, envCtx *env.RuntimeEnvContext) (*LuaPlugin, *Metadata, error) {
	vm := luai.NewLuaVM()
	if err := vm.Prepare(&module.PreloadOptions{
		Config: envCtx.UserConfig,
	}); err != nil {
		return nil, nil, err
	}

	mainPath := filepath.Join(pluginDirPath, "main.lua")
	// main.lua first
	if util.FileExists(mainPath) {
		vm.LimitPackagePath(filepath.Join(pluginDirPath, "?.lua"))
		if err := vm.Instance.DoFile(mainPath); err != nil {
			return nil, nil, err
		}
	} else {
		// Limit package search scope, hooks directory search priority is higher than lib directory
		hookPath := filepath.Join(pluginDirPath, "hooks", "?.lua")
		libPath := filepath.Join(pluginDirPath, "lib", "?.lua")
		vm.LimitPackagePath(hookPath, libPath)

		// load metadata file
		metadataPath := filepath.Join(pluginDirPath, "metadata.lua")
		if !util.FileExists(metadataPath) {
			return nil, nil, fmt.Errorf("plugin invalid, metadata file not found")
		}

		if err := vm.Instance.DoFile(metadataPath); err != nil {
			return nil, nil, fmt.Errorf("failed to load metadata file, %w", err)
		}

		// load hook func files
		for _, hf := range HookFuncMap {
			hp := filepath.Join(pluginDirPath, "hooks", hf.Filename+".lua")

			if !hf.Required && !util.FileExists(hp) {
				continue
			}
			if err := vm.Instance.DoFile(hp); err != nil {
				return nil, nil, fmt.Errorf("failed to load [%s] hook function: %s", hf.Name, err.Error())
			}
		}
	}

	// !!!! Must be set after loading the script to prevent overwriting!
	// set OS_TYPE and ARCH_TYPE
	vm.Instance.SetGlobal(luai.OsType, lua.LString(util.GetOSType()))
	vm.Instance.SetGlobal(luai.ArchType, lua.LString(util.GetArchType()))

	r, err := codec.Marshal(vm.Instance, RuntimeInfo{
		OsType:        string(util.GetOSType()),
		ArchType:      string(util.GetArchType()),
		Version:       envCtx.RuntimeVersion,
		PluginDirPath: pluginDirPath,
	})
	if err != nil {
		return nil, nil, err
	}

	vm.Instance.SetGlobal(luai.Runtime, r)
	pluginObj := vm.Instance.GetGlobal(luai.PluginObjKey)
	if pluginObj.Type() == lua.LTNil {
		return nil, nil, fmt.Errorf("plugin object not found")
	}
	PLUGIN := pluginObj.(*lua.LTable)
	pluginInfo := &Metadata{}
	if err = codec.Unmarshal(PLUGIN, pluginInfo); err != nil {
		return nil, nil, err
	}

	navigator, err := codec.Marshal(vm.Instance, codec.Navigator{
		UserAgent: luai.ComputeUserAgent(envCtx.RuntimeVersion, pluginInfo.Name, pluginInfo.Version),
	})
	if err != nil {
		return nil, nil, err
	}
	vm.Instance.SetGlobal(codec.NavigatorObjKey, navigator)

	source := &LuaPlugin{
		vm:        vm,
		pluginObj: PLUGIN,
	}

	return source, pluginInfo, nil
}
