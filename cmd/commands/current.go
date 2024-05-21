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
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
)

var Current = &cli.Command{
	Name:      "current",
	Aliases:   []string{"c"},
	Usage:     "Show current version of the target SDK",
	UsageText: "Show current version of all SDK's if no parameters are passed",
	Action:    currentCmd,
	Category:  CategorySDK,
}

func currentCmd(ctx *cli.Context) error {
	manager := internal.NewSdkManager()
	defer manager.Close()
	sdkName := ctx.Args().First()
	if sdkName == "" {
		allSdk, err := manager.LoadAllSdk()
		if err != nil {
			return err
		}

		allSdk.ForEachBySort(func(name string, s *internal.Sdk) {
			current := s.Current()
			if current == "" {
				pterm.Printf("%s -> N/A \n", name)
			} else {
				pterm.Printf("%s -> %s\n", name, pterm.LightGreen("v"+string(current)))
			}
		})

		return nil
	}
	source, err := manager.LookupSdk(sdkName)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", sdkName, err)
	}
	current := source.Current()
	if current == "" {
		return fmt.Errorf("no current version of %s", sdkName)
	}
	pterm.Println("->", pterm.LightGreen("v"+string(current)))
	return nil
}
