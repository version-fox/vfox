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

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
)

const allFlag = "all"

var Update = &cli.Command{
	Name:  "update",
	Usage: "Update specified plugin, use --all/-a to update all installed plugins",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    allFlag,
			Aliases: []string{"a"},
			Usage:   "all plugins flag",
		},
	},
	Action:   updateCmd,
	Category: CategoryPlugin,
}

func updateCmd(ctx *cli.Context) error {
	manager := internal.NewSdkManager()
	defer manager.Close()
	if ctx.Bool(allFlag) {
		if sdks, err := manager.LoadAllSdk(); err == nil {
			var (
				index int
				total = len(sdks)
			)
			for _, s := range sdks {
				sdkName := s.Plugin.SdkName
				index++
				pterm.Printf("[%s/%d]: Updating %s plugin...\n", pterm.Green(index), total, pterm.Green(sdkName))
				if err = manager.Update(sdkName); err != nil {
					pterm.Println(fmt.Sprintf("Update plugin(%s) failed, %s", sdkName, err.Error()))
				}
			}
		} else {
			return cli.Exit(err.Error(), 1)
		}
	} else {
		args := ctx.Args()
		l := args.Len()
		if l < 1 {
			return cli.Exit("invalid arguments", 1)
		}

		return manager.Update(args.First())
	}
	return nil
}
