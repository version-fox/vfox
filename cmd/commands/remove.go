/*
 *    Copyright 2024 Han Li and contributors
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
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
)

var Remove = &cli.Command{
	Name:   "remove",
	Usage:  "remove a plugin of sdk",
	Action: removeCmd,
}

func removeCmd(ctx *cli.Context) error {
	args := ctx.Args()
	l := args.Len()
	if l < 1 {
		return cli.Exit("invalid arguments", 1)
	}
	manager := internal.NewSdkManager()
	defer manager.Close()
	pterm.Println("Removing this plugin will remove the installed sdk along with the plugin.")
	result, _ := pterm.DefaultInteractiveConfirm.
		WithTextStyle(&pterm.ThemeDefault.DefaultText).
		WithConfirmStyle(&pterm.ThemeDefault.DefaultText).
		WithRejectStyle(&pterm.ThemeDefault.DefaultText).
		WithDefaultText("Please confirm").
		Show()
	if result {
		return manager.Remove(args.First())
	} else {
		return cli.Exit("remove canceled", 1)
	}
}
