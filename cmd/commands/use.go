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
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/version-fox/vfox/internal"

	"github.com/urfave/cli/v2"
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

func useCmd(ctx *cli.Context) error {
	sdkArg := ctx.Args().First()
	if len(sdkArg) == 0 {
		return fmt.Errorf("invalid parameter. format: <sdk-name>[@<version>]")
	}
	var (
		name    string
		version internal.Version
	)
	argArr := strings.Split(sdkArg, "@")
	if len(argArr) <= 1 {
		name = argArr[0]
		version = ""
	} else {
		name = argArr[0]
		version = internal.Version(argArr[1])
	}

	scope := internal.Session
	if ctx.IsSet("global") {
		scope = internal.Global
	} else if ctx.IsSet("project") {
		scope = internal.Project
	} else {
		scope = internal.Session
	}
	manager := internal.NewSdkManager()
	defer manager.Close()

	source, err := manager.LookupSdk(name)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", name, err)
	}

	if version == "" {
		list := source.List()
		var arr []string
		for _, version := range list {
			arr = append(arr, string(version))
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
		version = internal.Version(result)
	}

	return source.Use(version, scope)
}
