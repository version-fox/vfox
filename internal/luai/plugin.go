package luai

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/version-fox/vfox/internal/cache"
	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/pluginsys"
	"github.com/version-fox/vfox/internal/util"
	lua "github.com/yuin/gopher-lua"
)

const (
	luaPluginObjKey = "PLUGIN"
)

type LuaPlugin2 struct {
	vm        *LuaVM
	pluginObj *lua.LTable

	*pluginsys.PluginInfo
}

func (l *LuaPlugin2) HasFunction(name string) bool {
	return l.pluginObj.RawGetString(name) != lua.LNil
}

func (l *LuaPlugin2) Close() {
	l.vm.Close()
}

func (l *LuaPlugin2) Available(ctx *pluginsys.AvailableHookCtx) (*pluginsys.AvailableHookResult, error) {
	L := l.vm.Instance
	ctxTable, err := Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("Available", ctxTable)
	if err != nil {
		return nil, err
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, errors.New("no result provided")
	}

	hookResult := pluginsys.AvailableHookResult{}
	err = Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}

	return &hookResult, nil
}
func (l *LuaPlugin2) PreInstall(ctx *pluginsys.PreInstallHookCtx) (*pluginsys.PreInstallHookResult, error) {
	L := l.vm.Instance
	ctxTable, err := Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("PreInstall", ctxTable)
	if err != nil {
		return nil, err
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, errors.New("no result provided")
	}
	hookResult := pluginsys.PreInstallHookResult{}
	err = Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}
	return &hookResult, nil
}

func (l *LuaPlugin2) EnvKeys(ctx *pluginsys.EnvKeysHookCtx) ([]*pluginsys.EnvKeysHookResultItem, error) {
	L := l.vm.Instance
	ctxTable, err := Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("EnvKeys", ctxTable)
	if err != nil {
		return nil, err
	}
	if table == nil || table.Type() == lua.LTNil || table.Len() == 0 {
		return nil, fmt.Errorf("no environment variables provided")
	}

	var hookResult []*pluginsys.EnvKeysHookResultItem
	err = Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}
	return hookResult, nil
}

// PreUse
func (l *LuaPlugin2) PreUse(ctx *pluginsys.PreUseHookCtx) (*pluginsys.PreUseHookResult, error) {
	L := l.vm.Instance
	ctxTable, err := Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("PreUse", ctxTable)
	if err != nil {
		return nil, err
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, errors.New("no result provided")
	}
	hookResult := pluginsys.PreUseHookResult{}
	err = Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}
	return &hookResult, nil
}

func (l *LuaPlugin2) PreUninstall(ctx *pluginsys.PreUninstallHookCtx) error {
	L := l.vm.Instance
	ctxTable, err := Marshal(L, ctx)
	if err != nil {
		return err
	}
	_, err = l.CallFunction("PreUninstall", ctxTable)
	return err
}

func (l *LuaPlugin2) PostInstall(ctx *pluginsys.PostInstallHookCtx) error {
	L := l.vm.Instance
	ctxTable, err := Marshal(L, ctx)
	if err != nil {
		return err
	}
	_, err = l.CallFunction("PostInstall", ctxTable)
	return err
}

func (l *LuaPlugin2) ParseLegacyFile(ctx *pluginsys.ParseLegacyFileHookCtx) (*pluginsys.ParseLegacyFileResult, error) {
	L := l.vm.Instance
	ctxTable, err := Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("ParseLegacyFile", ctxTable)
	if err != nil {
		return nil, err
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, errors.New("no result provided")
	}
	hookResult := pluginsys.ParseLegacyFileResult{}
	err = Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}
	return &hookResult, nil
}

func (l *LuaPlugin2) CallFunction(funcName string, args ...lua.LValue) (*lua.LTable, error) {
	logger.Debugf("CallFunction: %s\n", funcName)

	table, err := l.vm.CallFunction(l.pluginObj.RawGetString(funcName), append([]lua.LValue{l.pluginObj}, args...)...)

	return table, err
}

func NewLuaPlugin2(pluginDirPath string, config *config.Config, runtimeVersion string) (*LuaPlugin2, error) {
	vm := NewLuaVM()
	if err := vm.Prepare(&PrepareOptions{
		Config: config,
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
			return nil, fmt.Errorf("failed to load metadata file, %w", err)
		}

		// load hook func files
		for _, hf := range pluginsys.HookFuncMap {
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
	vm.Instance.SetGlobal(pluginsys.OsType, lua.LString(util.GetOSType()))
	vm.Instance.SetGlobal(pluginsys.ArchType, lua.LString(util.GetArchType()))

	r, err := Marshal(vm.Instance, pluginsys.RuntimeInfo{
		OsType:        string(util.GetOSType()),
		ArchType:      string(util.GetArchType()),
		Version:       runtimeVersion,
		PluginDirPath: pluginDirPath,
	})
	if err != nil {
		return nil, err
	}

	vm.Instance.SetGlobal(pluginsys.Runtime, r)
	pluginObj := vm.Instance.GetGlobal(luaPluginObjKey)
	if pluginObj.Type() == lua.LTNil {
		return nil, fmt.Errorf("plugin object not found")
	}
	PLUGIN := pluginObj.(*lua.LTable)
	pluginInfo := &pluginsys.PluginInfo{}
	if err = Unmarshal(PLUGIN, pluginInfo); err != nil {
		return nil, err
	}

	source := &LuaPlugin2{
		vm:        vm,
		pluginObj: PLUGIN,

		PluginInfo: pluginInfo,
	}
	// wrap Available hook with Cache.
	if source.HasFunction("Available") {
		targetHook := PLUGIN.RawGetString("Available")
		source.pluginObj.RawSetString("Available", vm.Instance.NewFunction(func(L *lua.LState) int {
			ctxTable := L.CheckTable(2)

			cachePath := filepath.Join(pluginDirPath, "available.cache")
			invokeAvailableHook := func() int {
				logger.Debugf("Calling the original Available hook. \n")
				table, err := vm.CallFunction(targetHook, PLUGIN, ctxTable)
				if err != nil {
					L.RaiseError(err.Error())
					return 0
				}
				if util.FileExists(cachePath) {
					logger.Debugf("Removing the old cache file: %s \n", cachePath)
					_ = os.Remove(cachePath)
				}
				L.Push(table)
				return 1
			}

			logger.Debugf("Available hook cache duration: %v\n", config.Cache.AvailableHookDuration)
			// Cache is disabled
			if config.Cache.AvailableHookDuration == 0 {
				return invokeAvailableHook()
			}

			ctx := &pluginsys.AvailableHookCtx{}
			if err := Unmarshal(ctxTable, ctx); err != nil {
				L.RaiseError(err.Error())
				return 0
			}

			cacheKey := strings.Join(ctx.Args, "##")
			if cacheKey == "" {
				cacheKey = "empty"
			}
			fileCache, err := cache.NewFileCache(cachePath)
			if err != nil {
				return invokeAvailableHook()
			}
			cacheValue, ok := fileCache.Get(cacheKey)
			logger.Debugf("Available hook cache key: %s, hit: %v \n", cacheKey, ok)
			if ok {
				var hookResult []map[string]interface{}
				if err = cacheValue.Unmarshal(&hookResult); err != nil {
					return invokeAvailableHook()
				}
				table, err := Marshal(L, hookResult)
				if err != nil {
					return invokeAvailableHook()
				}
				L.Push(table)
				return 1
			} else {
				table, err := vm.CallFunction(targetHook, PLUGIN, ctxTable)
				if err != nil {
					L.RaiseError(err.Error())
					return 0
				}
				if table == nil || table.Type() == lua.LTNil {
					fileCache.Set(cacheKey, nil, cache.ExpireTime(config.Cache.AvailableHookDuration))
					_ = fileCache.Close()
				} else {
					var hookResult []map[string]interface{}
					if err = Unmarshal(table, &hookResult); err == nil {
						if value, err := cache.NewValue(hookResult); err == nil {
							fileCache.Set(cacheKey, value, cache.ExpireTime(config.Cache.AvailableHookDuration))
							_ = fileCache.Close()
						}
					}
				}
				L.Push(table)
				return 1
			}

		}))

	}
	return source, nil
}
