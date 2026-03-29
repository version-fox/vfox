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
	"slices"
	"strings"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/sdk"
)

type execSDKSpec struct {
	Name    string
	Version sdk.Version
}

const (
	execCommandName  = "exec"
	execCommandAlias = "x"
)

var Exec = &cli.Command{
	Name:    execCommandName,
	Aliases: []string{execCommandAlias},
	Usage:   "Execute a command in vfox managed environment",
	Action:  execCmd,
}

func execCmd(ctx context.Context, cmd *cli.Command) error {
	sdkSpecs, command, cmdArgs, err := parseExecInvocation(cmd.Args().Slice(), os.Args)
	if err != nil {
		return err
	}
	return executeInVfoxEnv(sdkSpecs, command, cmdArgs)
}

func parseExecInvocation(parsedArgs, rawArgs []string) ([]execSDKSpec, string, []string, error) {
	if len(parsedArgs) < 2 {
		return nil, "", nil, execUsageError()
	}

	rawExecArgs := rawExecArgs(rawArgs)
	if separatorIndex := slices.Index(rawExecArgs, "--"); separatorIndex >= 0 {
		sdkArgs := rawExecArgs[:separatorIndex]
		commandArgs := rawExecArgs[separatorIndex+1:]
		if len(sdkArgs) == 0 || len(commandArgs) == 0 {
			return nil, "", nil, execUsageError()
		}

		sdkSpecs := make([]execSDKSpec, 0, len(sdkArgs))
		for _, sdkArg := range sdkArgs {
			spec, err := parseExecSDKSpec(sdkArg)
			if err != nil {
				return nil, "", nil, err
			}
			sdkSpecs = append(sdkSpecs, spec)
		}
		return sdkSpecs, commandArgs[0], commandArgs[1:], nil
	}

	if len(parsedArgs) > 2 && strings.Contains(parsedArgs[1], "@") {
		return nil, "", nil, fmt.Errorf("multiple SDKs require '--' before the command\n%s", execUsageLine())
	}

	spec, err := parseExecSDKSpec(parsedArgs[0])
	if err != nil {
		return nil, "", nil, err
	}
	return []execSDKSpec{spec}, parsedArgs[1], parsedArgs[2:], nil
}

func rawExecArgs(rawArgs []string) []string {
	for i := 1; i < len(rawArgs); i++ {
		if rawArgs[i] == execCommandName || rawArgs[i] == execCommandAlias {
			return rawArgs[i+1:]
		}
	}
	return nil
}

func parseExecSDKSpec(arg string) (execSDKSpec, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return execSDKSpec{}, execUsageError()
	}

	parts := strings.SplitN(arg, "@", 2)
	spec := execSDKSpec{Name: parts[0]}
	if spec.Name == "" {
		return execSDKSpec{}, fmt.Errorf("invalid SDK spec %q", arg)
	}
	if len(parts) == 2 {
		spec.Version = sdk.Version(strings.TrimPrefix(parts[1], "v"))
	}
	return spec, nil
}

func execUsageLine() string {
	return "usage: vfox exec <sdk>[@<version>]... -- <command> [args...]\nExample: vfox exec nodejs@24.14.0 golang@1.25.6 -- npm install -g pnpm"
}

func execUsageError() error {
	return fmt.Errorf("%s", execUsageLine())
}

// executeInVfoxEnv executes a command in vfox managed environment
func executeInVfoxEnv(sdkSpecs []execSDKSpec, command string, cmdArgs []string) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return fmt.Errorf("failed to create sdk manager: %w", err)
	}
	defer manager.Close()

	sdkEnvs := make([]*env.Envs, 0, len(sdkSpecs))
	for _, sdkSpec := range sdkSpecs {
		specEnvs, err := resolveExecSDKEnv(manager, sdkSpec)
		if err != nil {
			return err
		}
		sdkEnvs = append(sdkEnvs, specEnvs)
	}

	mergedEnvs := mergeExecEnvsByPriority(sdkEnvs)
	applyExecSystemPaths(manager.RuntimeEnvContext, mergedEnvs)

	envMap := make(map[string]string, len(mergedEnvs.Variables)+1)
	for key, value := range mergedEnvs.Variables {
		if value != nil {
			envMap[key] = *value
		}
	}
	envMap["PATH"] = mergedEnvs.Paths.String()
	return executeCommand(command, cmdArgs, envMap)
}

func resolveExecSDKEnv(manager *internal.Manager, sdkSpec execSDKSpec) (*env.Envs, error) {
	sdkSource, err := manager.LookupSdk(sdkSpec.Name)
	if err != nil {
		return nil, fmt.Errorf("%s not supported, error: %w", sdkSpec.Name, err)
	}

	resolvedVersion, err := resolveExecSDKVersion(manager, sdkSpec)
	if err != nil {
		return nil, err
	}
	if sdkSpec.Version != "" && !sdkSource.CheckRuntimeExist(resolvedVersion) {
		fmt.Printf("SDK %s@%s not found, installing...\n", sdkSpec.Name, resolvedVersion)
		if err := sdkSource.Install(resolvedVersion); err != nil {
			return nil, fmt.Errorf("failed to install %s@%s: %w", sdkSpec.Name, resolvedVersion, err)
		}
	}

	runtimePackage, err := sdkSource.GetRuntimePackage(resolvedVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get runtime package for %s@%s: %w", sdkSpec.Name, resolvedVersion, err)
	}

	envKeys, err := sdkSource.EnvKeys(runtimePackage)
	if err != nil {
		return nil, fmt.Errorf("failed to get env keys for %s@%s: %w", sdkSpec.Name, resolvedVersion, err)
	}
	return envKeys, nil
}

func resolveExecSDKVersion(manager *internal.Manager, sdkSpec execSDKSpec) (sdk.Version, error) {
	if sdkSpec.Version != "" {
		return sdkSpec.Version, nil
	}

	chain, err := manager.RuntimeEnvContext.LoadVfoxTomlChainByScopes(env.Global, env.Session, env.Project)
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	version, _, ok := chain.GetToolVersion(sdkSpec.Name)
	if !ok || version == "" {
		return "", fmt.Errorf("no version configured for %s. Please use 'vfox use' to set a version first", sdkSpec.Name)
	}
	return sdk.Version(version), nil
}

func mergeExecEnvsByPriority(envsByPriority []*env.Envs) *env.Envs {
	merged := env.NewEnvs()
	for _, sdkEnvs := range envsByPriority {
		if sdkEnvs == nil {
			continue
		}
		merged.Paths.Merge(sdkEnvs.Paths)
	}
	for i := len(envsByPriority) - 1; i >= 0; i-- {
		if sdkEnvs := envsByPriority[i]; sdkEnvs != nil {
			merged.Variables.Merge(sdkEnvs.Variables)
		}
	}
	return merged
}

func applyExecSystemPaths(runtimeEnvContext *env.RuntimeEnvContext, sdkEnvs *env.Envs) {
	if sdkEnvs == nil {
		return
	}
	prefixPaths, cleanSystemPaths := runtimeEnvContext.SplitSystemPaths()
	sdkEnvs.Paths.Merge(prefixPaths)
	sdkEnvs.Paths.Merge(cleanSystemPaths)
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
