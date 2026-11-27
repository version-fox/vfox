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
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/logger"
	"github.com/version-fox/vfox/internal/shared/printer"
	"github.com/version-fox/vfox/internal/shared/util"
	"golang.org/x/term"
)

var Search = &cli.Command{
	Name:  "search",
	Usage: "Search a version of the target SDK",
	// The arguments after the SDK name belong to the plugin's Available hook,
	// also the ones that start with a dash: vfox search clang --no-cache.
	// searchArgs reads vfox's own options.
	SkipFlagParsing: true,
	Action:          searchCmd,
	Category:        CategorySDK,
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
			value = string(p.Version)
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

	prompt := printer.StdinPrompt()
	if prompt.Input != printer.Keyboard {
		return installChosen(source, prompt, options, installedVersions, sdkName)
	}
	_, height, _ := term.GetSize(int(os.Stdout.Fd()))
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

// versionInstaller is what installing a chosen version needs from sdk.Sdk.
type versionInstaller interface {
	Install(version sdk.Version) error
}

// installChosen lets the person choose a version with prompt, which is not the
// keyboard, and installs it. Cancelling installs nothing.
func installChosen(installer versionInstaller, prompt printer.Prompt, options []*printer.KV, installed util.Set[string], sdkName string) error {
	selected, err := prompt.Choose("Please select a version of "+sdkName+" to install", options, installed)
	if err != nil {
		return fmt.Errorf("select version error: %w", err)
	}
	if selected == nil {
		return nil
	}
	return installer.Install(sdk.Version(selected.Key))
}

// searchHelpTemplate is the command help without the version flag under GLOBAL
// OPTIONS, because -v, -V and --version after the SDK name go to the plugin.
var searchHelpTemplate = strings.Replace(cli.CommandHelpTemplate,
	`{{template "visiblePersistentFlagTemplate" .}}`,
	`{{range .VisiblePersistentFlags}}{{if ne (index .Names 0) "version"}}
   {{wrap .String 6}}{{end}}{{end}}`, 1)

// searchCall is the search arguments split between vfox and the plugin.
type searchCall struct {
	sdkName    string
	pluginArgs []string
	help       bool
	debug      bool
}

// searchArgs splits the search arguments. Before a "--", vfox reads its own
// options wherever they are: -h, -help and -debug, each with one or two dashes
// and an optional =value, as the flag parser read them. The first other argument
// is the SDK name, and the rest go to the plugin in their order. After a "--",
// every argument goes to the plugin, except the SDK name if it is not given yet.
// -v, -V and --version are not vfox options here: after the SDK name they go
// to the plugin like any other argument. An SDK name that starts with a dash is
// an error. An empty SDK name stays empty, so the call fails with "sdk name is
// required" instead of taking the next argument as the SDK name.
func searchArgs(args []string) (searchCall, error) {
	var opts searchCall
	own := true
	sdkSeen := false
	for _, arg := range args {
		if own {
			if arg == "--" {
				own = false
				continue
			}
			if strings.HasPrefix(arg, "-") {
				name, value, hasValue := strings.Cut(strings.TrimPrefix(arg[1:], "-"), "=")
				if name == "h" || name == "help" || name == "debug" {
					on := true
					if hasValue {
						var err error
						if on, err = strconv.ParseBool(value); err != nil {
							return opts, fmt.Errorf("invalid value %q for flag -%s: parse error", value, name)
						}
					}
					if name == "debug" {
						opts.debug = on
					} else {
						opts.help = on
					}
					continue
				}
			}
		}
		if !sdkSeen {
			if strings.HasPrefix(arg, "-") {
				return opts, fmt.Errorf("the sdk name cannot start with a dash: %s (vfox --version prints the vfox version)", arg)
			}
			opts.sdkName = arg
			sdkSeen = true
			continue
		}
		opts.pluginArgs = append(opts.pluginArgs, arg)
	}
	return opts, nil
}

func searchCmd(ctx context.Context, cmd *cli.Command) error {
	opts, err := searchArgs(cmd.Args().Slice())
	if err != nil {
		return cli.Exit(err.Error(), 1)
	}
	if opts.help {
		cli.HelpPrinter(cmd.Root().Writer, searchHelpTemplate, cmd)
		return nil
	}
	if opts.debug {
		logger.SetLevel(logger.DebugLevel)
	}
	if opts.sdkName == "" {
		return cli.Exit("sdk name is required", 1)
	}
	return RunSearch(opts.sdkName, opts.pluginArgs)
}
