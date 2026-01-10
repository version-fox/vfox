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

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/util"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/shared/logger"

	"github.com/urfave/cli/v3"
)

var Use = &cli.Command{
	Name:    "use",
	Aliases: []string{"u"},
	Usage:   "Use a version of the target SDK",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "global",
			Aliases: []string{"g"},
			Usage:   "Used with the global environment",
		},
		&cli.BoolFlag{
			Name:    "project",
			Aliases: []string{"p"},
			Usage:   "Used with the current directory (default)",
		},
		&cli.BoolFlag{
			Name:    "session",
			Aliases: []string{"s"},
			Usage:   "Used with the current shell session",
		},
		&cli.BoolFlag{
			Name:  "unlink",
			Usage: "Do not create symlinks for project scope (downgrade to session scope)",
		},
	},
	Action:   useCmd,
	Category: CategorySDK,
}

func useCmd(ctx context.Context, cmd *cli.Command) error {
	sdkArg := cmd.Args().First()
	if len(sdkArg) == 0 {
		return fmt.Errorf("invalid parameter. format: <sdk-name>[@<version>]")
	}

	// Parse SDK name and version
	name, version := parseSdkArg(sdkArg)

	// Determine scope
	scope := determineScopeFromFlags(cmd)

	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	// Lookup SDK
	sdkSource, err := manager.LookupSdk(name)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", name, err)
	}

	// Resolve version (with interactive prompt if needed)
	resolvedVersion, err := resolveVersion(sdkSource, manager, version, name)
	if err != nil {
		return err
	}

	// Determine if should unlink (only valid for project scope)
	unlink := cmd.IsSet("unlink")

	// Execute use operation
	return sdkSource.UseWithConfig(resolvedVersion, scope, unlink)
}

// parseSdkArg parses the SDK argument in format "name@version"
func parseSdkArg(sdkArg string) (name string, version sdk.Version) {
	parts := strings.Split(sdkArg, "@")
	name = parts[0]
	if len(parts) > 1 {
		version = sdk.Version(parts[1])
	}
	return
}

// determineScopeFromFlags determines the scope based on command flags
func determineScopeFromFlags(cmd *cli.Command) env.UseScope {
	if cmd.IsSet("global") {
		return env.Global
	}
	if cmd.IsSet("session") {
		return env.Session
	}
	// Default to project if no scope specified
	return env.Project
}

// resolveVersion resolves the version, with interactive selection if needed
func resolveVersion(sdkSource sdk.Sdk, manager *internal.Manager, version sdk.Version, name string) (sdk.Version, error) {
	// Try to resolve version first
	resolvedVersion := manager.ResolveVersion(name, version)
	if resolvedVersion != "" {
		return resolvedVersion, nil
	}

	// If not resolved, try interactive selection
	availableVersions := sdkSource.InstalledList()
	if len(availableVersions) == 0 {
		return "", fmt.Errorf("no versions available for %s", name)
	}

	if util.IsNonInteractiveTerminal() {
		return "", cli.Exit("Please specify a version to use in non-interactive environments", 1)
	}

	// Convert versions to strings for selection
	versionStrings := make([]string, len(availableVersions))
	for i, v := range availableVersions {
		versionStrings[i] = string(v)
	}

	// Interactive selection
	selectedVersion, err := selectFromOptions(versionStrings, name)
	if err != nil {
		return "", err
	}

	return sdk.Version(selectedVersion), nil
}

// selectFromOptions displays an interactive selection prompt
func selectFromOptions(options []string, sdkName string) (string, error) {
	selectPrinter := pterm.InteractiveSelectPrinter{
		TextStyle:     &pterm.ThemeDefault.DefaultText,
		OptionStyle:   &pterm.ThemeDefault.DefaultText,
		Options:       options,
		DefaultOption: "",
		MaxHeight:     5,
		Selector:      "->",
		SelectorStyle: &pterm.ThemeDefault.SuccessMessageStyle,
		Filter:        true,
		OnInterruptFunc: func() {
			os.Exit(0)
		},
	}

	result, err := selectPrinter.Show(fmt.Sprintf("Please select a version of %s", sdkName))
	if err != nil {
		return "", fmt.Errorf("version selection failed: %w", err)
	}

	logger.Debugf("Selected version: %s for SDK: %s\n", result, sdkName)
	return result, nil
}
