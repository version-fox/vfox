/*
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
 */

package commands

import (
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/logger"
)

// loadToolChain loads the version files of the scopes, lowest priority first,
// and lets the legacy version files of the project (.nvmrc and the like) set
// the tools that .vfox.toml does not. It is how the versions of a shell are
// worked out, so `vfox env`, `vfox activate` and `vfox doctor` all use it.
func loadToolChain(manager *internal.Manager, scopes ...env.UseScope) (env.VfoxTomlChain, error) {
	runtimeEnvContext := manager.RuntimeEnvContext
	chain, err := runtimeEnvContext.LoadVfoxTomlChainByScopes(scopes...)
	if err != nil {
		return chain, err
	}

	projectToml, ok := chain.GetTomlByScope(env.Project)
	if !ok || projectToml == nil {
		return chain, nil
	}
	// Give a chance to legacy file to set tool versions if not already set in vfox.toml
	// This allows backward compatibility with older projects using legacy files
	// New projects should use vfox.toml directly
	_ = manager.ParseLegacyFile(runtimeEnvContext.CurrentWorkingDir, func(sdkname, version string) {
		// Set only if not already set in vfox.toml
		if _, ok := projectToml.GetToolVersion(sdkname); !ok {
			projectToml.SetTool(sdkname, version)
		}
	})
	return chain, nil
}

// versionInEffect tells `vfox doctor` which version of an SDK the shell uses:
// the one `vfox env` exports, which is the first configured version, highest
// priority first, that is installed (see resolveInstalledToolConfig). The
// version files are read the way env reads them. Sdk.Current is not the same:
// it stops at the highest-priority version and finds none when that one is not
// installed. It returns nil when the version files cannot be read, and the
// caller then goes by Current.
func versionInEffect(manager *internal.Manager) internal.VersionInEffect {
	chain, err := loadToolChain(manager, env.Global, env.Session, env.Project)
	if err != nil {
		logger.Debugf("Cannot tell which versions are in effect: %v\n", err)
		return nil
	}
	return func(s sdk.Sdk) sdk.Version {
		if _, _, version, ok := resolveInstalledToolConfig(chain, s, s.Metadata().Name); ok {
			return version
		}
		return ""
	}
}

func resolveInstalledToolConfig(chain env.VfoxTomlChain, sdkObj sdk.Sdk, sdkName string) (*pathmeta.ToolConfig, env.UseScope, sdk.Version, bool) {
	toolConfigs := chain.GetToolConfigsByPriority(sdkName)
	if len(toolConfigs) == 0 {
		return nil, env.Global, "", false
	}

	for _, toolConfig := range toolConfigs {
		version := sdk.Version(toolConfig.Config.Version)
		if sdkObj.CheckRuntimeExist(version) {
			return toolConfig.Config, toolConfig.Scope, version, true
		}
		logger.Debugf("SDK %s@%s from %s scope not installed, trying lower-priority config",
			sdkName, version, toolConfig.Scope.String())
	}

	logger.Debugf("No installed configured version found for SDK %s", sdkName)
	return nil, env.Global, "", false
}
