/*
 *    Copyright 2024 [lihan aooohan@gmail.com]
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
	"github.com/version-fox/vfox/sdk"
)

var Available = &cli.Command{
	Name:   "available",
	Usage:  "show available plugins",
	Action: availableCmd,
}

func availableCmd(ctx *cli.Context) error {
	manager := sdk.NewSdkManager()
	categoryName := ctx.Args().First()
	categories, err := manager.Available()
	if err != nil {
		return err
	}
	data := pterm.TableData{
		{"NAME", "VERSION", "AUTHOR", "DESCRIPTION"},
	}
	for _, category := range categories {
		if len(categoryName) > 0 {
			if categoryName != category.Name {
				continue
			}
		}
		for _, p := range category.Plugins {
			desc := p.Desc
			if len(desc) == 0 {
				desc = "-"
			} else if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			data = append(data, []string{category.Name + "/" + p.Filename, p.Version, p.Author, desc})
		}
	}

	_ = pterm.DefaultTable.
		WithHasHeader().
		WithSeparator("\t ").
		WithData(data).Render()
	pterm.Printf("Please use %s to install plugin\n", pterm.LightBlue("vfox add <plugin name>"))
	return nil

}
