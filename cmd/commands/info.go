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
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/sdk"
)

var Info = &cli.Command{
	Name:      "info",
	Usage:     "Show plugin info or SDK path",
	ArgsUsage: "[<sdk> | <sdk>@<version>]",
	Action:    infoCmd,
	Category:  CategoryPlugin,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "format",
			Aliases: []string{"f"},
			Usage:   "Format the output using the given Go template",
		},
	},
}

func infoCmd(ctx context.Context, cmd *cli.Command) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()
	args := cmd.Args().First()
	if args == "" {
		return cli.Exit("invalid arguments", 1)
	}

	// Check if the argument is in the format <sdk>@<version>
	if strings.Contains(args, "@") {
		argArr := strings.Split(args, "@")
		if len(argArr) == 2 {
			name := strings.ToLower(argArr[0])
			version := sdk.Version(argArr[1])

			// Check for empty SDK name or version
			if name != "" && string(version) != "" {
				sdkSource, err := manager.LookupSdk(name)
				if err != nil {
					if cmd.IsSet("format") {
						// For template output, we still need to output something
						data := struct {
							Name    string
							Version string
							Path    string
						}{
							Name:    name,
							Version: string(version),
							Path:    "notfound",
						}
						return executeTemplate(cmd, data)
					}
					fmt.Println("notfound")
					return nil
				}
				path := ""
				runtimePackage, err := sdkSource.GetRuntimePackage(version)
				if err == nil {
					// TODO check
					path = filepath.Join(runtimePackage.PackagePath)
				} else {
					path = "notfound"
				}

				// Check if format flag is set
				formatValue := cmd.String("format")
				if formatValue != "" {
					data := struct {
						Name    string
						Version string
						Path    string
					}{
						Name:    name,
						Version: string(version),
						Path:    path,
					}
					return executeTemplate(cmd, data)
				}

				fmt.Println(path)
				return nil
			}
		}
	}

	// Show plugin info
	s, err := manager.LookupSdk(args)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", args, err)
	}
	source := s.Metadata()

	// If format flag is set, prepare data for template
	if cmd.IsSet("format") {
		data := struct {
			Name        string
			Version     string
			Homepage    string
			InstallPath string
			Description string
		}{
			Name:        source.PluginMetadata.Name,
			Version:     source.PluginMetadata.Version,
			Homepage:    source.PluginMetadata.Homepage,
			InstallPath: source.SdkInstalledPath,
			Description: source.PluginMetadata.Description,
		}
		return executeTemplate(cmd, data)
	}

	pterm.Println("Plugin Info:")
	pterm.Println("Name    ", "->", pterm.LightBlue(source.PluginMetadata.Name))
	pterm.Println("Version ", "->", pterm.LightBlue(source.PluginMetadata.Version))
	pterm.Println("Homepage", "->", pterm.LightBlue(source.PluginMetadata.Homepage))
	pterm.Println("Desc    ", "->")
	pterm.Println(pterm.LightBlue(source.PluginMetadata.Description))
	source.PluginMetadata.ShowNotes()
	return nil
}

func executeTemplate(cmd *cli.Command, data interface{}) error {
	tmplStr := cmd.String("format")
	tmpl, err := template.New("format").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}
	return tmpl.Execute(cmd.Writer, data)
}
