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
	"encoding/json"
	"fmt"
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
		&cli.BoolFlag{
			Name:  "json",
			Usage: "Output as JSON",
		},
	},
}

const notFound = "notfound"

// versionData is what a --format template for <sdk>@<version> can use.
type versionData struct {
	Name    string
	Version string
	// Path is the directory that holds the version: the parent of Home. It is
	// kept as it always was so scripts that use it keep working.
	Path string
	// Home is the main runtime directory, the one to point JAVA_HOME and the
	// like at.
	Home string
	// Origin is "installed" for versions vfox installed and "linked" for SDKs
	// that live elsewhere and were registered with `vfox link`.
	Origin string
	// Target is where a linked version points to.
	Target string
}

type runtimeJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

type versionJSON struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Installed   bool          `json:"installed"`
	Origin      string        `json:"origin,omitempty"`
	Target      string        `json:"target,omitempty"`
	Home        string        `json:"home,omitempty"`
	PackagePath string        `json:"packagePath,omitempty"`
	Additions   []runtimeJSON `json:"additions,omitempty"`
	Env         *envJSON      `json:"env,omitempty"`
	Receipt     *sdk.Receipt  `json:"receipt,omitempty"`
}

type envJSON struct {
	Paths     []string          `json:"paths"`
	Variables map[string]string `json:"variables"`
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
	if cmd.IsSet("json") && cmd.IsSet("format") {
		return cli.Exit("use either --format or --json", 1)
	}

	// Check if the argument is in the format <sdk>@<version>
	if strings.Contains(args, "@") {
		argArr := strings.Split(args, "@")
		if len(argArr) == 2 {
			name := strings.ToLower(argArr[0])
			version := sdk.Version(argArr[1])

			// Check for empty SDK name or version
			if name != "" && string(version) != "" {
				return infoVersion(cmd, manager, name, version)
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
	if cmd.IsSet("json") {
		return writeJSON(cmd, struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Homepage    string `json:"homepage"`
			InstallPath string `json:"installPath"`
			Description string `json:"description"`
		}{
			Name:        source.PluginMetadata.Name,
			Version:     source.PluginMetadata.Version,
			Homepage:    source.PluginMetadata.Homepage,
			InstallPath: source.SdkInstalledPath,
			Description: source.PluginMetadata.Description,
		})
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

// infoVersion shows where an installed version lives and what it is.
func infoVersion(cmd *cli.Command, manager *internal.Manager, name string, version sdk.Version) error {
	sdkSource, err := manager.LookupSdk(name)
	if err != nil {
		if cmd.IsSet("json") {
			return writeJSON(cmd, versionJSON{Name: name, Version: string(version)})
		}
		if cmd.IsSet("format") {
			// For template output, we still need to output something
			return executeTemplate(cmd, versionData{
				Name: name, Version: string(version), Path: notFound, Home: notFound,
			})
		}
		fmt.Println(notFound)
		return nil
	}
	installation, err := sdkSource.Installation(version)

	if cmd.IsSet("json") {
		return writeJSON(cmd, describeVersion(sdkSource, name, version, installation, err))
	}

	data := versionData{Name: name, Version: string(version), Path: notFound, Home: notFound}
	if err == nil {
		data.Path = installation.PackagePath
		data.Home = installation.Runtime.Path
		data.Origin = string(installation.Origin)
		data.Target = installation.Target
	}
	if cmd.IsSet("format") {
		return executeTemplate(cmd, data)
	}
	fmt.Println(data.Path)
	return nil
}

func describeVersion(s sdk.Sdk, name string, version sdk.Version, installation *sdk.Installation, err error) versionJSON {
	out := versionJSON{Name: name, Version: string(version)}
	if err != nil {
		return out
	}
	out.Installed = true
	out.Origin = string(installation.Origin)
	out.Target = installation.Target
	out.Home = installation.Runtime.Path
	out.PackagePath = installation.PackagePath
	out.Receipt = installation.Receipt
	for _, addition := range installation.Additions {
		out.Additions = append(out.Additions, runtimeJSON{Name: addition.Name, Version: string(addition.Version), Path: addition.Path})
	}
	if envs, envErr := s.EnvKeys(installation.RuntimePackage); envErr == nil {
		out.Env = &envJSON{Paths: envs.Paths.Slice(), Variables: map[string]string{}}
		for key, value := range envs.Variables {
			if value != nil {
				out.Env.Variables[key] = *value
			}
		}
	}
	return out
}

func writeJSON(cmd *cli.Command, v any) error {
	encoder := json.NewEncoder(cmd.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func executeTemplate(cmd *cli.Command, data interface{}) error {
	tmplStr := cmd.String("format")
	tmpl, err := template.New("format").Parse(tmplStr)
	if err != nil {
		return fmt.Errorf("error parsing template: %w", err)
	}
	return tmpl.Execute(cmd.Writer, data)
}
