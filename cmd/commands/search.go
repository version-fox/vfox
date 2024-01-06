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
	"fmt"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal/printer"
	"github.com/version-fox/vfox/internal/sdk"
	"strings"
)

var Search = &cli.Command{
	Name:   "search",
	Usage:  "search a version of sdk",
	Action: searchCmd,
}

func searchCmd(ctx *cli.Context) error {
	sdkName := ctx.Args().First()
	if sdkName == "" {
		return cli.Exit("sdk name is required", 1)
	}
	manager := sdk.NewSdkManager()
	source, err := manager.LookupSdk(sdkName)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", sdkName, err)
	}
	result, err := source.Available()
	if err != nil {
		pterm.Printf("Plugin [Available] error: %s\n", err)
		return nil
	}
	if len(result) == 0 {
		pterm.Println("No Available version.")
		return nil
	}
	kvSelect := printer.PageKVSelect{
		TopText: "Please select a version of " + sdkName,
		Filter:  true,
		Size:    20,
		SourceFunc: func(page, size int) ([]*printer.KV, error) {
			start := page * size
			end := start + size

			if start > len(result) {
				return nil, fmt.Errorf("page is out of range")
			}
			if end > len(result) {
				end = len(result)
			}
			versions := result[start:end]
			var arr []*printer.KV
			for _, p := range versions {
				var value string
				if p.Main.Note != "" {
					value = fmt.Sprintf("v%s (%s)", p.Main.Version, p.Main.Note)
				} else {
					value = fmt.Sprintf("v%s", p.Main.Version)
				}
				if len(p.Additional) != 0 {
					var additional []string
					for _, a := range p.Additional {
						additional = append(additional, fmt.Sprintf("%s v%s", a.Name, a.Version))
					}
					value = fmt.Sprintf("%s [%s]", value, strings.Join(additional, ","))
				}
				arr = append(arr, &printer.KV{
					Key:   string(p.Main.Version),
					Value: value,
				})
			}
			return arr, nil
		},
	}
	version, err := kvSelect.Show()
	if err != nil {
		pterm.Printf("Select version error, err: %s\n", err)
		return err
	}
	return source.Install(sdk.Version(version.Key))
}
