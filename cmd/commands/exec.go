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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/logger"
)

type execSDKSpec struct {
	Name    string
	Version sdk.Version
}

type execOptions struct {
	ShowHelp bool
	Debug    bool
	DebugSet bool
}

const (
	execCommandName  = "exec"
	execCommandAlias = "x"
)

var Exec = &cli.Command{
	Name:            execCommandName,
	Aliases:         []string{execCommandAlias},
	Usage:           "Execute a command in vfox managed environment",
	ArgsUsage:       "<sdk>[@<version>]... -- <command> [args...] | <sdk>[@<version>] <command> [args...]",
	Description:     "The first '--' always separates SDK specs from the command. If <command> itself takes '--', pass '--' before <command>, for example: vfox exec nodejs@24.14.0 -- npm run build -- --watch",
	SkipFlagParsing: true,
	Action:          execCmd,
}

func execCmd(ctx context.Context, cmd *cli.Command) error {
	rawArgs := restoreExecTrailingCompletionArg(cmd, cmd.Args().Slice())
	args, opts, err := parseExecLeadingArgs(rawArgs)
	if err != nil {
		return err
	}
	if opts.DebugSet {
		if opts.Debug {
			logger.SetLevel(logger.DebugLevel)
		} else {
			logger.SetLevel(logger.InfoLevel)
		}
	}
	if opts.ShowHelp {
		return showExecHelp(cmd)
	}
	sdkSpecs, command, cmdArgs, err := parseExecInvocation(args)
	if err != nil {
		return err
	}
	hasSeparator := slices.Index(args, "--") >= 0
	return executeInVfoxEnvWithSeparator(sdkSpecs, command, cmdArgs, hasSeparator)
}

func restoreExecTrailingCompletionArg(cmd *cli.Command, args []string) []string {
	root := cmd.Root()
	if root == nil || !root.EnableShellCompletion || !slices.Contains(args, "--") {
		return args
	}
	raw := os.Args
	if root.Metadata != nil {
		if metaArgs, ok := root.Metadata["rawArgs"].([]string); ok && len(metaArgs) > 0 {
			raw = metaArgs
		}
	}
	const completionFlag = "--generate-shell-completion"
	if len(raw) > 0 && raw[len(raw)-1] == completionFlag {
		return append(slices.Clone(args), completionFlag)
	}
	return args
}

func showExecHelp(cmd *cli.Command) error {
	tmpl := cmd.CustomHelpTemplate
	if tmpl == "" {
		tmpl = cli.CommandHelpTemplate
	}
	cli.HelpPrinter(cmd.Root().Writer, tmpl, cmd)
	return nil
}

func parseExecDebugFlag(arg string) (bool, bool) {
	switch arg {
	case "--debug", "-debug":
		return true, true
	}
	var raw string
	switch {
	case strings.HasPrefix(arg, "--debug="):
		raw = strings.TrimPrefix(arg, "--debug=")
	case strings.HasPrefix(arg, "-debug="):
		raw = strings.TrimPrefix(arg, "-debug=")
	default:
		return false, false
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false
	}
	return parsed, true
}

func parseExecLeadingArgs(args []string) ([]string, execOptions, error) {
	var opts execOptions
	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") || arg == "-" || arg == "--" {
			return args[i:], opts, nil
		}
		switch arg {
		case "--help", "-h":
			opts.ShowHelp = true
		default:
			if debug, ok := parseExecDebugFlag(arg); ok {
				opts.Debug = debug
				opts.DebugSet = true
				continue
			}
			return nil, opts, fmt.Errorf("unexpected flag %q before SDK\n%s", arg, execUsageLine())
		}
	}
	return nil, opts, nil
}

func parseExecInvocation(args []string) ([]execSDKSpec, string, []string, error) {
	if len(args) < 2 {
		return nil, "", nil, execUsageError()
	}

	if separatorIndex := slices.Index(args, "--"); separatorIndex >= 0 {
		sdkArgs := args[:separatorIndex]
		commandArgs := args[separatorIndex+1:]
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

	if len(args) > 2 && looksLikeExecSDKSpec(args[1]) {
		return nil, "", nil, fmt.Errorf("multiple SDKs require '--' before the command\n%s", execUsageLine())
	}

	spec, err := parseExecSDKSpec(args[0])
	if err != nil {
		return nil, "", nil, err
	}
	if args[1] != "-" && strings.HasPrefix(args[1], "-") {
		return nil, "", nil, fmt.Errorf("unexpected flag %q instead of command\n%s", args[1], execUsageLine())
	}
	return []execSDKSpec{spec}, args[1], args[2:], nil
}

func looksLikeExecSDKSpec(arg string) bool {
	if strings.HasPrefix(arg, "-") || strings.ContainsAny(arg, "/\\") {
		return false
	}
	name, _, hasAt := strings.Cut(arg, "@")
	return hasAt && strings.TrimSpace(name) != ""
}

func parseExecSDKSpec(arg string) (execSDKSpec, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return execSDKSpec{}, execUsageError()
	}
	if strings.HasPrefix(arg, "-") {
		return execSDKSpec{}, fmt.Errorf("unexpected flag %q in SDK spec\n%s", arg, execUsageLine())
	}

	parts := strings.SplitN(arg, "@", 2)
	spec := execSDKSpec{Name: parts[0]}
	if spec.Name == "" {
		return execSDKSpec{}, fmt.Errorf("invalid SDK spec %q", arg)
	}
	if len(parts) == 2 {
		// As typed: see Manager.ResolveVersionName for what happens to a leading 'v'.
		spec.Version = sdk.Version(parts[1])
	}
	return spec, nil
}

func execUsageLine() string {
	return "usage: vfox exec <sdk>[@<version>]... -- <command> [args...]\n" +
		"       vfox exec <sdk>[@<version>] <command> [args...]\n" +
		"The first '--' always separates SDK specs from the command. If <command> itself takes '--', pass '--' before <command> (for example: vfox exec nodejs@24.14.0 -- npm run build -- --watch).\n" +
		"Example: vfox exec nodejs@24.14.0 golang@1.25.6 -- npm install -g pnpm"
}

func execUsageError() error {
	return fmt.Errorf("%s", execUsageLine())
}

type execCommandNotFoundError struct {
	command string
	err     error
}

func (e *execCommandNotFoundError) Error() string {
	return fmt.Sprintf("command not found: %s: %v", e.command, e.err)
}

func (e *execCommandNotFoundError) Unwrap() error {
	return e.err
}

func hasDoubleDashOptionAfterSeparator(command string, cmdArgs []string) bool {
	if strings.HasPrefix(command, "--") {
		return true
	}
	return slices.ContainsFunc(cmdArgs, func(arg string) bool {
		return strings.HasPrefix(arg, "--")
	})
}

func executeInVfoxEnvWithSeparator(sdkSpecs []execSDKSpec, command string, cmdArgs []string, hasSeparator bool) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return fmt.Errorf("failed to create sdk manager: %w", err)
	}
	defer manager.Close()

	sdkEnvs := make([]*env.Envs, 0, len(sdkSpecs))
	for i, sdkSpec := range sdkSpecs {
		specEnvs, err := resolveExecSDKEnv(manager, sdkSpec)
		if err != nil {
			if len(sdkSpecs) > 1 && i > 0 && sdkSpec.Version == "" && hasDoubleDashOptionAfterSeparator(command, cmdArgs) {
				return fmt.Errorf("%w\nnote: arguments before the first '--' (including %q) are parsed as SDKs; pass '--' before the command when it also uses '--'", err, sdkSpec.Name)
			}
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
	envMap["PATH"] = buildExecPath(mergedEnvs.Paths)
	err = executeCommand(command, cmdArgs, envMap)
	var notFoundErr *execCommandNotFoundError
	if err != nil && len(sdkSpecs) > 0 && errors.As(err, &notFoundErr) {
		if hasSeparator && len(sdkSpecs) > 1 && strings.HasPrefix(command, "-") {
			return fmt.Errorf("%w\nnote: arguments before the first '--' (including %q) are parsed as SDKs; pass '--' before the command", err, sdkSpecs[len(sdkSpecs)-1].Name)
		}
		if !hasSeparator && len(cmdArgs) > 0 && !strings.HasPrefix(cmdArgs[0], "-") {
			return fmt.Errorf("%w\nnote: without '--', %q is parsed as a single SDK and %q as the command; pass '--' before the command to use multiple SDKs", err, sdkSpecs[0].Name, command)
		}
	}
	return err
}

// buildExecPath builds the PATH passed to the process that exec launches.
// Paths.String() is shell-facing and uses a POSIX ":" separator in bash/zsh
// sessions on Windows, which a native child process cannot split (it uses ";").
// Join with the platform separator so exec can locate SDK tools in every shell.
func buildExecPath(paths *env.Paths) string {
	return strings.Join(paths.Slice(), string(os.PathListSeparator))
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
		installedVersion, err := sdkSource.InstallResolved(resolvedVersion, false)
		if err != nil {
			return nil, fmt.Errorf("failed to install %s@%s: %w", sdkSpec.Name, resolvedVersion, err)
		}
		resolvedVersion = installedVersion
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
		source, _ := manager.LookupSdk(sdkSpec.Name)
		version := manager.ResolveVersionName(sdkSpec.Name, source, sdkSpec.Version)
		if source == nil || source.CheckRuntimeExist(version) {
			return version, nil
		}
		// A partial version stands for the installed version it extends (4.5 for an
		// installed 4.5.0), as it does for `vfox use`, rather than for a version to
		// install. Only a version nothing installed matches is left to be installed.
		installed, err := source.ResolveInstalledVersion(version)
		if err != nil {
			return "", fmt.Errorf("failed to resolve %s@%s: %w", sdkSpec.Name, version, err)
		}
		if source.CheckRuntimeExist(installed) {
			return installed, nil
		}
		return version, nil
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
	pathExtEnv := tmpEnv["PATHEXT"]
	if pathExtEnv == "" {
		pathExtEnv = os.Getenv("PATHEXT")
	}
	execPath, err := lookPathInEnvWithExt(command, tmpEnv["PATH"], pathExtEnv)
	if err != nil {
		return &execCommandNotFoundError{command: command, err: err}
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
	return lookPathInEnvWithExt(command, pathEnv, os.Getenv("PATHEXT"))
}

func lookPathInEnvWithExt(command, pathEnv, pathExtEnv string) (string, error) {
	if pathEnv == "" {
		return exec.LookPath(command)
	}

	// If command contains path separator, check directly
	if strings.Contains(command, "/") || strings.Contains(command, "\\") {
		if runtime.GOOS == "windows" {
			command = filepath.FromSlash(command)
		}
		return exec.LookPath(command)
	}

	// Search in PATH
	paths := filepath.SplitList(pathEnv)
	var exts []string
	if runtime.GOOS == "windows" {
		exts = windowsPathExts(pathExtEnv)
	}
	for _, dir := range paths {
		if dir == "" {
			continue
		}
		if runtime.GOOS == "windows" {
			if found, ok := findWindowsExecutable(dir, command, exts); ok {
				return found, nil
			}
			continue
		}
		fullPath := filepath.Join(dir, command)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			return fullPath, nil
		}
	}

	// If not found in custom PATH, fallback to exec.LookPath
	return exec.LookPath(command)
}

func windowsPathExts(pathExtEnv string) []string {
	if pathExtEnv == "" {
		return []string{".COM", ".EXE", ".BAT", ".CMD"}
	}
	var exts []string
	for _, ext := range strings.Split(pathExtEnv, ";") {
		if ext == "" {
			continue
		}
		if ext[0] != '.' {
			ext = "." + ext
		}
		exts = append(exts, ext)
	}
	if len(exts) == 0 {
		return []string{".COM", ".EXE", ".BAT", ".CMD"}
	}
	return exts
}

func findWindowsExecutable(dir, command string, exts []string) (string, bool) {
	if filepath.Ext(command) != "" {
		fullPath := filepath.Join(dir, command)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			return fullPath, true
		}
	}
	for _, ext := range exts {
		fullPath := filepath.Join(dir, command+ext)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			return fullPath, true
		}
	}
	return "", false
}
