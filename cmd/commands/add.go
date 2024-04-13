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
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
)

var Add = &cli.Command{
	Name:  "add",
	Usage: "Add a plugin",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "source",
			Aliases: []string{"s"},
			Usage:   "plugin source",
		},
		&cli.StringFlag{
			Name:  "alias",
			Usage: "plugin alias",
		},
	},
	Action:   addCmd,
	Category: CategoryPlugin,
}

// addCmd is the command to add a plugin of sdk
func addCmd(ctx *cli.Context) error {
	manager := internal.NewSdkManager()
	defer manager.Close()
	sdkName := ctx.Args().First()
	source := ctx.String("source")
	alias := ctx.String("alias")
	return manager.Add(sdkName, source, alias)
}
