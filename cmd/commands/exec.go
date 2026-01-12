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
	"os/exec"
	"strings"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/sdk"
)

var Exec = &cli.Command{
	Name:    "exec",
	Aliases: []string{"x"},
	Usage:   "Execute a command in vfox managed environment",
	Action:  execCmd,
}

func execCmd(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	if args.Len() < 2 {
		return fmt.Errorf("usage: vfox exec <sdk>[@<version>] <command> [args...]\nExample: vfox exec node@20 -- node -v")
	}

	// 1. Parse sdk@version (first argument)
	firstArg := args.First()
	parts := strings.Split(firstArg, "@")
	sdkName := parts[0]
	var sdkVersion sdk.Version
	if len(parts) > 1 {
		sdkVersion = sdk.Version(strings.TrimPrefix(parts[1], "v"))
	}

	// 2. Second argument is the command, rest are command arguments
	command := args.Get(1)
	cmdArgs := args.Slice()[2:]

	// 3. Execute the command
	return executeInVfoxEnv(sdkName, sdkVersion, command, cmdArgs)
}

// executeInVfoxEnv executes a command in vfox managed environment
func executeInVfoxEnv(sdkName string, sdkVersion sdk.Version, command string, cmdArgs []string) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return fmt.Errorf("failed to create sdk manager: %w", err)
	}
	defer manager.Close()

	// Lookup SDK
	sdkSource, err := manager.LookupSdk(sdkName)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", sdkName, err)
	}

	// If version is not specified, try to get it from current scope
	var resolvedVersion sdk.Version
	if sdkVersion == "" {
		// Get version from scope chain: Global > Session > Project
		chain, err := manager.RuntimeEnvContext.LoadVfoxTomlChainByScopes(
			env.Global, env.Session, env.Project,
		)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		version, _, ok := chain.GetToolVersion(sdkName)
		if !ok || version == "" {
			return fmt.Errorf("no version configured for %s. Please use 'vfox use' to set a version first", sdkName)
		}
		resolvedVersion = sdk.Version(version)
	} else {
		// Use the user-specified version
		resolvedVersion = sdkVersion

		// Check if installed, auto-install if not
		if !sdkSource.CheckRuntimeExist(resolvedVersion) {
			fmt.Printf("SDK %s@%s not found, installing...\n", sdkName, resolvedVersion)
			if err := sdkSource.Install(resolvedVersion); err != nil {
				return fmt.Errorf("failed to install %s@%s: %w", sdkName, resolvedVersion, err)
			}
		}
	}

	// Get environment variables
	// Note: Using EnvKeys to get the runtime package paths
	runtimePackage, err := sdkSource.GetRuntimePackage(resolvedVersion)
	if err != nil {
		return fmt.Errorf("failed to get runtime package for %s@%s: %w", sdkName, resolvedVersion, err)
	}
	envKeys, err := sdkSource.EnvKeys(runtimePackage)
	if err != nil {
		return fmt.Errorf("failed to get env keys for %s@%s: %w", sdkName, resolvedVersion, err)
	}

	// Build environment variable map
	envMap := make(map[string]string)

	// Add PATH from envKeys.Paths
	if envKeys.Paths != nil && envKeys.Paths.Slice() != nil {
		paths := envKeys.Paths.Slice()
		pathStr := strings.Join(paths, string(os.PathListSeparator))
		envMap["PATH"] = pathStr
	}

	// Add other variables from envKeys.Variables
	for key, value := range envKeys.Variables {
		if value != nil {
			envMap[key] = *value
		}
	}

	// Execute command
	return executeCommand(command, cmdArgs, envMap)
}

// executeCommand executes a command in the specified environment
func executeCommand(command string, args []string, envMap map[string]string) error {
	// Build environment variable array
	envVars := os.Environ()

	// First, clean vfox paths from PATH
	cleanedEnvVars := make([]string, 0, len(envVars))
	for _, env := range envVars {
		// Skip environment variables containing vfox sdk paths (checked via PathMeta.IsVfoxRelatedPath)
		if strings.HasPrefix(env, "PATH=") {
			// Keep system PATH, vfox will add correct PATH via envMap
			cleanedEnvVars = append(cleanedEnvVars, env)
		} else if !strings.HasPrefix(env, "VFOX_") {
			// Keep non-vfox related environment variables
			cleanedEnvVars = append(cleanedEnvVars, env)
		}
	}

	// Add vfox managed environment variables (note: PATH will override system PATH)
	for key, value := range envMap {
		cleanedEnvVars = append(cleanedEnvVars, fmt.Sprintf("%s=%s", key, value))
	}

	// Build temporary environment for finding executable
	tmpEnv := make(map[string]string)
	for _, env := range cleanedEnvVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			tmpEnv[parts[0]] = parts[1]
		}
	}

	// Find executable - first search in new PATH
	execPath, err := lookPathInEnv(command, tmpEnv["PATH"])
	if err != nil {
		return fmt.Errorf("command not found: %s: %w", command, err)
	}

	// Execute command
	execCmd := exec.Command(execPath, args...)
	execCmd.Env = cleanedEnvVars
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	// Start process and wait
	return execCmd.Run()
}

// lookPathInEnv searches for executable file in specified PATH environment variable
func lookPathInEnv(command, pathEnv string) (string, error) {
	if pathEnv == "" {
		return exec.LookPath(command)
	}

	// If command contains path separator, check directly
	if strings.Contains(command, "/") || strings.Contains(command, "\\") {
		return exec.LookPath(command)
	}

	// Search in PATH
	paths := strings.Split(pathEnv, string(os.PathListSeparator))
	for _, dir := range paths {
		if dir == "" {
			continue
		}
		fullPath := dir + string(os.PathSeparator) + command
		// Check if file is executable
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			// On Unix systems, check executable permissions
			// On Windows, .exe/.bat extensions are handled by exec.LookPath
			return fullPath, nil
		}
	}

	// If not found in custom PATH, fallback to exec.LookPath
	return exec.LookPath(command)
}
