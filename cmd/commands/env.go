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
	"sort"
	"strings"
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

	// 1. Load configs from all scopes with priority: Global < Session < Project,
	// and the legacy version files of the project
	chain, err := loadToolChain(manager, env.Global, env.Session, env.Project)
	if err != nil {
		return err
	}

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
	// Output rendered under another PATH ordering policy is stale. `path.mode`
	// and `path.preserve` are in config.yaml, which configPaths does not track.
	pathPolicy := runtimeEnvContext.PathPolicyKey()
	if state.PathPolicyChanged(pathPolicy) {
		changed = true
	}
	// What the tools resolve to also depends on files configPaths does not
	// track: legacy version files (.nvmrc and the like) read into the project
	// scope, the project SDK directory, the installed versions and plugins, and
	// the links the output points at. All of them go into the resolution key.
	projectKey := projectToolsKey(chain, runtimeEnvContext.PathMeta.Working.ProjectSdkDir)
	installs := installsStamp(runtimeEnvContext, chain)
	if state.ResolutionKeyChanged(resolutionKey(projectKey, installs, linksStamp(runtimeEnvContext, chain))) {
		changed = true
	}

	logger.Debugf("Config changed: %v, configPaths: %+v\n", changed, configPaths)

	// 5. Fast path: return cached output if no changes
	if !changed {
		cachedOutput := state.GetCachedOutput(shellName)
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

	type sdkScopedEnvs struct {
		scope env.UseScope
		envs  *env.Envs
	}
	results := make(map[string]*sdkScopedEnvs, len(allTools))

	// Process SDKs concurrently using errgroup
	g, _ := errgroup.WithContext(context.Background())

	for sdkName := range allTools {
		sdkName := sdkName // Capture loop variable

		g.Go(func() error {
			// Lookup SDK
			sdkObj, err := manager.LookupSdk(sdkName)
			if err != nil {
				logger.Debugf("SDK %s not found: %v", sdkName, err)
				return nil // Continue processing other SDKs
			}

			// Resolve to the highest-priority installed version across scopes
			toolConfig, scope, sdkVersion, ok := resolveInstalledToolConfig(chain, sdkObj, sdkName)
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
					sdkName, sdkVersion, actualScope.String(), err)
				return nil // Continue processing other SDKs
			}

			// Get environment variables pointing to symlinks
			sdkEnvs, err := sdkObj.EnvKeysForScope(sdkVersion, actualScope)
			if err != nil {
				logger.Debugf("Failed to get env keys for %s@%s: %v", sdkName, sdkVersion, err)
				return nil // Continue processing other SDKs
			}

			// Collect the result per SDK; merging happens after g.Wait()
			// in a deterministic order (thread-safe)
			mu.Lock()
			results[sdkName] = &sdkScopedEnvs{scope: actualScope, envs: sdkEnvs}
			mu.Unlock()

			return nil
		})
	}

	// Wait for all SDK processing to complete
	if err := g.Wait(); err != nil {
		return err
	}

	// Merge per-SDK envs in deterministic (sorted-by-name) order.
	// Goroutine completion order is scheduling-dependent: merging inside the
	// goroutines reshuffles same-scope PATH entries between rebuilds, so the
	// emitted PATH almost never equals the PATH recorded in env-state.json
	// (cached_path), permanently invalidating the cache once the caller
	// eval's the output. Scope priority is untouched — only the ordering of
	// same-scope entries (previously random) is fixed.
	names := make([]string, 0, len(results))
	for name := range results {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		envsByScope[results[name].scope].Merge(results[name].envs)
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
	state.SetPathPolicy(pathPolicy)
	// The slow path made the links again, so take their stamp after it.
	state.SetResolutionKey(resolutionKey(projectKey, installs, linksStamp(runtimeEnvContext, chain)))
	if err := state.Update(configPaths, exportStr, shellName); err != nil {
		logger.Debugf("Failed to update state: %v", err)
	}

	fmt.Print(exportStr)
	return nil
}

// projectToolsKey describes the tools of the project scope, including those
// read from legacy version files, and the project SDK directory they are
// exported through, so that `env` can tell when either changes. Without
// project tools the key is empty, so a cd between plain directories still
// hits the cache.
func projectToolsKey(chain env.VfoxTomlChain, projectSdkDir string) string {
	projectToml, ok := chain.GetTomlByScope(env.Project)
	if !ok || projectToml == nil || len(projectToml.Tools) == 0 {
		return ""
	}
	tools := projectToml.Tools
	lines := []string{projectSdkDir}
	for _, name := range tools.SortedKeys() {
		lines = append(lines, name+" = "+tools[name].MarshalInline())
	}
	return strings.Join(lines, "\n")
}

// resolutionKey joins the parts of the key the cached output is stored under.
func resolutionKey(projectKey, installs, links string) string {
	return projectKey + "\n--\n" + installs + "\n--\n" + links
}

// stampNames returns the lower-cased names of all configured tools, sorted.
func stampNames(chain env.VfoxTomlChain) []string {
	tools := chain.GetAllTools()
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, strings.ToLower(name))
	}
	sort.Strings(names)
	return names
}

// installsStamp describes what is installed for each configured tool: whether
// its plugin is added and the modification time of its install directory,
// which moves when a version is installed or removed. `vfox install` and
// `vfox add` change these without touching a tracked file. The plugin counts
// by presence only, since a plugin may write into its own directory.
func installsStamp(ctx *env.RuntimeEnvContext, chain env.VfoxTomlChain) string {
	shared := ctx.PathMeta.Shared
	var b strings.Builder
	for _, name := range stampNames(chain) {
		plugin := "-"
		if _, err := os.Stat(filepath.Join(shared.Plugins, name)); err == nil {
			plugin = "+"
		}
		installed := "-"
		// The same fallback as sdk.NewSdk.
		for _, dir := range []string{filepath.Join(shared.Installs, name), filepath.Join(shared.Installs, "cache", name)} {
			if info, err := os.Stat(dir); err == nil {
				installed = fmt.Sprint(info.ModTime().UnixNano())
				break
			}
		}
		fmt.Fprintf(&b, "%s %s %s\n", name, plugin, installed)
	}
	return b.String()
}

// linksStamp describes which of the links the output can point at exist, in
// the project, session and global link directories. A cache hit does not make
// them again, so a removed .vfox directory must count as a change.
func linksStamp(ctx *env.RuntimeEnvContext, chain env.VfoxTomlChain) string {
	scopes := []env.UseScope{env.Project, env.Session, env.Global}
	var b strings.Builder
	for _, name := range stampNames(chain) {
		b.WriteString(name)
		for _, scope := range scopes {
			mark := " -"
			if _, err := os.Lstat(filepath.Join(ctx.GetLinkDirPathByScope(scope), name)); err == nil {
				mark = " +"
			}
			b.WriteString(mark)
		}
		b.WriteString("\n")
	}
	return b.String()
}
