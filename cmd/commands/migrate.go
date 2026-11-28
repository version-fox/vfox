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

	"github.com/urfave/cli/v3"

	"github.com/version-fox/vfox/internal"
)

var Migrate = &cli.Command{
	Name:      "migrate",
	Usage:     "Move the SDK install root and set storage.sdkPath",
	ArgsUsage: "<path>",
	Action:    migrateCmd,
	Category:  CategorySDK,
}

func migrateCmd(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return cli.Exit("usage: vfox migrate <path>", 1)
	}
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()
	return manager.MigrateSdkPath(cmd.Args().First())
}
