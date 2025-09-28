/*
 *    Copyright 2025 Han Li and contributors
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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/base"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/toolset"
	"github.com/version-fox/vfox/internal/util"
)

var Install = &cli.Command{
	Name:    "install",
	Aliases: []string{"i"},
	Usage:   "Install a version of the target SDK",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "all",
			Aliases: []string{"a"},
			Usage:   "Install all SDK versions recorded in .tool-versions",
		},
	},
	Action:   installCmd,
	Category: CategorySDK,
}

func installCmd(ctx *cli.Context) error {
	if ctx.Bool("all") {
		return installAll()
	}

	args := ctx.Args()
	if args.First() == "" {
		return cli.Exit("sdk name is required", 1)
	}

	manager := internal.NewSdkManager()
	defer manager.Close()

	errorStore := util.NewErrorStore()

	for i := 0; i < args.Len(); i++ {
		sdkArg := args.Get(i)
		argArr := strings.Split(sdkArg, "@")
		argsLen := len(argArr)

		if argsLen > 2 {
			errorStore.AddAndShow(sdkArg, fmt.Errorf("your input is invalid: %s", sdkArg))
		} else {
			var name string
			var version base.Version
			if argsLen == 2 {
				name = strings.ToLower(argArr[0])
				version = base.Version(argArr[1])
			} else {
				name = strings.ToLower(argArr[0])
				version = ""
			}
			sdk, err := manager.LookupSdkWithInstall(name)
			if err != nil {
				errorStore.AddAndShow(name, err)
				continue
			}

			var resolvedVersion = manager.ResolveVersion(sdk.Name, version)
			logger.Debugf("resolved version: %s\n", resolvedVersion)
			if resolvedVersion == "" {
				showAvailable, _ := pterm.DefaultInteractiveConfirm.Show(fmt.Sprintf("No %s version provided, do you want to select a version to install?", pterm.Red(name)))
				if showAvailable {
					err := RunSearch(name, []string{})
					if err != nil {
						errorStore.AddAndShow(name, err)
					}
					continue
				}
			} else {
				if err = sdk.Install(resolvedVersion); err != nil {
					errorStore.AddAndShow(name, err)
					continue
				}
			}
		}
	}

	notes := errorStore.GetNotesSet()

	if notes.Len() == 1 {
		return fmt.Errorf("failed to install %s", notes.Slice()[0])
	} else if notes.Len() > 1 {
		return fmt.Errorf("failed to install some SDKs: %s", strings.Join(notes.Slice(), ", "))
	}

	return nil
}

func installAll() error {
	manager := internal.NewSdkManager()
	defer manager.Close()

	plugins, sdks, err := notInstalled(manager)
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
	if result, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultValue(true).
		Show("Do you want to install these plugins and SDKs?"); !result {
		return nil
	}

	var (
		count         = len(plugins) + len(sdks)
		index         = 0
		errorStr      string
		stdout        = os.Stdout
		stderr        = os.Stderr
		pluginsResult = make(map[string]bool)
		sdksResult    = make(map[string]bool)
	)
	os.Stdout = nil
	os.Stderr = nil
	pterm.SetDefaultOutput(os.Stdout)

	spinnerInfo, _ := pterm.DefaultSpinner.
		WithSequence([]string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "}...).
		WithText("Installing...").
		WithWriter(stdout).
		Start()
	for _, plugin := range plugins {
		index++
		spinnerInfo.UpdateText(fmt.Sprintf("[%v/%v] %s: %s installing...\033[K", index, count, "Plugin", plugin))
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
	for sdk, version := range sdks {
		index++
		spinnerInfo.UpdateText(fmt.Sprintf("[%v/%v] %s: %s@%s installing...\033[K", index, count, "SDK", sdk, version))
		sdkVersion := fmt.Sprintf("%s@%s", sdk, version)
		sdksResult[sdkVersion] = false
		lookupSdk, err := manager.LookupSdk(sdk)
		if err != nil {
			errorStr = fmt.Sprintf("%s\n%s", errorStr, err)
			continue
		}
		err = lookupSdk.Install(base.Version(version))
		if err != nil {
			errorStr = fmt.Sprintf("%s\n%s", errorStr, err)
			continue
		}
		sdksResult[sdkVersion] = true
	}
	spinnerInfo.UpdateText(fmt.Sprintf("[%v/%v] Installation completed.\033[K", count, count))
	_ = spinnerInfo.Stop()
	os.Stdout = stdout
	os.Stderr = stderr
	pterm.SetDefaultOutput(os.Stdout)

	fmt.Printf("%s indicates successful installation, while %s indicates installation failure.\n", pterm.Green("Green"), pterm.Red("red"))
	printPlugin(plugins, pluginsResult)
	printSdk(sdks, sdksResult)

	if len(errorStr) > 0 {
		fmt.Println(errorStr)
	}
	return nil
}

func notInstalled(manager *internal.Manager) (plugins []string, sdks map[string]string, err error) {
	tvs, err := toolset.NewMultiToolVersions([]string{
		manager.PathMeta.WorkingDirectory,
		manager.PathMeta.CurTmpPath,
		manager.PathMeta.HomePath,
	})
	if err != nil {
		return
	}
	sdks = tvs.FilterTools(func(name, version string) bool {
		lookupSdk, err := manager.LookupSdk(name)
		if err != nil {
			plugins = append(plugins, name)
			return true
		}
		if !lookupSdk.CheckExists(base.Version(version)) {
			return true
		}
		return false
	})
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
	for sdk, version := range sdks {
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
