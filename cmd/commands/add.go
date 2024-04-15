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
	"errors"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
)

var Add = &cli.Command{
	Name:  "add",
	Usage: "Add plugins, use blank spaces separate the SDK name",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "source",
			Aliases: []string{"s"},
			Usage:   "plugin source",
		},
		&cli.StringFlag{
			Name:  "alias",
			Usage: "plugin alias",
		},
	},
	Action:   addCmd,
	Category: CategoryPlugin,
}

// addCmd is the command to add a plugin of sdk
func addCmd(ctx *cli.Context) error {
	args := ctx.Args()
	var errStr = ""
	var source = ""
	var alias = ""
	var err error

	for _, sdkName := range args.Slice() {
		// only for when adding one plugin
		if args.Len() == 1 {
			source = ctx.String("source")
			alias = ctx.String("alias")
		}

		manager := internal.NewSdkManager()
		err = manager.Add(sdkName, source, alias)
		if err != nil {
			errStr += err.Error() + "\r\n"
			continue
		}
		manager.Close()
	}
	if errStr != "" {
		err = errors.New(errStr)
	}

	return err
}
