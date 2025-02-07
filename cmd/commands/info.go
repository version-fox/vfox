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

var Info = &cli.Command{
	Name:     "info",
	Usage:    "Show plugin info",
	Action:   infoCmd,
	Category: CategoryPlugin,
}

func infoCmd(ctx *cli.Context) error {
	manager := internal.NewSdkManager()
	defer manager.Close()
	args := ctx.Args().First()
	if args == "" {
		return cli.Exit("invalid arguments", 1)
	}
	s, err := manager.LookupSdk(args)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", args, err)
	}
	source := s.Plugin

	pterm.Println("Plugin Info:")
	pterm.Println("Name    ", "->", pterm.LightBlue(source.Name))
	pterm.Println("Version ", "->", pterm.LightBlue(source.Version))
	pterm.Println("Homepage", "->", pterm.LightBlue(source.Homepage))
	pterm.Println("Desc    ", "->")
	pterm.Println(pterm.LightBlue(source.Description))
	if len(source.LegacyFilenames) == 0 {
		pterm.Println("Legacy Files ->", pterm.LightRed("None"))
	} else {
		pterm.Println("Legacy Files ->", pterm.LightBlue(source.LegacyFilenames))
	}
	source.ShowNotes()
	return nil
}
