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
	"math"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/printer"
	"github.com/version-fox/vfox/internal/shared/util"
	"golang.org/x/crypto/ssh/terminal"
)

var Search = &cli.Command{
	Name:     "search",
	Usage:    "Search a version of the target SDK",
	Action:   searchCmd,
	Category: CategorySDK,
}

func RunSearch(sdkName string, availableArgs []string) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()
	source, err := manager.LookupSdkWithInstall(sdkName, false)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", sdkName, err)
	}
	result, err := source.Available(availableArgs)
	if err != nil {
		return fmt.Errorf("plugin [Available] method error: %w", err)
	}
	if len(result) == 0 {
		return fmt.Errorf("no available version")
	}

	var options []*printer.KV
	for _, p := range result {
		var value string
		if p.Note != "" {
			value = fmt.Sprintf("%s (%s)", p.Version, p.Note)
		} else {
			value = fmt.Sprintf("%s", p.Version)
		}
		if len(p.Additions) != 0 {
			var additional []string
			for _, a := range p.Additions {
				additional = append(additional, fmt.Sprintf("%s %s", a.Name, a.Version))
			}
			value = fmt.Sprintf("%s [%s]", value, strings.Join(additional, ","))
		}
		options = append(options, &printer.KV{
			Key:   string(p.Version),
			Value: value,
		})
	}

	installedVersions := util.NewSet[string]()
	for _, version := range source.InstalledList() {
		installedVersions.Add(string(version))
	}

	if util.IsNonInteractiveTerminal() {
		fmt.Println("Available versions:")
		for _, option := range options {
			label := option.Value
			if installedVersions.Contains(option.Key) {
				label = fmt.Sprintf("%s (installed)", label)
			}
			fmt.Printf(" - %s\n", label)
		}
		return nil
	}

	_, height, _ := terminal.GetSize(int(os.Stdout.Fd()))
	kvSelect := printer.PageKVSelect{
		TopText:          "Please select a version of " + sdkName + " to install",
		Filter:           true,
		Size:             int(math.Min(math.Max(float64(height-3), 1), 20)),
		HighlightOptions: installedVersions,
		DisabledOptions:  installedVersions,
		Options:          options,
		SourceFunc: func(page, size int, options []*printer.KV) ([]*printer.KV, error) {
			start := page * size
			end := start + size

			if start > len(options) {
				return nil, fmt.Errorf("page is out of range")
			}
			if end > len(options) {
				end = len(options)
			}

			return options[start:end], nil
		},
	}
	version, err := kvSelect.Show()
	if err != nil {
		return fmt.Errorf("select version error: %w", err)
	}
	if version == nil {
		return nil
	}
	return source.Install(sdk.Version(version.Key))
}

func searchCmd(ctx context.Context, cmd *cli.Command) error {
	sdkName := cmd.Args().First()
	if sdkName == "" {
		return cli.Exit("sdk name is required", 1)
	}
	return RunSearch(sdkName, cmd.Args().Tail())
}
