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
	"github.com/version-fox/vfox/internal/sdk"
)

var Update = &cli.Command{
	Name:   "update",
	Usage:  "update specified plug-ins",
	Action: updateCmd,
}

func updateCmd(ctx *cli.Context) error {
	args := ctx.Args()
	l := args.Len()
	if l < 1 {
		return cli.Exit("invalid arguments", 1)
	}
	manager := sdk.NewSdkManager()
	defer manager.Close()
	_ = manager.Update(args.First())
	return nil
}
