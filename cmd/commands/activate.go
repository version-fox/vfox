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
	"fmt"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/sdk"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/shared/logger"
	"github.com/version-fox/vfox/internal/shell"
	"golang.org/x/sync/errgroup"
)

var Activate = &cli.Command{
	Name:     "activate",
	Hidden:   true,
	Action:   activateCmd,
	Category: CategorySDK,
}

func activateCmd(ctx context.Context, cmd *cli.Command) error {
	name := cmd.Args().First()
	if name == "" {
		return cli.Exit("shell name is required", 1)
	}
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	runtimeEnvContext := manager.RuntimeEnvContext

	// 1. Load Project and Global configs with scope information
	chain, err := runtimeEnvContext.LoadVfoxTomlChainByScopes(env.Global, env.Project)
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

	// 2. Process each SDK: check if link is needed, create symlinks if necessary
	// Collect envs by scope to ensure proper PATH priority: Project > Session > Global > System
	allTools := chain.GetAllTools()
	envsByScope := map[env.UseScope]*env.Envs{
		env.Project: env.NewEnvs(),
		env.Session: env.NewEnvs(),
		env.Global:  env.NewEnvs(),
	}
	var mu sync.Mutex

	// Process SDKs concurrently using errgroup
	g, _ := errgroup.WithContext(ctx)

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

	// 3. Merge envs by scope priority: Project > Session > Global
	// This ensures proper priority for both PATH (Project first) and Vars (Project overrides)
	finalEnvs := env.NewEnvs()
	scopePriority := []env.UseScope{env.Project, env.Session, env.Global}
	finalEnvs.MergeByScopePriority(envsByScope, scopePriority)

	// 3. Build final PATH with proper priority: Project > Session > Global > Cleaned System PATH
	// Get clean system PATH (removes all vfox-managed paths)
	cleanSystemPaths := runtimeEnvContext.CleanSystemPaths()

	// Merge in priority order: vfox paths (already sorted by scope) > clean system paths
	finalEnvs.Paths.Merge(cleanSystemPaths)

	// 4. Export environment variables
	// Note: This step must be the first.
	// the Paths will handle the path format of GitBash which is different from other shells.
	// So we need to set the env.HookFlag first to let the Paths know
	// which shell we are using.
	_ = os.Setenv(env.HookFlag, name)

	exportEnvs := make(env.Vars)
	for k, v := range finalEnvs.Variables {
		exportEnvs[k] = v
	}

	// Add PATH with proper priority
	pathStr := finalEnvs.Paths.String()
	exportEnvs["PATH"] = &pathStr

	// Export __VFOX_CURTMFPATH so that vfox env can use the same session directory
	// This ensures state cache works across multiple vfox env calls in the same session
	curTmpPath := runtimeEnvContext.PathMeta.Working.SessionSdkDir
	exportEnvs[pathmeta.HookCurTmpPath] = &curTmpPath

	logger.Debugf("Export envs: %+v", exportEnvs)

	vfoxPath := runtimeEnvContext.PathMeta.Executable
	vfoxPath = strings.Replace(vfoxPath, "\\", "/", -1)
	s := shell.NewShell(name)
	if s == nil {
		return fmt.Errorf("unknown target shell %s", name)
	}

	exportStr := s.Export(exportEnvs)
	str, err := s.Activate(
		shell.ActivateConfig{
			SelfPath:       vfoxPath,
			Args:           cmd.Args().Tail(),
			EnablePidCheck: env.IsMultiplexerEnvironment(),
		},
	)
	if err != nil {
		return err
	}
	hookTemplate, err := template.New("hook").Parse(str)
	if err != nil {
		return nil
	}
	tmpCtx := struct {
		SelfPath       string
		EnvContent     string
		EnablePidCheck bool
	}{
		SelfPath:       vfoxPath,
		EnvContent:     exportStr,
		EnablePidCheck: env.IsMultiplexerEnvironment(),
	}
	return hookTemplate.Execute(cmd.Writer, tmpCtx)
}
