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
	"github.com/pterm/pterm/putils"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/sdk"
)

var List = &cli.Command{
	Name:     "list",
	Aliases:  []string{"ls"},
	Usage:    "List all versions of the target SDK",
	Action:   listCmd,
	Category: CategorySDK,
}

func listCmd(ctx context.Context, cmd *cli.Command) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()
	sdkName := cmd.Args().First()
	if sdkName == "" {
		allSdk, err := manager.LoadAllSdk()
		if err != nil {
			return err
		}
		if len(allSdk) == 0 {
			return fmt.Errorf("you don't have any sdk installed yet")
		}
		printInstalledSdks(allSdk, manager.AliasesFor)
		return nil
	}
	source, err := manager.LookupSdk(sdkName)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", sdkName, err)
	}
	curVersion := source.Current()
	list := source.InstalledList()
	if len(list) == 0 {
		return fmt.Errorf("no available version")
	}
	aliases := aliasesByVersion(manager.AliasesFor(sdkName))
	for _, version := range list {
		label := versionLabel(source, version, aliases)
		if version == curVersion {
			pterm.Println("->", label, pterm.LightGreen("<— current"))
		} else {
			pterm.Println("->", label)
		}
	}
	return nil
}

// aliasesByVersion turns alias -> version into version -> sorted aliases.
func aliasesByVersion(aliases map[string]string) map[string][]string {
	byVersion := make(map[string][]string)
	for alias, version := range aliases {
		byVersion[version] = append(byVersion[version], alias)
	}
	for _, names := range byVersion {
		sort.Strings(names)
	}
	return byVersion
}

// versionLabel is how `vfox list` shows a version: its name, where it is linked
// to if it is a link, and the aliases it has.
func versionLabel(s sdk.Sdk, version sdk.Version, aliases map[string][]string) string {
	origin, target := sdk.OriginInstalled, ""
	if installation, err := s.Installation(version); err == nil {
		origin, target = installation.Origin, installation.Target
	}
	return formatVersionLabel(version, origin, target, aliases[string(version)])
}

func formatVersionLabel(version sdk.Version, origin sdk.Origin, target string, aliases []string) string {
	label := string(version)
	if origin == sdk.OriginLinked {
		label += " (linked"
		if target != "" {
			label += " → " + target
		}
		label += ")"
	}
	if len(aliases) > 0 {
		label += " [alias: " + strings.Join(aliases, ", ") + "]"
	}
	return label
}

// printInstalledSdks prints a tree of the SDKs with their installed versions.
// The Clink completion script, internal/shell/clink_vfox.lua, reads the SDK
// names and the versions from the lines of the tree; clink_format_test.go locks
// the layout. aliasesFor returns the aliases of an SDK as alias -> version.
func printInstalledSdks(allSdk []sdk.Sdk, aliasesFor func(sdkName string) map[string]string) {
	tree := pterm.LeveledList{}

	for _, s := range allSdk {
		name := s.Metadata().Name
		tree = append(tree, pterm.LeveledListItem{Level: 0, Text: name})
		aliases := aliasesByVersion(aliasesFor(name))
		for _, version := range s.InstalledList() {
			tree = append(tree, pterm.LeveledListItem{Level: 1, Text: versionLabel(s, version, aliases)})
		}
	}
	// Generate tree from LeveledList.
	root := putils.TreeFromLeveledList(tree)
	root.Text = "All installed sdk versions"
	// Render TreePrinter
	_ = pterm.DefaultTree.WithRoot(root).Render()
}
