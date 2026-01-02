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

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/base"
)

var Unuse = &cli.Command{
	Name:  "unuse",
	Usage: "Unset a version of the target SDK",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "global",
			Aliases: []string{"g"},
			Usage:   "Unset from the global environment",
		},
		&cli.BoolFlag{
			Name:    "project",
			Aliases: []string{"p"},
			Usage:   "Unset from the current directory",
		},
		&cli.BoolFlag{
			Name:    "session",
			Aliases: []string{"s"},
			Usage:   "Unset from the current shell session",
		},
	},
	Action:   unuseCmd,
	Category: CategorySDK,
}

func unuseCmd(ctx context.Context, cmd *cli.Command) error {
	sdkName := cmd.Args().First()
	if len(sdkName) == 0 {
		return fmt.Errorf("invalid parameter. format: <sdk-name>")
	}

	scope := base.Session
	if cmd.IsSet("global") {
		scope = base.Global
	} else if cmd.IsSet("project") {
		scope = base.Project
	} else {
		scope = base.Session
	}

	manager := internal.NewSdkManager()
	defer manager.Close()

	source, err := manager.LookupSdk(sdkName)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", sdkName, err)
	}

	return source.Unuse(scope)
}
