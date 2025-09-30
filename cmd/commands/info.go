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
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/base"
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
	manager := internal.NewSdkManager()
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
			version := base.Version(argArr[1])

			// Check for empty SDK name or version
			if name != "" && string(version) != "" {
				sdk, err := manager.LookupSdk(name)
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
				if sdk.CheckExists(version) {
					path = filepath.Join(sdk.VersionPath(version), fmt.Sprintf("%s-%s", name, version))
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
	source := s.Plugin

	// If format flag is set, prepare data for template
	if cmd.IsSet("format") {
		data := struct {
			Name        string
			Version     string
			Homepage    string
			InstallPath string
			Description string
		}{
			Name:        source.Name,
			Version:     source.Version,
			Homepage:    source.Homepage,
			InstallPath: s.InstallPath,
			Description: source.Description,
		}
		return executeTemplate(cmd, data)
	}

	pterm.Println("Plugin Info:")
	pterm.Println("Name    ", "->", pterm.LightBlue(source.Name))
	pterm.Println("Version ", "->", pterm.LightBlue(source.Version))
	pterm.Println("Homepage", "->", pterm.LightBlue(source.Homepage))
	pterm.Println("Desc    ", "->")
	pterm.Println(pterm.LightBlue(source.Description))
	source.ShowNotes()
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
