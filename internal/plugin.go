/*
 *    Copyright 2025 Han Li and contributors
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
	"fmt"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/luai"
	"github.com/version-fox/vfox/internal/plugin/base"
	"github.com/version-fox/vfox/internal/util"
	lua "github.com/yuin/gopher-lua"

	_ "embed"
	"errors"
	"regexp"
)

const (
	luaPluginObjKey = "PLUGIN"
	osType          = "OS_TYPE"
	archType        = "ARCH_TYPE"
	runtime         = "RUNTIME"
)

var ErrPluginNotFound = errors.New("plugin not found")

func CreatePluginFromPath(tempInstallPath string, manager *Manager) (*PluginWrapper, error) {
	if IsLuaPluginDir(tempInstallPath) {
		luaPlugin, err := NewLuaPlugin(tempInstallPath, manager)
		if err != nil {
			return nil, err
		}
		return luaPlugin, nil
	}

	return nil, ErrPluginNotFound
}

func isValidName(name string) bool {
	// The regular expression means: start with a letter,
	// followed by any number of letters, digits, underscores, or hyphens.
	re := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_\-]*$`)
	return re.MatchString(name)
}

type PluginWrapper struct {
	impl base.Plugin

	// plugin source path
	Path string
	*base.PluginInfo
}

// ShowNotes prints the notes of the plugin.
func (l *PluginWrapper) ShowNotes() {
	// print some notes if there are
	if len(l.Notes) != 0 {
		fmt.Println(pterm.Yellow("Notes:"))
		fmt.Println("======")
		for _, note := range l.Notes {
			fmt.Println("  ", note)
		}
	}
}

func (l *PluginWrapper) validate() error {
	for _, hf := range base.HookFuncMap {
		if hf.Required {
			if !l.impl.HasFunction(hf.Name) {
				return fmt.Errorf("[%s] function not found", hf.Name)
			}
		}
	}
	return nil
}

func (l *PluginWrapper) Close() {
	l.impl.Close()
}

func (l *PluginWrapper) Available(args []string) ([]*base.Package, error) {
	ctx := base.AvailableHookCtx{
		Args: args,
	}
	result, err := l.impl.Available(&ctx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return []*base.Package{}, nil
	}

	return base.CreatePackages(l.Name, *result), nil
}

func (l *PluginWrapper) PreInstall(version Version) (*base.Package, error) {
	ctx := base.PreInstallHookCtx{
		Version: string(version),
	}

	result, err := l.impl.PreInstall(&ctx)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &base.Package{}, nil
	}

	mainSdk, err := result.Info()
	if err != nil {
		return nil, err
	}

	mainSdk.Name = l.Name

	var additionalArr []*base.Info

	for i, addition := range result.Addition {
		if addition.Name == "" {
			return nil, fmt.Errorf("[PreInstall] additional file %d no name provided", i+1)
		}

		additionalArr = append(additionalArr, addition.Info())
	}

	return &base.Package{
		Main:      mainSdk,
		Additions: additionalArr,
	}, nil
}

func (l *PluginWrapper) PostInstall(rootPath string, sdks []*base.Info) error {
	if !l.impl.HasFunction("PostInstall") {
		return nil
	}

	ctx := &base.PostInstallHookCtx{
		RootPath: rootPath,
		SdkInfo:  make(map[string]*base.Info),
	}

	logger.Debugf("PostInstallHookCtx: %+v \n", ctx)
	for _, v := range sdks {
		ctx.SdkInfo[v.Name] = v
	}

	return l.impl.PostInstall(ctx)
}

func (l *PluginWrapper) EnvKeys(sdkPackage *base.Package) (*env.Envs, error) {
	mainInfo := sdkPackage.Main

	ctx := &base.EnvKeysHookCtx{
		// TODO Will be deprecated in future versions
		Path:    mainInfo.Path,
		Main:    mainInfo,
		SdkInfo: make(map[string]*base.Info),
	}

	for _, v := range sdkPackage.Additions {
		ctx.SdkInfo[v.Name] = v
	}

	logger.Debugf("EnvKeysHookCtx: %+v \n", ctx)
	items, err := l.impl.EnvKeys(ctx)
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

func (l *PluginWrapper) Label(version string) string {
	return fmt.Sprintf("%s@%s", l.Name, version)
}

func (l *PluginWrapper) PreUse(version Version, previousVersion Version, scope UseScope, cwd string, installedSdks []*base.Package) (Version, error) {
	if !l.impl.HasFunction("PreUse") {
		logger.Debug("plugin does not have PreUse function")
		return "", nil
	}

	ctx := base.PreUseHookCtx{
		Cwd:             cwd,
		Scope:           scope.String(),
		Version:         string(version),
		PreviousVersion: string(previousVersion),
		InstalledSdks:   make(map[string]*base.Info),
	}

	for _, v := range installedSdks {
		lSdk := v.Main
		ctx.InstalledSdks[string(lSdk.Version)] = lSdk
	}

	logger.Debugf("PreUseHookCtx: %+v \n", ctx)

	result, err := l.impl.PreUse(&ctx)
	if err != nil {
		return "", err
	}

	return Version(result.Version), nil
}

func (l *PluginWrapper) ParseLegacyFile(path string, installedVersions func() []Version) (Version, error) {
	if len(l.LegacyFilenames) == 0 {
		return "", nil
	}
	if !l.impl.HasFunction("ParseLegacyFile") {
		return "", nil
	}

	filename := filepath.Base(path)

	ctx := base.ParseLegacyFileHookCtx{
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

	result, err := l.impl.ParseLegacyFile(&ctx)
	if err != nil {
		return "", err
	}

	return Version(result.Version), nil

}

// PreUninstall executes the PreUninstall hook function.
func (l *PluginWrapper) PreUninstall(p *base.Package) error {
	if !l.impl.HasFunction("PreUninstall") {
		logger.Debug("plugin does not have PreUninstall function")
		return nil
	}

	ctx := &base.PreUninstallHookCtx{
		Main:    p.Main,
		SdkInfo: make(map[string]*base.Info),
	}
	logger.Debugf("PreUninstallHookCtx: %+v \n", ctx)

	for _, v := range p.Additions {
		ctx.SdkInfo[v.Name] = v
	}

	return l.impl.PreUninstall(ctx)
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
func NewLuaPlugin(pluginDirPath string, manager *Manager) (*PluginWrapper, error) {
	config := manager.Config
	plugin2, err := luai.CreateLuaPlugin(pluginDirPath, config, RuntimeVersion)
	if err != nil {
		return nil, err
	}

	source := &PluginWrapper{
		impl:       plugin2,
		Path:       pluginDirPath,
		PluginInfo: plugin2.PluginInfo,
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
