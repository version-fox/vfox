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

	"github.com/version-fox/vfox/internal/util"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/base"

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
			Usage:   "Used with the current directory",
		},
		&cli.BoolFlag{
			Name:    "session",
			Aliases: []string{"s"},
			Usage:   "Used with the current shell session",
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
	var (
		name    string
		version base.Version
	)
	argArr := strings.Split(sdkArg, "@")
	if len(argArr) <= 1 {
		name = argArr[0]
		version = ""
	} else {
		name = argArr[0]
		version = base.Version(argArr[1])
	}

	scope := base.Session
	if cmd.IsSet("global") {
		scope = base.Global
	} else if cmd.IsSet("project") {
		scope = base.Project
	} else {
		scope = base.Session
	}
	manager := internal.NewSdkManager()
	defer manager.Close()

	source, err := manager.LookupSdk(name)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", name, err)
	}

	var resolvedVersion = manager.ResolveVersion(name, version)
	if resolvedVersion == "" {
		list := source.List()
		var arr []string
		for _, version := range list {
			arr = append(arr, string(version))
		}
		if len(arr) == 0 {
			return fmt.Errorf("no versions available for %s", name)
		}
		if util.IsNonInteractiveTerminal() {
			return cli.Exit("Please specify a version to use in non-interactive environments", 1)
		}
		selectPrinter := pterm.InteractiveSelectPrinter{
			TextStyle:     &pterm.ThemeDefault.DefaultText,
			OptionStyle:   &pterm.ThemeDefault.DefaultText,
			Options:       arr,
			DefaultOption: "",
			MaxHeight:     5,
			Selector:      "->",
			SelectorStyle: &pterm.ThemeDefault.SuccessMessageStyle,
			Filter:        true,
			OnInterruptFunc: func() {
				os.Exit(0)
			},
		}
		result, _ := selectPrinter.Show(fmt.Sprintf("Please select a version of %s", name))
		resolvedVersion = base.Version(result)
	}

	return source.Use(resolvedVersion, scope)
}
