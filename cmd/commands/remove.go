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
	"context"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/util"
)

var Remove = &cli.Command{
	Name:  "remove",
	Usage: "Remove a plugin",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "Skip confirmation prompt",
		},
	},
	Action:   removeCmd,
	Category: CategoryPlugin,
}

func removeCmd(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	l := args.Len()
	if l < 1 {
		return cli.Exit("invalid arguments", 1)
	}
	yes := cmd.Bool("yes")

	manager := internal.NewSdkManager()
	defer manager.Close()
	pterm.Println("Removing this plugin will remove the installed sdk along with the plugin.")

	if !yes {
		if util.IsNonInteractiveTerminal() {
			return cli.Exit("Use the -y flag to skip confirmation in non-interactive environments", 1)
		}
		result, _ := pterm.DefaultInteractiveConfirm.
			WithTextStyle(&pterm.ThemeDefault.DefaultText).
			WithConfirmStyle(&pterm.ThemeDefault.DefaultText).
			WithRejectStyle(&pterm.ThemeDefault.DefaultText).
			WithDefaultText("Please confirm").
			Show()
		if !result {
			return cli.Exit("remove canceled", 1)
		}
	}

	return manager.Remove(args.First())
}
