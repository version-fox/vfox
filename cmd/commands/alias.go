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
	"sort"
	"strings"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/sdk"
)

var Alias = &cli.Command{
	Name:     "alias",
	Usage:    "Give installed versions of an SDK short names of your own",
	Category: CategorySDK,
	Commands: []*cli.Command{
		{
			Name:      "set",
			Usage:     "Name an installed version",
			ArgsUsage: "<sdk> <alias> <version>",
			Action:    aliasSetCmd,
		},
		{
			Name:      "list",
			Aliases:   []string{"ls"},
			Usage:     "List aliases",
			ArgsUsage: "[<sdk>]",
			Action:    aliasListCmd,
		},
		{
			Name:      "unset",
			Usage:     "Remove an alias",
			ArgsUsage: "<sdk> <alias>",
			Action:    aliasUnsetCmd,
		},
	},
}

func aliasSetCmd(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 3 {
		return cli.Exit("usage: vfox alias set <sdk> <alias> <version>", 1)
	}
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	name := strings.ToLower(cmd.Args().Get(0))
	alias := cmd.Args().Get(1)
	version := sdk.Version(cmd.Args().Get(2))
	// `vfox current` prints versions with a leading v, so accept that spelling.
	if source, err := manager.LookupSdk(name); err == nil && !source.CheckRuntimeExist(version) {
		if trimmed := sdk.Version(strings.TrimPrefix(string(version), "v")); source.CheckRuntimeExist(trimmed) {
			version = trimmed
		}
	}
	if err := manager.SetAlias(name, alias, version); err != nil {
		return err
	}
	pterm.Printf("%s now stands for %s.\n", pterm.LightGreen(fmt.Sprintf("%s@%s", name, alias)), fmt.Sprintf("%s@%s", name, version))
	return nil
}

func aliasUnsetCmd(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 2 {
		return cli.Exit("usage: vfox alias unset <sdk> <alias>", 1)
	}
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	name := strings.ToLower(cmd.Args().Get(0))
	alias := cmd.Args().Get(1)
	if err := manager.UnsetAlias(name, alias); err != nil {
		return err
	}
	pterm.Printf("Removed the alias %s.\n", pterm.LightGreen(fmt.Sprintf("%s@%s", name, alias)))
	return nil
}

func aliasListCmd(ctx context.Context, cmd *cli.Command) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	var names []string
	if sdkName := cmd.Args().First(); sdkName != "" {
		names = []string{strings.ToLower(sdkName)}
	} else {
		for name := range manager.RuntimeEnvContext.UserConfig.Aliases {
			names = append(names, name)
		}
		sort.Strings(names)
	}
	printed := false
	for _, name := range names {
		aliases := manager.AliasesFor(name)
		keys := make([]string, 0, len(aliases))
		for alias := range aliases {
			keys = append(keys, alias)
		}
		sort.Strings(keys)
		for _, alias := range keys {
			fmt.Fprintf(cmd.Writer, "%s@%s -> %s@%s\n", name, alias, name, aliases[alias])
			printed = true
		}
	}
	if !printed {
		fmt.Fprintln(cmd.Writer, "No aliases defined. Create one with: vfox alias set <sdk> <alias> <version>")
	}
	return nil
}
