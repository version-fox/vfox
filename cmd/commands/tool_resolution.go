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
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/logger"
)

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
