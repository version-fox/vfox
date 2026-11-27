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

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
)

var Add = &cli.Command{
	Name:  "add",
	Usage: "Add a plugin or plugins",
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
func addCmd(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	if args.Len() > 1 && (cmd.IsSet("source") || cmd.IsSet("alias")) {
		return fmt.Errorf("--source and --alias apply to one plugin, run add once per plugin")
	}

	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	// multiple plugins
	if args.Len() > 1 {
		seen := make(map[string]struct{}, args.Len())
		uniqueArgs := make([]string, 0, args.Len())
		for _, sdkName := range args.Slice() {
			key := internal.NormalizePluginName(sdkName)
			if key == "" {
				key = sdkName
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			uniqueArgs = append(uniqueArgs, sdkName)
		}

		errOut := cmd.ErrWriter
		if errOut == nil {
			errOut = os.Stderr
		}

		var failed []string
		for index, sdkName := range uniqueArgs {
			pterm.Printf("[%s/%d]: Adding %s plugin...\n", pterm.Green(index+1), len(uniqueArgs), pterm.Green(sdkName))
			if err := manager.Add(sdkName, "", ""); err != nil {
				_, _ = fmt.Fprintf(errOut, "Add plugin(%s) failed: %s\n", sdkName, err.Error())
				failed = append(failed, sdkName)
			}
		}
		if len(failed) == 1 {
			return fmt.Errorf("failed to add plugin: %s", failed[0])
		}
		if len(failed) > 1 {
			return fmt.Errorf("failed to add plugins: %s", strings.Join(failed, ", "))
		}
		return nil
	} else {
		sdkName := args.First()
		source := cmd.String("source")
		alias := cmd.String("alias")
		err := manager.Add(sdkName, source, alias)
		if err == nil {
			pterm.Printf("Please use `%s` to install the version you need.\n", pterm.LightBlue(fmt.Sprintf("vfox install %s@<version>", sdkName)))
		}
		return err
	}
}
