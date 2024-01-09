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
	"os"

	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shell"
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
	},
	Action: envCmd,
}

func envCmd(ctx *cli.Context) error {
	if ctx.IsSet("cleanup") {
		manager := sdk.NewSdkManager()
		defer manager.Close()
		temp, err := sdk.NewTemp(manager.PathMeta.TempPath, os.Getppid())
		if err != nil {
			return err
		}
		// Clean up the old temp files, before today.
		temp.Remove()
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
		manager := sdk.NewSdkManagerWithSource(sdk.SessionRecordSource, sdk.ProjectRecordSource)
		defer manager.Close()
		envKeys := manager.EnvKeys()
		exportStr := s.Export(envKeys)
		fmt.Println(exportStr)
		return nil
	}
}
