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
	"strings"
)

var Available = &cli.Command{
	Name:   "available",
	Usage:  "show available plugins",
	Action: availableCmd,
}

func availableCmd(ctx *cli.Context) error {
	manager := internal.NewSdkManager()
	defer manager.Close()
	//categoryName := ctx.Args().First()
	available, err := manager.Available()
	if err != nil {
		return err
	}
	data := pterm.TableData{
		{"NAME", "OFFICIAL", "HOMEPAGE", "DESCRIPTION"},
	}
	for _, item := range available {
		official := pterm.LightRed("NO")
		if strings.HasPrefix(item.Homepage, "https://github.com/version-fox/") {
			official = pterm.LightGreen("YES")
		}
		data = append(data, []string{item.Name, official, item.Homepage, item.Desc})
	}

	_ = pterm.DefaultTable.
		WithHasHeader().
		WithSeparator("\t ").
		WithData(data).Render()
	pterm.Printf("Please use %s to install plugin\n", pterm.LightBlue("vfox add <plugin name>"))
	return nil

}
