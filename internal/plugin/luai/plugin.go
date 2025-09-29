package luai

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/version-fox/vfox/internal/base"
	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/plugin/luai/codec"
	"github.com/version-fox/vfox/internal/plugin/luai/module"
	"github.com/version-fox/vfox/internal/util"
	lua "github.com/yuin/gopher-lua"
)

// LuaError represents a structured Lua error for CLI tools
type LuaError struct {
	PluginName string
	HookName   string
	File       string
	Line       int
	Message    string
	OriginalErr error
}

// Error implements the error interface
func (e *LuaError) Error() string {
	var parts []string
	
	// Plugin and hook context
	if e.PluginName != "" && e.HookName != "" {
		parts = append(parts, fmt.Sprintf("plugin %s [%s] failed", e.PluginName, e.HookName))
	}
	
	// File and line information
	if e.File != "" {
		filename := filepath.Base(e.File)
		if e.Line > 0 {
			parts = append(parts, fmt.Sprintf("at %s:%d", filename, e.Line))
		} else {
			parts = append(parts, fmt.Sprintf("in %s", filename))
		}
	}
	
	// Error message
	if e.Message != "" {
		parts = append(parts, e.Message)
	}
	
	return strings.Join(parts, ": ")
}

// Unwrap returns the original error for error unwrapping
func (e *LuaError) Unwrap() error {
	return e.OriginalErr
}

type LuaPlugin struct {
	vm        *LuaVM
	pluginObj *lua.LTable

	*base.PluginInfo
}

// wrapLuaError wraps a Lua error with enhanced context information
func (l *LuaPlugin) wrapLuaError(err error, hookName string) error {
	if err == nil {
		return nil
	}
	
	// If it's already a LuaError, don't double-wrap
	var luaErr *LuaError
	if errors.As(err, &luaErr) {
		return err
	}
	
	return l.parseLuaError(err.Error(), hookName, err)
}

// parseLuaError parses a raw Lua error string into a structured LuaError
func (l *LuaPlugin) parseLuaError(rawError, hookName string, originalErr error) *LuaError {
	err := &LuaError{
		PluginName:  l.Name,
		HookName:    hookName,
		Message:     rawError,
		OriginalErr: originalErr,
	}

	// Extract main error information from the first line
	lines := strings.Split(rawError, "\n")
	if len(lines) == 0 {
		return err
	}

	firstLine := strings.TrimSpace(lines[0])
	
	// Try to extract file:line: message format
	if matches := regexp.MustCompile(`^(.+):(\d+): (.+)$`).FindStringSubmatch(firstLine); len(matches) == 4 {
		err.File = matches[1]
		if line, parseErr := l.parseInt(matches[2]); parseErr == nil {
			err.Line = line
		}
		err.Message = matches[3]
	}

	// Clean up common Lua error messages for better CLI output
	err.Message = l.cleanErrorMessage(err.Message)
	
	return err
}

// cleanErrorMessage cleans up common Lua error patterns for better CLI output
func (l *LuaPlugin) cleanErrorMessage(message string) string {
	// Remove redundant "Compilation Failure" wrapper
	if strings.Contains(message, "Compilation Failure") {
		return "compilation failed"
	}
	
	// Simplify common error patterns
	replacements := map[string]string{
		"attempt to call a nil value":     "function not found",
		"attempt to index a nil value":    "variable not initialized", 
		"attempt to index field":          "invalid field access",
		"bad argument":                    "invalid argument",
		"stack overflow":                  "infinite recursion detected",
		"permission denied":               "permission denied",
		"no such file or directory":       "file not found",
	}
	
	lower := strings.ToLower(message)
	for pattern, replacement := range replacements {
		if strings.Contains(lower, pattern) {
			return replacement
		}
	}
	
	return message
}

// parseInt safely parses a string to int
func (l *LuaPlugin) parseInt(s string) (int, error) {
	var result int
	_, parseErr := fmt.Sscanf(s, "%d", &result)
	return result, parseErr
}

func (l *LuaPlugin) HasFunction(name string) bool {
	return l.pluginObj.RawGetString(name) != lua.LNil
}

func (l *LuaPlugin) Close() {
	l.vm.Close()
}

func (l *LuaPlugin) Available(ctx *base.AvailableHookCtx) ([]*base.AvailableHookResultItem, error) {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("Available", ctxTable)
	if err != nil {
		return nil, l.wrapLuaError(err, "Available")
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, base.ErrNoResultProvide
	}

	var hookResult []*base.AvailableHookResultItem
	err = codec.Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}

	return hookResult, nil
}
func (l *LuaPlugin) PreInstall(ctx *base.PreInstallHookCtx) (*base.PreInstallHookResult, error) {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("PreInstall", ctxTable)
	if err != nil {
		return nil, l.wrapLuaError(err, "PreInstall")
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, base.ErrNoResultProvide
	}
	hookResult := base.PreInstallHookResult{}
	err = codec.Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}
	return &hookResult, nil
}

func (l *LuaPlugin) EnvKeys(ctx *base.EnvKeysHookCtx) ([]*base.EnvKeysHookResultItem, error) {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("EnvKeys", ctxTable)
	if err != nil {
		return nil, l.wrapLuaError(err, "EnvKeys")
	}
	if table == nil || table.Type() == lua.LTNil || table.Len() == 0 {
		return nil, base.ErrNoResultProvide
	}

	var hookResult []*base.EnvKeysHookResultItem
	err = codec.Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}
	return hookResult, nil
}

func (l *LuaPlugin) PreUse(ctx *base.PreUseHookCtx) (*base.PreUseHookResult, error) {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("PreUse", ctxTable)
	if err != nil {
		return nil, l.wrapLuaError(err, "PreUse")
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, base.ErrNoResultProvide
	}
	hookResult := base.PreUseHookResult{}
	err = codec.Unmarshal(table, &hookResult)
	if err != nil {
		return nil, errors.New("failed to unmarshal the return value: " + err.Error())
	}
	return &hookResult, nil
}

func (l *LuaPlugin) PreUninstall(ctx *base.PreUninstallHookCtx) error {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return err
	}
	_, err = l.CallFunction("PreUninstall", ctxTable)
	if err != nil {
		return l.wrapLuaError(err, "PreUninstall")
	}
	return nil
}

func (l *LuaPlugin) PostInstall(ctx *base.PostInstallHookCtx) error {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return err
	}
	_, err = l.CallFunction("PostInstall", ctxTable)
	if err != nil {
		return l.wrapLuaError(err, "PostInstall")
	}
	return nil
}

func (l *LuaPlugin) ParseLegacyFile(ctx *base.ParseLegacyFileHookCtx) (*base.ParseLegacyFileResult, error) {
	L := l.vm.Instance
	ctxTable, err := codec.Marshal(L, ctx)
	if err != nil {
		return nil, err
	}
	table, err := l.CallFunction("ParseLegacyFile", ctxTable)
	if err != nil {
		return nil, l.wrapLuaError(err, "ParseLegacyFile")
	}
	if table == nil || table.Type() == lua.LTNil {
		return nil, base.ErrNoResultProvide
	}
	hookResult := base.ParseLegacyFileResult{}
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

func CreateLuaPlugin(pluginDirPath string, config *config.Config, runtimeVersion string) (*LuaPlugin, error) {
	vm := NewLuaVM()
	if err := vm.Prepare(&module.PreloadOptions{
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
		for _, hf := range base.HookFuncMap {
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
	vm.Instance.SetGlobal(base.OsType, lua.LString(util.GetOSType()))
	vm.Instance.SetGlobal(base.ArchType, lua.LString(util.GetArchType()))

	r, err := codec.Marshal(vm.Instance, base.RuntimeInfo{
		OsType:        string(util.GetOSType()),
		ArchType:      string(util.GetArchType()),
		Version:       runtimeVersion,
		PluginDirPath: pluginDirPath,
	})
	if err != nil {
		return nil, err
	}

	vm.Instance.SetGlobal(base.Runtime, r)
	pluginObj := vm.Instance.GetGlobal(base.PluginObjKey)
	if pluginObj.Type() == lua.LTNil {
		return nil, fmt.Errorf("plugin object not found")
	}
	PLUGIN := pluginObj.(*lua.LTable)
	pluginInfo := &base.PluginInfo{}
	if err = codec.Unmarshal(PLUGIN, pluginInfo); err != nil {
		return nil, err
	}

	navigator, err := codec.Marshal(vm.Instance, base.Navigator{
		UserAgent: computeUserAgent(runtimeVersion, pluginInfo.Name, pluginInfo.Version),
	})
	if err != nil {
		return nil, err
	}
	vm.Instance.SetGlobal(base.NavigatorObjKey, navigator)

	source := &LuaPlugin{
		vm:         vm,
		pluginObj:  PLUGIN,
		PluginInfo: pluginInfo,
	}

	return source, nil
}
