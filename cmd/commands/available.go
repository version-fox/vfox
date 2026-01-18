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
	"strings"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
)

var Available = &cli.Command{
	Name:     "available",
	Usage:    "Show all available plugins",
	Action:   availableCmd,
	Category: CategoryPlugin,
}

func availableCmd(ctx context.Context, cmd *cli.Command) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()
	//categoryName := cmd.Args().First()
	available, err := manager.Available()
	if err != nil {
		return err
	}

	pterm.Println(pterm.Bold.Sprint("AVAILABLE PLUGINS"))
	pterm.Println()

	maxNameLen := 0
	for _, item := range available {
		if len(item.Name) > maxNameLen {
			maxNameLen = len(item.Name)
		}
	}
	nameWidth := maxNameLen + 2

	for _, item := range available {
		isOfficial := strings.HasPrefix(item.Homepage, "https://github.com/version-fox/")

		official := pterm.LightRed("✗")
		if isOfficial {
			official = pterm.LightGreen("✓")
		}

		nameCol := fmt.Sprintf("%-*s", nameWidth, item.Name)
		pterm.Printf("  %s %s  %s\n", pterm.FgCyan.Sprint(nameCol), official, pterm.FgLightWhite.Sprint(item.Homepage))
	}

	pterm.Println()
	pterm.Printf("  %s\n", pterm.FgGray.Sprint("Use 'vfox add <plugin>' to install"))
	return nil

}
