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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/logger"
	"github.com/version-fox/vfox/internal/shell"
	"golang.org/x/sync/errgroup"
)

var Env = &cli.Command{
	Name: "env",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "shell",
			Aliases: []string{"s"},
			Usage:   "shell name",
		},
		&cli.BoolFlag{
			Name:    "cleanup",
			Aliases: []string{"c"},
			Usage:   "cleanup old temp files",
		},
		&cli.BoolFlag{
			Name:    "json",
			Aliases: []string{"j"},
			Usage:   "output json format",
		},
		&cli.BoolFlag{
			Name:  "full",
			Usage: "output full env",
		},
	},
	Action:   envCmd,
	Category: CategorySDK,
}

func envCmd(ctx context.Context, cmd *cli.Command) error {
	if cmd.IsSet("json") {
		return outputJSON()
	} else if cmd.IsSet("cleanup") {
		return cleanTmp()
	} else {
		return envFlag(cmd)
	}
}

func outputJSON() error {
	type SDKs map[string]map[string]*string
	data := struct {
		IsHookEnv bool     `json:"is_hook_env"`
		Paths     []string `json:"paths"`
		SDKs      SDKs     `json:"sdks"`
	}{
		IsHookEnv: env.IsHookEnv(),
		Paths:     []string{},
		SDKs:      make(SDKs),
	}
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	// Load all scope configs with priority: Global < Session < Project
	chain, err := manager.RuntimeEnvContext.LoadVfoxTomlChainByScopes(env.Global, env.Session, env.Project)
	if err != nil {
		return err
	}

	// Filter tools to only include those that are installed
	allTools := chain.GetAllTools()
	for name, version := range allTools {
		if lookupSdk, err := manager.LookupSdk(name); err == nil {
			if runtimePackage, err := lookupSdk.GetRuntimePackage(sdk.Version(version)); err == nil {
				if keys, err := lookupSdk.EnvKeys(runtimePackage); err == nil {
					metadata := lookupSdk.Metadata()
					data.SDKs[metadata.Name] = keys.Variables
					data.Paths = append(data.Paths, keys.Paths.Slice()...)
				}
			}
		}
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

func cleanTmp() error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	// Clean up the old temp files, before today.
	manager.CleanTmp()
	return nil
}

func envFlag(cmd *cli.Command) error {
	shellName := cmd.String("shell")
	if shellName == "" {
		return cli.Exit("shell name is required", 1)
	}
	s := shell.NewShell(shellName)
	if s == nil {
		return fmt.Errorf("unknown target shell %s", shellName)
	}

	// Create manager
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	runtimeEnvContext := manager.RuntimeEnvContext

	// 1. Load configs from all scopes with priority: Global < Session < Project
	chain, err := runtimeEnvContext.LoadVfoxTomlChainByScopes(env.Global, env.Session, env.Project)
	if err != nil {
		return err
	}

	projectToml, _ := chain.GetTomlByScope(env.Project)
	// Give a chance to legacy file to set tool versions if not already set in vfox.toml
	// This allows backward compatibility with older projects using legacy files
	// New projects should use vfox.toml directly
	_ = manager.ParseLegacyFile(runtimeEnvContext.CurrentWorkingDir, func(sdkname, version string) {
		// Set only if not already set in vfox.toml
		if _, ok := projectToml.GetToolVersion(sdkname); !ok {
			projectToml.SetTool(sdkname, version)
		}
	})

	// 2. Collect config file paths for change detection
	// Only include paths that actually exist to avoid false positives
	configPaths := map[env.UseScope]string{}
	for _, scope := range []env.UseScope{env.Global, env.Session, env.Project} {
		if toml, ok := chain.GetTomlByScope(scope); ok && toml != nil && toml.Path != "" {
			// Only track files that exist (empty configs have a Path but don't exist yet)
			if _, err := os.Stat(toml.Path); err == nil {
				configPaths[scope] = toml.Path
			}
		}
	}

	// 3. Initialize and load state
	// State file path: ~/.vfox/tmp/env-state.json
	stateFile := filepath.Join(runtimeEnvContext.PathMeta.Working.SessionSdkDir, "env-state.json")
	state := env.NewConfigState(stateFile)
	if err := state.Load(); err != nil {
		logger.Debugf("Failed to load state, will recalculate: %v\n", err)
	}

	// 4. Check if config has changed
	changed, err := state.HasChanged(configPaths)
	if err != nil {
		logger.Debugf("Failed to check config changes, will recalculate: %v\n", err)
		changed = true // Assume changed on error
	}

	logger.Debugf("Config changed: %v, configPaths: %+v\n", changed, configPaths)

	// 5. Fast path: return cached output if no changes
	if !changed {
		cachedOutput := state.GetCachedOutput()
		if cachedOutput != "" {
			logger.Debugf("Using cached output")
			fmt.Print(cachedOutput)
			return nil
		}
	}

	// 6. Slow path: recalculate env (full computation)
	// Process each SDK concurrently (same logic as before)
	allTools := chain.GetAllTools()
	envsByScope := map[env.UseScope]*env.Envs{
		env.Project: env.NewEnvs(),
		env.Session: env.NewEnvs(),
		env.Global:  env.NewEnvs(),
	}
	var mu sync.Mutex

	// Process SDKs concurrently using errgroup
	g, _ := errgroup.WithContext(context.Background())

	for sdkName, version := range allTools {
		sdkName := sdkName // Capture loop variable
		version := version // Capture loop variable

		g.Go(func() error {
			// Lookup SDK
			sdkObj, err := manager.LookupSdk(sdkName)
			if err != nil {
				logger.Debugf("SDK %s not found: %v", sdkName, err)
				return nil // Continue processing other SDKs
			}

			sdkVersion := sdk.Version(version)

			// Get tool config with scope information (searches by priority)
			toolConfig, scope, ok := chain.GetToolConfig(sdkName)
			if !ok {
				return nil
			}

			// Determine the actual scope to use
			// - If the scope is Project but linking is not enabled, downgrade to Session
			// - Global always uses link (no change)
			// - Session always uses link (no change)
			actualScope := scope
			if env.Project == scope && sdk.IsUseUnLink(toolConfig.Attr) {
				actualScope = env.Session
			}

			// Create symlinks if needed (internal logic checks if symlink already exists)
			if err := sdkObj.CreateSymlinksForScope(sdkVersion, actualScope); err != nil {
				logger.Debugf("Failed to create symlinks for %s@%s (scope: %s): %v",
					sdkName, version, actualScope.String(), err)
				return nil // Continue processing other SDKs
			}

			// Get environment variables pointing to symlinks
			sdkEnvs, err := sdkObj.EnvKeysForScope(sdkVersion, actualScope)
			if err != nil {
				logger.Debugf("Failed to get env keys for %s@%s: %v", sdkName, version, err)
				return nil // Continue processing other SDKs
			}

			// Collect envs by scope to ensure proper PATH priority (thread-safe)
			mu.Lock()
			envsByScope[actualScope].Merge(sdkEnvs)
			mu.Unlock()

			return nil
		})
	}

	// Wait for all SDK processing to complete
	if err := g.Wait(); err != nil {
		return err
	}

	// 7. Merge envs by scope priority: Project > Session > Global
	// This ensures proper priority for both PATH (Project first) and Vars (Project overrides)
	finalEnvs := env.NewEnvs()
	scopePriority := []env.UseScope{env.Project, env.Session, env.Global}
	finalEnvs.MergeByScopePriority(envsByScope, scopePriority)

	// 8. Build final PATH with proper priority:
	// User-injected paths (e.g., virtualenv) > Project > Session > Global > Cleaned System PATH
	//
	// SplitSystemPaths separates:
	// - prefixPaths: paths appearing BEFORE first vfox path (user-injected, highest priority)
	// - cleanSystemPaths: remaining non-vfox paths (lowest priority)
	prefixPaths, cleanSystemPaths := runtimeEnvContext.SplitSystemPaths()

	// Build final path order: prefix > vfox > clean system
	// We need to prepend prefixPaths to maintain their highest priority
	newPaths := env.NewPaths(env.EmptyPaths)
	newPaths.Merge(prefixPaths)
	newPaths.Merge(finalEnvs.Paths)
	newPaths.Merge(cleanSystemPaths)
	finalEnvs.Paths = newPaths

	// 9. Export environment variables
	if len(finalEnvs.Variables) == 0 && len(finalEnvs.Paths.Slice()) == 0 {
		return nil
	}

	exportEnvs := make(env.Vars)
	for k, v := range finalEnvs.Variables {
		exportEnvs[k] = v
	}

	// Add PATH with proper priority
	pathStr := finalEnvs.Paths.String()
	exportEnvs["PATH"] = &pathStr

	exportStr := s.Export(exportEnvs)

	// 10. Update state with new output
	if err := state.Update(configPaths, exportStr); err != nil {
		logger.Debugf("Failed to update state: %v", err)
	}

	fmt.Print(exportStr)
	return nil
}
