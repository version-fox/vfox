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
	"encoding/json"
	"fmt"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/shell"
	"os"
)

var Env = &cli.Command{
	Name:   "env",
	Hidden: true,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "shell",
			Aliases: []string{"s"},
			Usage:   "shell",
		},
		&cli.BoolFlag{
			Name:    "cleanup",
			Aliases: []string{"c"},
			Usage:   "cleanup temp file",
		},
		&cli.BoolFlag{
			Name:    "json",
			Aliases: []string{"j"},
			Usage:   "get envs as json",
		},
	},
	Action: envCmd,
}

func envCmd(ctx *cli.Context) error {
	if ctx.IsSet("json") {
		type SDKs map[string]map[string]string
		data := struct {
			IsHookEnv bool     `json:"is_hook_env"`
			Paths     []string `json:"paths"`
			SDKs      SDKs     `json:"sdks"`
		}{
			IsHookEnv: env.IsHookEnv(),
			Paths:     []string{},
			SDKs:      make(SDKs),
		}
		manager := internal.NewSdkManagerWithSource(internal.GlobalRecordSource, internal.SessionRecordSource, internal.ProjectRecordSource)
		defer manager.Close()
		for k, v := range manager.Record.Export() {
			if lookupSdk, err := manager.LookupSdk(k); err == nil {
				if keys, err := lookupSdk.EnvKeys(internal.Version(v)); err == nil {
					newEnv := make(map[string]string)
					for key, value := range keys {
						if key == "PATH" {
							data.Paths = append(data.Paths, *value)
						} else {
							newEnv[key] = *value
						}
					}
					if len(newEnv) > 0 {
						data.SDKs[lookupSdk.Plugin.Name] = newEnv
					}
				}
			}
		}
		jsonData, err := json.Marshal(data)
		if err != nil {
			return err
		}
		fmt.Println(string(jsonData))
		return nil
	} else if ctx.IsSet("cleanup") {
		manager := internal.NewSdkManager()
		defer manager.Close()
		// Clean up the old temp files, before today.
		manager.CleanTmp()
		return nil
	} else {
		shellName := ctx.String("shell")
		if shellName == "" {
			return cli.Exit("shell name is required", 1)
		}
		s := shell.NewShell(shellName)
		if s == nil {
			return fmt.Errorf("unknow target shell %s", shellName)
		}
		manager := internal.NewSdkManagerWithSource(internal.SessionRecordSource, internal.ProjectRecordSource)
		defer manager.Close()
		envKeys, err := manager.EnvKeys()
		if err != nil {
			return err
		}

		sdkPaths := envKeys["PATH"]
		if sdkPaths != nil {
			originPath := os.Getenv(env.PathFlag)
			paths := manager.EnvManager.Paths([]string{*sdkPaths, originPath})
			envKeys["PATH"] = &paths
		}

		exportStr := s.Export(envKeys)
		fmt.Println(exportStr)
		return nil
	}
}
