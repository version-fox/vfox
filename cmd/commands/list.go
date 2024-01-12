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
	"fmt"
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal/sdk"
)

var List = &cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "list all versions of the target sdk",
	Action:  listCmd,
}

func listCmd(ctx *cli.Context) error {
	manager := sdk.NewSdkManager()
	defer manager.Close()
	sdkName := ctx.Args().First()
	if sdkName == "" {
		allSdk, err := manager.LoadAllSdk()
		if err != nil {
			return err
		}
		if len(allSdk) == 0 {
			pterm.Println("You don't have any sdk installed yet.")
			return nil
		}
		tree := pterm.LeveledList{}
		for name, s := range allSdk {
			tree = append(tree, pterm.LeveledListItem{Level: 0, Text: name})
			for _, version := range s.List() {
				tree = append(tree, pterm.LeveledListItem{Level: 1, Text: "v" + string(version)})
			}
		}
		// Generate tree from LeveledList.
		root := putils.TreeFromLeveledList(tree)
		root.Text = "All installed sdk versions"
		// Render TreePrinter
		_ = pterm.DefaultTree.WithRoot(root).Render()
		return nil
	}
	source, err := manager.LookupSdk(sdkName)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", sdkName, err)
	}
	curVersion := source.Current()
	list := source.List()
	if len(list) == 0 {
		pterm.Println("No available version.")
		return nil
	}
	for _, version := range list {
		if version == curVersion {
			pterm.Println("->", fmt.Sprintf("v%s", version), pterm.LightGreen("<— current"))
		} else {
			pterm.Println("->", fmt.Sprintf("v%s", version))
		}
	}
	return nil
}
