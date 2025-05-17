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

package plugin

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal/base"
	"github.com/version-fox/vfox/internal/cache"
	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/plugin/luai"
	"github.com/version-fox/vfox/internal/util"

	_ "embed"
	"errors"
	"regexp"
)

var ErrPluginNotFound = errors.New("plugin not found")

func CreatePluginFromPath(tempInstallPath string, config *config.Config, runtimeVersion string) (*PluginWrapper, error) {
	if IsLuaPluginDir(tempInstallPath) {
		luaPlugin, err := NewLuaPlugin(tempInstallPath, config, runtimeVersion)
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
	impl   base.Plugin
	config config.Config

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

func (l *PluginWrapper) HasFunction(name string) bool {
	return l.impl.HasFunction(name)
}

func (l *PluginWrapper) validate() error {
	if l.Name == "" {
		return fmt.Errorf("no plugin name provided")
	}

	if !isValidName(l.Name) {
		return fmt.Errorf("invalid plugin name [%s]", l.Name)
	}

	for _, hf := range base.HookFuncMap {
		if hf.Required {
			if !l.HasFunction(hf.Name) {
				return fmt.Errorf("[%s] function not found", hf.Name)
			}
		}
	}

	return nil
}

func (l *PluginWrapper) Close() {
	l.impl.Close()
}
func (l *PluginWrapper) invokeAvailable(args []string) ([]*base.Package, error) {
	logger.Debug("Calling Available hook")
	ctx := base.AvailableHookCtx{
		Args: args,
	}
	hookResult, err := l.impl.Available(&ctx)
	if l.isNoResultProvided(err) {
		return []*base.Package{}, nil
	}

	if err != nil {
		return nil, err
	}

	var result []*base.Package
	for _, item := range hookResult {
		mainSdk := &base.Info{
			Name:    l.Name,
			Version: item.Version,
			Note:    item.Note,
		}

		var additionalArr []*base.Info

		for i, addition := range item.Addition {
			if addition.Name == "" {
				logger.Errorf("[Available] additional %d no name provided", i+1)
			}

			additionalArr = append(additionalArr, &base.Info{
				Name:    addition.Name,
				Version: addition.Version,
				Path:    addition.Path,
				Note:    addition.Note,
			})
		}

		result = append(result, &base.Package{
			Main:      mainSdk,
			Additions: additionalArr,
		})
	}

	return result, nil
}

func (l *PluginWrapper) Available(args []string) ([]*base.Package, error) {
	cachePath := filepath.Join(l.Path, ".available.cache")
	cacheDuration := l.config.Cache.AvailableHookDuration
	logger.Debugf("Available hook cache duration: %v\n", cacheDuration)

	// Cache is disabled
	if cacheDuration == 0 {
		return l.invokeAvailable(args)
	}

	cacheKey := strings.Join(args, "##")
	if cacheKey == "" {
		cacheKey = "empty"
	}

	fileCache, err := cache.NewFileCache(cachePath)
	if err == nil {
		cacheValue, ok := fileCache.Get(cacheKey)
		logger.Debugf("Available hook cache key: %s, hit: %+v \n", cacheKey, ok)
		if ok {
			var hookResult []*base.Package
			if err = cacheValue.Unmarshal(&hookResult); err == nil {
				return hookResult, nil
			}
		}
	}

	result, err := l.invokeAvailable(args)
	if err != nil {
		return result, err
	}

	if result == nil {
		fileCache.Set(cacheKey, nil, cache.ExpireTime(cacheDuration))
	}

	if value, err := cache.NewValue(result); err == nil {
		logger.Debugf("Available hook cache set\n")
		fileCache.Set(cacheKey, value, cache.ExpireTime(cacheDuration))
		_ = fileCache.Close()
	}

	return result, nil
}

func (l *PluginWrapper) PreInstall(version base.Version) (*base.Package, error) {
	ctx := base.PreInstallHookCtx{
		Version: string(version),
	}

	result, err := l.impl.PreInstall(&ctx)
	if l.isNoResultProvided(err) {
		return &base.Package{}, nil
	}
	if err != nil {
		return nil, err
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
	if !l.HasFunction("PostInstall") {
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
	if l.isNoResultProvided(err) {
		return nil, fmt.Errorf("no environment variables provided")
	}
	if err != nil {
		return nil, err
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

func (l *PluginWrapper) PreUse(version base.Version, previousVersion base.Version, scope base.UseScope, cwd string, installedSdks []*base.Package) (base.Version, error) {
	if !l.HasFunction("PreUse") {
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

	if l.isNoResultProvided(err) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	return result.Version, nil
}

func (l *PluginWrapper) ParseLegacyFile(path string, installedVersions func() []base.Version) (base.Version, error) {
	if len(l.LegacyFilenames) == 0 {
		return "", nil
	}
	if !l.HasFunction("ParseLegacyFile") {
		return "", nil
	}

	filename := filepath.Base(path)

	ctx := base.ParseLegacyFileHookCtx{
		Filepath: path,
		Filename: filename,
		GetInstalledVersions: func() []base.Version {
			versions := installedVersions()
			logger.Debugf("Invoking GetInstalledVersions result: %+v \n", versions)
			return versions
		},
	}

	logger.Debugf("ParseLegacyFile: %+v \n", ctx)

	result, err := l.impl.ParseLegacyFile(&ctx)
	if err != nil {
		return "", err
	}

	return result.Version, nil

}

func (l *PluginWrapper) PreUninstall(p *base.Package) error {
	if !l.HasFunction("PreUninstall") {
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

func (l *PluginWrapper) isNoResultProvided(err error) bool {
	return errors.Is(err, base.ErrNoResultProvide)
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
func NewLuaPlugin(pluginDirPath string, config *config.Config, runtimeVersion string) (*PluginWrapper, error) {
	plugin, err := luai.CreateLuaPlugin(pluginDirPath, config, runtimeVersion)
	if err != nil {
		return nil, err
	}

	source := &PluginWrapper{
		impl:       plugin,
		Path:       pluginDirPath,
		PluginInfo: plugin.PluginInfo,
		config:     *config,
	}

	if err = source.validate(); err != nil {
		return nil, err
	}

	return source, nil
}
