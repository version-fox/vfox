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
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/base"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/shell"
	"github.com/version-fox/vfox/internal/toolset"
)

var Env = &cli.Command{
	Name: "env",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "shell",
			Aliases: []string{"s"},
			Usage:   "shell name",
		},
		&cli.BoolFlag{
			Name:    "cleanup",
			Aliases: []string{"c"},
			Usage:   "cleanup old temp files",
		},
		&cli.BoolFlag{
			Name:    "json",
			Aliases: []string{"j"},
			Usage:   "output json format",
		},
		&cli.BoolFlag{
			Name:  "full",
			Usage: "output full env",
		},
	},
	Action:   envCmd,
	Category: CategorySDK,
}

func envCmd(ctx context.Context, cmd *cli.Command) error {
	if cmd.IsSet("json") {
		return outputJSON()
	} else if cmd.IsSet("cleanup") {
		return cleanTmp()
	} else if cmd.IsSet("full") {
		return envFlag(cmd, "full")
	} else {
		return envFlag(cmd, "cwd")
	}
}

func outputJSON() error {
	type SDKs map[string]map[string]*string
	data := struct {
		IsHookEnv bool     `json:"is_hook_env"`
		Paths     []string `json:"paths"`
		SDKs      SDKs     `json:"sdks"`
	}{
		IsHookEnv: env.IsHookEnv(),
		Paths:     []string{},
		SDKs:      make(SDKs),
	}
	manager := internal.NewSdkManager()
	defer manager.Close()
	tvs, err := toolset.NewMultiToolVersions([]string{
		manager.PathMeta.WorkingDirectory,
		manager.PathMeta.CurTmpPath,
		manager.PathMeta.HomePath,
	})
	if err != nil {
		return err
	}
	tvs.FilterTools(func(name, version string) bool {
		if lookupSdk, err := manager.LookupSdk(name); err == nil {
			if keys, err := lookupSdk.EnvKeys(base.Version(version), base.OriginalLocation); err == nil {
				data.SDKs[lookupSdk.Plugin.Name] = keys.Variables
				data.Paths = append(data.Paths, keys.Paths.Slice()...)
				return true
			}
		}
		return false
	})
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

func cleanTmp() error {
	manager := internal.NewSdkManager()
	defer manager.Close()
	// Clean up the old temp files, before today.
	manager.CleanTmp()
	return nil
}

func envFlag(cmd *cli.Command, mode string) error {
	shellName := cmd.String("shell")
	if shellName == "" {
		// Try to auto-detect shell if not provided
		if detected := shell.GetShellName(); detected != "" {
			shellName = detected
			fmt.Fprintf(os.Stderr, "Warning: No shell specified, auto-detected: %s\n", shellName)
			fmt.Fprintf(os.Stderr, "To avoid this warning, specify shell explicitly:\n")
			fmt.Fprintf(os.Stderr, "  vfox env --shell %s\n\n", shellName)
		} else {
			fmt.Fprintf(os.Stderr, "Error: shell name is required and auto-detection failed\n")
			fmt.Fprintf(os.Stderr, "\nUsage:\n")
			fmt.Fprintf(os.Stderr, "  vfox env --shell <shell>\n\n")
			fmt.Fprintf(os.Stderr, "Examples:\n")
			fmt.Fprintf(os.Stderr, "  vfox env --shell zsh\n")
			fmt.Fprintf(os.Stderr, "  vfox env -s bash\n")
			fmt.Fprintf(os.Stderr, "  vfox env --shell fish\n\n")
			fmt.Fprintf(os.Stderr, "Options:\n")
			fmt.Fprintf(os.Stderr, "  --shell, -s    shell name\n")
			fmt.Fprintf(os.Stderr, "  --full        output full env\n")
			fmt.Fprintf(os.Stderr, "  --json, -j    output json format\n")
			fmt.Fprintf(os.Stderr, "  --cleanup, -c cleanup old temp files\n")
			return cli.Exit("", 1)
		}
	}
	s := shell.NewShell(shellName)
	if s == nil {
		return fmt.Errorf("unknown target shell %s", shellName)
	}
	manager := internal.NewSdkManager()
	defer manager.Close()
	var sdkEnvs internal.SdkEnvs
	var err error
	if mode == "full" {
		sdkEnvs, err = manager.SessionEnvKeys(internal.SessionEnvOptions{
			WithGlobalEnv: true,
		})
	} else {
		sdkEnvs, err = manager.SessionEnvKeys(internal.SessionEnvOptions{
			WithGlobalEnv: false,
		})
	}

	if err != nil {
		return err
	}

	if len(sdkEnvs) == 0 {
		return nil
	}

	exportEnvs := sdkEnvs.ToExportEnvs()

	exportStr := s.Export(exportEnvs)
	fmt.Println(exportStr)
	return nil
}
