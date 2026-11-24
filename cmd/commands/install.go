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
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/logger"
	"github.com/version-fox/vfox/internal/shared/printer"
	"github.com/version-fox/vfox/internal/shared/util"
)

var Install = &cli.Command{
	Name:    "install",
	Aliases: []string{"i"},
	Usage:   "Install a version of the target SDK",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Install all SDK versions recorded in .vfox.toml",
		},
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "Quick installation, skip interactive prompts",
		},
		&cli.BoolFlag{
			Name:  "reinstall",
			Usage: "Replace an installed version with a fresh copy, for example after the SDK modified itself in place",
		},
		&cli.BoolFlag{
			Name:    "use",
			Aliases: []string{"u"},
			Usage:   "Use the installed version in the current shell session",
		},
		&cli.StringFlag{
			Name:  "as",
			Usage: "Install the version as a second copy under this name, so that the same version can be installed more than once, for example python@3.12 --as 3.12-db",
		},
	},
	Action:   installCmd,
	Category: CategorySDK,
}

func installCmd(ctx context.Context, cmd *cli.Command) error {
	yes := cmd.Bool("yes")
	reinstall := cmd.Bool("reinstall")
	useAfterInstall := cmd.Bool("use")
	asName := cmd.String("as")
	named := cmd.IsSet("as")

	if cmd.Bool("all") {
		if reinstall {
			return cli.Exit("--reinstall cannot be combined with --all", 1)
		}
		if useAfterInstall {
			return cli.Exit("--use cannot be combined with --all", 1)
		}
		if named {
			return cli.Exit("--as cannot be combined with --all", 1)
		}
		return installAll(yes)
	}

	args := cmd.Args()
	if args.First() == "" {
		return cli.Exit("sdk name is required", 1)
	}

	if named {
		if reinstall {
			return cli.Exit("--as cannot be combined with --reinstall", 1)
		}
		if err := sdk.ValidCopyName(asName); err != nil {
			return cli.Exit(err.Error(), 1)
		}
		if args.Len() != 1 {
			return cli.Exit("--as names one installation, so give one sdk@version", 1)
		}
	}

	if useAfterInstall {
		seen := make(map[string]bool, args.Len())
		for i := 0; i < args.Len(); i++ {
			name := strings.ToLower(strings.Split(args.Get(i), "@")[0])
			if seen[name] {
				return cli.Exit(fmt.Sprintf("--use cannot switch two versions of %s, give one", name), 1)
			}
			seen[name] = true
		}
	}

	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	if named {
		// An alias of the same name would stop working once the copy is installed.
		sdkName := strings.ToLower(strings.Split(args.First(), "@")[0])
		if err := manager.CheckNotAlias(sdkName, sdk.Version(asName)); err != nil {
			return cli.Exit(err.Error(), 1)
		}
	}

	errorStore := util.NewErrorStore()
	switchFailed := false

	for i := 0; i < args.Len(); i++ {
		sdkArg := args.Get(i)
		argArr := strings.Split(sdkArg, "@")
		argsLen := len(argArr)

		if argsLen > 2 {
			errorStore.AddAndShow(sdkArg, fmt.Errorf("your input is invalid: %s", sdkArg))
		} else {
			var name string
			var typedVersion sdk.Version
			if argsLen == 2 {
				name = strings.ToLower(argArr[0])
				typedVersion = sdk.Version(argArr[1])
			} else {
				name = strings.ToLower(argArr[0])
			}

			sdkSource, err := manager.LookupSdkWithInstall(name, yes)

			if err != nil {
				errorStore.AddAndShow(name, err)
				continue
			}
			sdkMetadata := sdkSource.Metadata()

			// Remove 'v' prefix if present (e.g., "v20.0.0" -> "20.0.0"), unless a
			// version is installed under that very name, as a linked one can be.
			version := sdk.StripVersionPrefix(sdkSource, typedVersion)

			// Handle @latest tag
			if version == "latest" {
				availableVersions, err := sdkSource.Available([]string{})
				if err != nil {
					errorStore.AddAndShow(name, fmt.Errorf("failed to get available versions: %w", err))
					continue
				}
				if len(availableVersions) == 0 {
					errorStore.AddAndShow(name, fmt.Errorf("no available versions"))
					continue
				}
				// Use the first available version (should be the latest)
				latestVersion := availableVersions[0].Version
				pterm.Printf("Using latest version: %s\n", pterm.LightGreen(latestVersion))
				version = latestVersion
			}

			var resolvedVersion = manager.ResolveVersion(sdkMetadata.Name, version)
			logger.Debugf("resolved version: %s\n", resolvedVersion)
			if resolvedVersion == "" {
				if useAfterInstall {
					return cli.Exit(fmt.Sprintf("install --use needs a version for %s, for example %s@latest", name, name), 1)
				}
				if named {
					return cli.Exit(fmt.Sprintf("install --as needs a version for %s, for example %s@latest", name, name), 1)
				}
				if util.IsNonInteractiveTerminal() {
					return cli.Exit(fmt.Sprintf("install requires specifying a version for %s", name), 1)
				}
				showAvailable, _ := printer.Confirm(&pterm.DefaultInteractiveConfirm, fmt.Sprintf("No %s version provided, do you want to select a version to install?", pterm.Red(name)))
				if showAvailable {
					err := RunSearch(name, []string{})
					if err != nil {
						errorStore.AddAndShow(name, err)
					}
					continue
				}
			} else {
				var installed sdk.Version
				if named {
					installed, err = sdkSource.InstallAs(resolvedVersion, sdk.Version(asName))
				} else {
					installed, err = sdkSource.InstallResolved(resolvedVersion, reinstall)
				}
				if err != nil {
					errorStore.AddAndShow(name, err)
					continue
				}
				if useAfterInstall {
					if err = sdkSource.UseWithConfig(installed, env.Session, false); err != nil {
						switchFailed = true
						errorStore.AddAndShow(name, fmt.Errorf("installed, but switching failed: %w", err))
						continue
					}
				}
			}
		}
	}

	notes := errorStore.GetNotesSet()

	if notes.Len() == 1 && switchFailed {
		return fmt.Errorf("failed to switch %s", notes.Slice()[0])
	} else if notes.Len() == 1 {
		return fmt.Errorf("failed to install %s", notes.Slice()[0])
	} else if notes.Len() > 1 {
		return fmt.Errorf("failed to install some SDKs: %s", strings.Join(notes.Slice(), ", "))
	}

	return nil
}

var openInstallAllDevNull = func() (*os.File, error) {
	return os.OpenFile(os.DevNull, os.O_WRONLY, 0)
}

func installAll(autoConfirm bool) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	restoreNotInstalled := redirectStdoutToStderr()
	plugins, sdks, err := notInstalled(manager)
	restoreNotInstalled()
	if err != nil {
		return err
	}
	if len(plugins) == 0 && len(sdks) == 0 {
		fmt.Println("All plugins and SDKs are already installed")
		return nil
	}

	fmt.Println("Install the following plugins and SDKs:")
	printPlugin(plugins, nil)
	printSdk(sdks, nil)

	if !autoConfirm {
		if util.IsNonInteractiveTerminal() {
			return cli.Exit("Use the -y flag to automatically confirm installation in non-interactive environments", 1)
		}
		result, _ := printer.Confirm(pterm.DefaultInteractiveConfirm.
			WithDefaultValue(true), "Do you want to install these plugins and SDKs?")
		if !result {
			return nil
		}
	}

	var (
		count         = len(plugins) + len(sdks)
		index         = 0
		errorStr      string
		stdout        = os.Stdout
		pluginsResult = make(map[string]bool)
		sdksResult    = make(map[string]bool)
	)
	devNull, devErr := openInstallAllDevNull()
	if devErr != nil {
		return fmt.Errorf("open %s: %w", os.DevNull, devErr)
	}
	// Route plugin stdout (top-level print/io.write during Add/LookupSdk and
	// os.execute) to stderr while keeping pterm spinners/notes on os.DevNull.
	os.Stdout = os.Stderr
	pterm.SetDefaultOutput(devNull)
	defer func() {
		os.Stdout = stdout
		pterm.SetDefaultOutput(stdout)
		// Best-effort close on the os.DevNull write handle opened only for batch stdout suppression.
		_ = devNull.Close()
	}()

	for _, plugin := range plugins {
		index++
		fmt.Fprintf(stdout, "[%v/%v] %s: %s installing...\n", index, count, "Plugin", plugin)
		pluginsResult[plugin] = false
		if err := manager.Add(plugin, "", ""); err != nil {
			if errors.Is(err, internal.ManifestNotFound) {
				errorStr = fmt.Sprintf("%s\n[%s] not found in remote registry, please check the name", errorStr, plugin)
			} else {
				errorStr = fmt.Sprintf("%s\n%s", errorStr, err)
			}
			continue
		}
		pluginsResult[plugin] = true
	}
	for _, sdkName := range slices.Sorted(maps.Keys(sdks)) {
		version := sdks[sdkName]
		index++
		fmt.Fprintf(stdout, "[%v/%v] %s: %s@%s installing...\n", index, count, "SDK", sdkName, version)
		sdkVersion := fmt.Sprintf("%s@%s", sdkName, version)
		sdksResult[sdkVersion] = false
		if ok, planned := pluginsResult[sdkName]; planned && !ok {
			continue
		}
		lookupSdk, err := manager.LookupSdk(sdkName)
		if err != nil {
			errorStr = fmt.Sprintf("%s\n%s", errorStr, err)
			continue
		}
		_, err = lookupSdk.InstallResolvedWithOptions(sdk.Version(version), sdk.InstallOptions{
			NoClipboard:    true,
			ProgressWriter: io.Discard,
			OutputWriter:   os.Stderr,
		})
		if err != nil {
			errorStr = fmt.Sprintf("%s\n%s", errorStr, err)
			continue
		}
		sdksResult[sdkVersion] = true
	}
	if len(errorStr) > 0 {
		fmt.Fprintf(stdout, "[%v/%v] Installation finished with failures.\n", count, count)
	} else {
		fmt.Fprintf(stdout, "[%v/%v] Installation completed.\n", count, count)
	}
	os.Stdout = stdout
	pterm.SetDefaultOutput(os.Stdout)

	fmt.Printf("%s indicates successful installation, while %s indicates installation failure.\n", pterm.Green("Green"), pterm.Red("red"))
	printPlugin(plugins, pluginsResult)
	printSdk(sdks, sdksResult)

	if len(errorStr) > 0 {
		return errors.New(strings.TrimPrefix(errorStr, "\n"))
	}
	return nil
}

func notInstalled(manager *internal.Manager) (plugins []string, sdks map[string]string, err error) {
	envContext := manager.RuntimeEnvContext

	// Load configs from all scopes with priority: Global < Session < Project
	chain, err := envContext.LoadVfoxTomlChainByScopes(env.Global, env.Session, env.Project)
	if err != nil {
		return
	}

	// Get all tools from the chain
	allTools := chain.GetAllTools()
	sdks = make(map[string]string)

	for name, version := range allTools {
		lookupSdk, err := manager.LookupSdk(name)
		if err != nil {
			// Plugin not installed yet: install the plugin first and the recorded SDK version afterwards.
			plugins = append(plugins, name)
			sdks[name] = version
		} else if !lookupSdk.CheckRuntimeExist(sdk.Version(version)) {
			// SDK not installed
			sdks[name] = version
		}
	}
	slices.Sort(plugins)
	return
}

func printPlugin(plugins []string, result map[string]bool) {
	if len(plugins) > 0 {
		fmt.Println("Plugin:")
		for _, plugin := range plugins {
			if result != nil {
				if result[plugin] {
					plugin = pterm.Green(plugin)
				} else {
					plugin = pterm.Red(plugin)
				}
			}

			fmt.Printf("  %s\n", plugin)
		}
	}
}

func printSdk(sdks map[string]string, result map[string]bool) {
	fmt.Println("SDK:")
	for _, sdk := range slices.Sorted(maps.Keys(sdks)) {
		version := sdks[sdk]
		sdkVersion := fmt.Sprintf("%s@%s", sdk, version)
		if result != nil {
			if result[sdkVersion] {
				sdkVersion = pterm.Green(sdkVersion)
			} else {
				sdkVersion = pterm.Red(sdkVersion)
			}
		}

		fmt.Printf("  %s\n", sdkVersion)
	}
}
