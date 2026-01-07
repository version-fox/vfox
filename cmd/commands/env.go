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

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/env/shell"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/sdk"
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
	} else {
		return envFlag(cmd)
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
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()
	envContext := manager.RuntimeEnvContext
	tvs, err := pathmeta.NewMultiToolVersions([]string{
		envContext.PathMeta.Working.Directory,
		envContext.PathMeta.Working.SessionShim,
		envContext.PathMeta.User.Home,
	})
	if err != nil {
		return err
	}
	tvs.FilterTools(func(name, version string) bool {
		if lookupSdk, err := manager.LookupSdk(name); err == nil {
			if keys, err := lookupSdk.EnvKeys(sdk.Version(version)); err == nil {
				metadata := lookupSdk.Metadata()
				data.SDKs[metadata.Name] = keys.Variables
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
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	// Clean up the old temp files, before today.
	manager.CleanTmp()
	return nil
}

func envFlag(cmd *cli.Command) error {
	shellName := cmd.String("shell")
	if shellName == "" {
		return cli.Exit("shell name is required", 1)
	}
	s := shell.NewShell(shellName)
	if s == nil {
		return fmt.Errorf("unknown target shell %s", shellName)
	}
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()
	projectToolVersions, err := manager.LoadToolVersionByScope(env.Project)
	if err != nil {
		return err
	}
	sessionToolVersions, err := manager.LoadToolVersionByScope(env.Session)
	if err != nil {
		return err
	}
	globalToolVersions, err := manager.LoadToolVersionByScope(env.Global)
	if err != nil {
		return err
	}

	sdkEnvs, err := manager.EnvKeys(pathmeta.MultiToolVersions{
		projectToolVersions,
		sessionToolVersions,
		globalToolVersions,
	})
	if err != nil {
		return err
	}

	if len(sdkEnvs.Variables) == 0 {
		return nil
	}

	exportStr := s.Export(sdkEnvs.Variables)
	fmt.Println(exportStr)
	return nil
}
