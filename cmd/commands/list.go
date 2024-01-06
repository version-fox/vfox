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
)

var List = &cli.Command{
	Name:    "list",
	Aliases: []string{"ls"},
	Usage:   "list all versions of the target sdk",
	Action:  listCmd,
}

func listCmd(ctx *cli.Context) error {
	manager := sdk.NewSdkManager()
	sdkName := ctx.Args().First()
	return manager.List(sdk.Arg{Name: sdkName})
}
