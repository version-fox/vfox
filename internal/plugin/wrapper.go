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
)

// Wrapper wraps a Plugin with its metadata and installed path.
type Wrapper struct {
	*Metadata            // Metadata is the metadata of the plugin
	Plugin               // Plugin is the plugin instance
	InstalledPath string // InstalledPath is the path where the plugin is installed
}

func (l *Wrapper) validate() error {
	if l.Name == "" {
		return fmt.Errorf("no plugin name provided")
	}

	if !isValidName(l.Name) {
		return fmt.Errorf("invalid plugin name [%s]", l.Name)
	}

	for _, hf := range HookFuncMap {
		if hf.Required {
			if !l.HasFunction(hf.Name) {
				return fmt.Errorf("[%s] function not found", hf.Name)
			}
		}
	}
	return nil
}

// IsNoResultProvided checks if the error indicates that no result was provided.
func (l *Wrapper) IsNoResultProvided(err error) bool {
	return errors.Is(err, ErrNoResultProvide)
}

//func (l *Wrapper) HasFunction(name string) bool {
//	return l.impl.HasFunction(name)
//}
//

//func (l *Wrapper) Close() {
//	l.impl.Close()
//}
//func (l *Wrapper) invokeAvailable(args []string) ([]*AvailableHookResultItem, error) {
//	logger.Debug("Calling Available hook")
//	ctx := AvailableHookCtx{
//		Args: args,
//	}
//	hookResult, err := l.impl.Available(&ctx)
//	if l.isNoResultProvided(err) {
//		return []*AvailableHookResultItem{}, nil
//	}
//	return hookResult, err
//}
//
//func (l *Wrapper) Available(args []string) ([]*AvailableHookResultItem, error) {
//	// TODO: check if have write permission
//	cachePath := filepath.Join(l.InstalledPath, ".available.cache")
//	cacheDuration := l.AvailableHookDuration
//	logger.Debugf("Available hook cache duration: %v\n", cacheDuration)
//
//	// Cache is disabled
//	if cacheDuration == 0 {
//		return l.invokeAvailable(args)
//	}
//
//	cacheKey := strings.Join(args, "##")
//	if cacheKey == "" {
//		cacheKey = "empty"
//	}
//
//	fileCache, err := cache.NewFileCache(cachePath)
//	if err == nil {
//		cacheValue, ok := fileCache.Get(cacheKey)
//		logger.Debugf("Available hook cache key: %s, hit: %+v \n", cacheKey, ok)
//		if ok {
//			var hookResult []*AvailableHookResultItem
//			if err = cacheValue.Unmarshal(&hookResult); err == nil {
//				return hookResult, nil
//			}
//		}
//	}
//
//	result, err := l.invokeAvailable(args)
//	if err != nil {
//		return result, err
//	}
//
//	if result == nil {
//		fileCache.Set(cacheKey, nil, cache.ExpireTime(cacheDuration))
//	}
//
//	if value, err := cache.NewValue(result); err == nil {
//		logger.Debugf("Available hook cache set\n")
//		fileCache.Set(cacheKey, value, cache.ExpireTime(cacheDuration))
//		_ = fileCache.Close()
//	}
//
//	return result, nil
//}
//
//func (l *Wrapper) PreInstall(version base.Version) (*PreInstallHookResult, error) {
//	ctx := PreInstallHookCtx{
//		Version: string(version),
//	}
//
//	result, err := l.impl.PreInstall(&ctx)
//	if l.isNoResultProvided(err) {
//		return &PreInstallHookResult{}, nil
//	}
//	if err != nil {
//		return nil, err
//	}
//	result.Name = l.Name
//	for i, addition := range result.Addition {
//		if addition.Name == "" {
//			return nil, fmt.Errorf("[PreInstall] additional file %d no name provided", i+1)
//		}
//	}
//	return result, nil
//}
//
//func (l *Wrapper) PostInstall(rootPath string, sdks []*base.Info) error {
//	if !l.HasFunction("PostInstall") {
//		return nil
//	}
//
//	ctx := &PostInstallHookCtx{
//		RootPath: rootPath,
//		SdkInfo:  make(map[string]*base.Info),
//	}
//
//	logger.Debugf("PostInstallHookCtx: %+v \n", ctx)
//	for _, v := range sdks {
//		ctx.SdkInfo[v.Name] = v
//	}
//
//	return l.impl.PostInstall(ctx)
//}
//
//func (l *Wrapper) EnvKeys(sdkPackage *EnvKeysHookCtx) (*env.Envs, error) {
//	mainInfo := sdkPackage.Main
//
//	ctx := &EnvKeysHookCtx{
//		Path:    mainInfo.Path,
//		Main:    mainInfo,
//		SdkInfo: make(map[string]*base.Info),
//	}
//
//	for _, v := range sdkPackage.Additions {
//		ctx.SdkInfo[v.Name] = v
//	}
//
//	logger.Debugf("EnvKeysHookCtx: %+v \n", ctx)
//	items, err := l.impl.EnvKeys(ctx)
//	if l.isNoResultProvided(err) {
//		return nil, fmt.Errorf("no environment variables provided")
//	}
//	if err != nil {
//		return nil, err
//	}
//
//	envKeys := &env.Envs{
//		Variables: make(env.Vars),
//	}
//
//	pathSet := env.NewPaths(env.EmptyPaths)
//	for _, item := range items {
//		if item.Key == "PATH" {
//			pathSet.Add(item.Value)
//		} else {
//			envKeys.Variables[item.Key] = &item.Value
//		}
//	}
//
//	envKeys.Paths = pathSet
//
//	logger.Debugf("EnvKeysHookResult: %+v \n", envKeys)
//	return envKeys, nil
//}
//
//func (l *Wrapper) preUse(version base.Version, previousVersion base.Version, scope base.UseScope, cwd string, installedSdks []*Package) (base.Version, error) {
//	if !l.HasFunction("preUse") {
//		logger.Debug("plugin does not have preUse function")
//		return "", nil
//	}
//
//	ctx := PreUseHookCtx{
//		Cwd:             cwd,
//		Scope:           scope.String(),
//		Version:         string(version),
//		PreviousVersion: string(previousVersion),
//		InstalledSdks:   make(map[string]*base.Info),
//	}
//
//	for _, v := range installedSdks {
//		lSdk := v.Main
//		ctx.InstalledSdks[string(lSdk.Version)] = lSdk
//	}
//
//	logger.Debugf("PreUseHookCtx: %+v \n", ctx)
//
//	result, err := l.impl.preUse(&ctx)
//
//	if l.isNoResultProvided(err) {
//		return "", nil
//	}
//
//	if err != nil {
//		return "", err
//	}
//
//	return result.Version, nil
//}
//
//func (l *Wrapper) ParseLegacyFile(path string, installedVersions func() []base.Version) (base.Version, error) {
//	if len(l.LegacyFilenames) == 0 {
//		return "", nil
//	}
//	if !l.HasFunction("ParseLegacyFile") {
//		return "", nil
//	}
//
//	filename := filepath.Base(path)
//
//	ctx := ParseLegacyFileHookCtx{
//		Filepath: path,
//		Filename: filename,
//		GetInstalledVersions: func() []base.Version {
//			versions := installedVersions()
//			logger.Debugf("Invoking GetInstalledVersions result: %+v \n", versions)
//			return versions
//		},
//		Strategy: l.config.LegacyVersionFile.Strategy,
//	}
//
//	logger.Debugf("ParseLegacyFile: %+v \n", ctx)
//
//	result, err := l.impl.ParseLegacyFile(&ctx)
//	if err != nil {
//		return "", err
//	}
//
//	return result.Version, nil
//
//}
//
//func (l *Wrapper) PreUninstall(p *Package) error {
//	if !l.HasFunction("PreUninstall") {
//		logger.Debug("plugin does not have PreUninstall function")
//		return nil
//	}
//
//	ctx := &PreUninstallHookCtx{
//		Main:    p.Main,
//		SdkInfo: make(map[string]*base.Info),
//	}
//	logger.Debugf("PreUninstallHookCtx: %+v \n", ctx)
//
//	for _, v := range p.Additions {
//		ctx.SdkInfo[v.Name] = v
//	}
//
//	return l.impl.PreUninstall(ctx)
//}
//
//func (l *Wrapper) isNoResultProvided(err error) bool {
//	return errors.Is(err, ErrNoResultProvide)
//}
