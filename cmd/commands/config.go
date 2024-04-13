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
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
)

var Config = &cli.Command{
	Name:  "config",
	Usage: "Config operation, list, storage etc.",
	Subcommands: []*cli.Command{
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "list all config info",
			Action:  ConfigListCmd,
		},
		{
			Name:  "storage",
			Usage: "Update storage info, such as sdkPath",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "sdk-path",
					Usage: "storage sdk path",
				},
			},
			Action: configCmd,
		},
	},
	Category: ConfigPlugin,
}

// configCmd config such as storage sdkPath
func configCmd(ctx *cli.Context) error {
	sdkPath := ctx.String("sdk-path")

	manager := internal.NewSdkManager()
	defer manager.Close()

	var err error
	if sdkPath != "" {
		err = manager.UpdateConfigStorageSdkPath(sdkPath)
	}

	if err != nil {
		return err
	}

	return err
}

// ConfigListCmd config list all
func ConfigListCmd(ctx *cli.Context) error {
	ctx.Args().First()

	manager := internal.NewSdkManager()
	defer manager.Close()
	manager.PrintConfigInfo()
	return nil
}
