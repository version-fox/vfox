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
	"context"
	"sort"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/logger"
)

var Reshim = &cli.Command{
	Name:  "reshim",
	Usage: "Re-sync stable shims for installed tools (generic, plugin-driven)",
	Description: "Recreate scope links for the currently selected versions and re-run " +
		"each plugin's optional PostUse hook so version-independent shims " +
		"(e.g. Volta-style global-package shims pinned to the install-time runtime) " +
		"are reconciled. vfox never interprets shim contents.",
	Category: CategorySDK,
	Action:   reshimCmd,
}

func reshimCmd(ctx context.Context, cmd *cli.Command) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	runtimeEnvContext := manager.RuntimeEnvContext
	chain, err := runtimeEnvContext.LoadVfoxTomlChainByScopes(env.Global, env.Session, env.Project)
	if err != nil {
		return err
	}

	allTools := chain.GetAllTools()
	if len(allTools) == 0 {
		pterm.Println("No tools configured.")
		return nil
	}

	names := make([]string, 0, len(allTools))
	for name := range allTools {
		names = append(names, name)
	}
	sort.Strings(names)

	reshimmed := 0
	for _, sdkName := range names {
		sdkObj, err := manager.LookupSdk(sdkName)
		if err != nil {
			logger.Debugf("SDK %s not found: %v", sdkName, err)
			continue
		}
		toolConfig, scope, version, ok := resolveInstalledToolConfig(chain, sdkObj, sdkName)
		if !ok {
			continue
		}
		actualScope := scope
		if env.Project == scope && sdk.IsUseUnLink(toolConfig.Attr) {
			actualScope = env.Session
		}
		if err := sdkObj.CreateSymlinksForScope(version, actualScope); err != nil {
			logger.Debugf("Failed to create symlinks for %s@%s: %v", sdkName, version, err)
			continue
		}
		// No version switch here, so previous == current.
		if err := sdkObj.TriggerPostUse(version, actualScope, version); err != nil {
			return err
		}
		reshimmed++
	}

	pterm.Printf("Reshimmed %d tool(s).\n", reshimmed)
	return nil
}
