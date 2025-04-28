package internal

import (
	"fmt"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/luai"
	"github.com/version-fox/vfox/internal/pluginsys"
	"github.com/version-fox/vfox/internal/util"
	lua "github.com/yuin/gopher-lua"
)

type Plugin interface {
	HasFunction(name string) bool
	Close()

	PreInstall(ctx *pluginsys.PreInstallHookCtx) (*pluginsys.PreInstallHookResult, error)
	PostInstall(ctx *pluginsys.PostInstallHookCtx) error

	PreUse(ctx *pluginsys.PreUseHookCtx) (*pluginsys.PreUseHookResult, error)
	ParseLegacyFile(ctx *pluginsys.ParseLegacyFileHookCtx) (*pluginsys.ParseLegacyFileResult, error)
	PreUninstall(ctx *pluginsys.PreUninstallHookCtx) error

	EnvKeys(ctx *pluginsys.EnvKeysHookCtx) ([]*pluginsys.EnvKeysHookResultItem, error)

	Available(ctx *pluginsys.AvailableHookCtx) (*pluginsys.AvailableHookResult, error)
}

type LuaPlugin struct {
	innerPlugin Plugin

	// plugin source path
	Path string
	*pluginsys.PluginInfo
}

// ShowNotes prints the notes of the plugin.
func (l *LuaPlugin) ShowNotes() {
	// print some notes if there are
	if len(l.Notes) != 0 {
		fmt.Println(pterm.Yellow("Notes:"))
		fmt.Println("======")
		for _, note := range l.Notes {
			fmt.Println("  ", note)
		}
	}
}

func (l *LuaPlugin) validate() error {
	for _, hf := range pluginsys.HookFuncMap {
		if hf.Required {
			if !l.innerPlugin.HasFunction(hf.Name) {
				return fmt.Errorf("[%s] function not found", hf.Name)
			}
		}
	}
	return nil
}

func (l *LuaPlugin) Close() {
	l.innerPlugin.Close()
}

func (l *LuaPlugin) Available(args []string) ([]*pluginsys.Package, error) {
	ctx := pluginsys.AvailableHookCtx{
		Args: args,
	}
	result, err := l.innerPlugin.Available(&ctx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []*pluginsys.Package{}, nil
	}

	return pluginsys.CreatePackages(l.Name, *result), nil
}

func (l *LuaPlugin) PreInstall(version Version) (*pluginsys.Package, error) {
	ctx := pluginsys.PreInstallHookCtx{
		Version: string(version),
	}

	result, err := l.innerPlugin.PreInstall(&ctx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &pluginsys.Package{}, nil
	}

	mainSdk, err := result.Info()
	if err != nil {
		return nil, err
	}

	mainSdk.Name = l.Name

	var additionalArr []*pluginsys.Info

	for i, addition := range result.Addition {
		if addition.Name == "" {
			return nil, fmt.Errorf("[PreInstall] additional file %d no name provided", i+1)
		}

		additionalArr = append(additionalArr, addition.Info())
	}

	return &pluginsys.Package{
		Main:      mainSdk,
		Additions: additionalArr,
	}, nil
}

func (l *LuaPlugin) PostInstall(rootPath string, sdks []*pluginsys.Info) error {
	if !l.innerPlugin.HasFunction("PostInstall") {
		return nil
	}

	ctx := &pluginsys.PostInstallHookCtx{
		RootPath: rootPath,
		SdkInfo:  make(map[string]*pluginsys.Info),
	}

	logger.Debugf("PostInstallHookCtx: %+v \n", ctx)
	for _, v := range sdks {
		ctx.SdkInfo[v.Name] = v
	}

	return l.innerPlugin.PostInstall(ctx)
}

func (l *LuaPlugin) EnvKeys(sdkPackage *pluginsys.Package) (*env.Envs, error) {
	mainInfo := sdkPackage.Main

	ctx := &pluginsys.EnvKeysHookCtx{
		// TODO Will be deprecated in future versions
		Path:    mainInfo.Path,
		Main:    mainInfo,
		SdkInfo: make(map[string]*pluginsys.Info),
	}

	for _, v := range sdkPackage.Additions {
		ctx.SdkInfo[v.Name] = v
	}

	logger.Debugf("EnvKeysHookCtx: %+v \n", ctx)
	items, err := l.innerPlugin.EnvKeys(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no environment variables provided")
	}

	envKeys := &env.Envs{
		Variables: make(env.Vars),
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

	logger.Debugf("EnvKeysHookResult: %+v \n", envKeys)
	return envKeys, nil
}

func (l *LuaPlugin) Label(version string) string {
	return fmt.Sprintf("%s@%s", l.Name, version)
}

func (l *LuaPlugin) PreUse(version Version, previousVersion Version, scope UseScope, cwd string, installedSdks []*pluginsys.Package) (Version, error) {
	if !l.innerPlugin.HasFunction("PreUse") {
		logger.Debug("plugin does not have PreUse function")
		return "", nil
	}

	ctx := pluginsys.PreUseHookCtx{
		Cwd:             cwd,
		Scope:           scope.String(),
		Version:         string(version),
		PreviousVersion: string(previousVersion),
		InstalledSdks:   make(map[string]*pluginsys.Info),
	}

	for _, v := range installedSdks {
		lSdk := v.Main
		ctx.InstalledSdks[string(lSdk.Version)] = lSdk
	}

	logger.Debugf("PreUseHookCtx: %+v \n", ctx)

	result, err := l.innerPlugin.PreUse(&ctx)
	if err != nil {
		return "", err
	}

	return Version(result.Version), nil
}

func (l *LuaPlugin) ParseLegacyFile(path string, installedVersions func() []Version) (Version, error) {
	if len(l.LegacyFilenames) == 0 {
		return "", nil
	}
	if !l.innerPlugin.HasFunction("ParseLegacyFile") {
		return "", nil
	}

	filename := filepath.Base(path)

	ctx := pluginsys.ParseLegacyFileHookCtx{
		Filepath: path,
		Filename: filename,
		GetInstalledVersions: func(L *lua.LState) int {
			versions := installedVersions()
			logger.Debugf("Invoking GetInstalledVersions result: %+v \n", versions)
			table, err := luai.Marshal(L, versions)
			if err != nil {
				L.RaiseError(err.Error())
				return 0
			}
			L.Push(table)
			return 1
		},
	}

	logger.Debugf("ParseLegacyFile: %+v \n", ctx)

	result, err := l.innerPlugin.ParseLegacyFile(&ctx)
	if err != nil {
		return "", err
	}

	return Version(result.Version), nil

}

// PreUninstall executes the PreUninstall hook function.
func (l *LuaPlugin) PreUninstall(p *pluginsys.Package) error {
	if !l.innerPlugin.HasFunction("PreUninstall") {
		logger.Debug("plugin does not have PreUninstall function")
		return nil
	}

	ctx := &pluginsys.PreUninstallHookCtx{
		Main:    p.Main,
		SdkInfo: make(map[string]*pluginsys.Info),
	}
	logger.Debugf("PreUninstallHookCtx: %+v \n", ctx)

	for _, v := range p.Additions {
		ctx.SdkInfo[v.Name] = v
	}

	return l.innerPlugin.PreUninstall(ctx)
}

func IsLuaPluginDir(pluginDirPath string) bool {
	metadataPath := filepath.Join(pluginDirPath, "metadata.lua")
	if util.FileExists(metadataPath) {
		return true
	}

	// legacy lua plugin
	hookPath := filepath.Join(pluginDirPath, "main.lua")
	if util.FileExists(hookPath) {
		return true
	}

	return false
}

// NewLuaPlugin creates a new LuaPlugin instance from the specified directory path.
// The plugin directory must meet one of the following conditions:
// - The directory must contain a metadata.lua file and a hooks directory that includes all must be implemented hook functions.
// - The directory contain a main.lua file that defines the plugin object and all hook functions.
func NewLuaPlugin(pluginDirPath string, manager *Manager) (*LuaPlugin, error) {
	config := manager.Config
	plugin2, err := luai.NewLuaPlugin2(pluginDirPath, config, RuntimeVersion)
	if err != nil {
		return nil, err
	}

	source := &LuaPlugin{
		innerPlugin: plugin2,
		Path:        pluginDirPath,
		PluginInfo:  plugin2.PluginInfo,
	}

	if err = source.validate(); err != nil {
		return nil, err
	}

	if !isValidName(source.Name) {
		return nil, fmt.Errorf("invalid plugin name")
	}

	if source.Name == "" {
		return nil, fmt.Errorf("no plugin name provided")
	}

	return source, nil
}
