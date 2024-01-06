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
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/sdk"
	"strings"
)

var Use = &cli.Command{
	Name:    "use",
	Aliases: []string{"u"},
	Usage:   "use a version of sdk",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "global",
			Aliases: []string{"g"},
			Usage:   "used with the global environment",
		},
		&cli.StringFlag{
			Name:    "project",
			Aliases: []string{"p"},
			Usage:   "used with the current directory",
		},
		&cli.StringFlag{
			Name:    "session",
			Aliases: []string{"s"},
			Usage:   "used with the current shell session",
		},
	},
	Action: useCmd,
}

func useCmd(ctx *cli.Context) error {
	manager := sdk.NewSdkManager()
	sdkArg := ctx.Args().First()
	arg := sdk.Arg{}
	if len(sdkArg) != 0 {
		argArr := strings.Split(sdkArg, "@")
		if len(argArr) <= 1 {
			arg.Name = argArr[0]
		} else {
			arg.Name = argArr[0]
			arg.Version = argArr[1]
		}
	}
	scope := sdk.Session
	if ctx.IsSet("global") {
		scope = sdk.Global
	} else if ctx.IsSet("project") {
		scope = sdk.Project
	} else {
		scope = sdk.Session
	}
	// TODO Consider how to handle exceptions appropriately. Print directly or return?
	_ = manager.Use(arg, scope)
	return nil
}
