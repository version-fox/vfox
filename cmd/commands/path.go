/*
 *    Copyright 2025 Han Li and contributors
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
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/base"
)

var Path = &cli.Command{
	Name:     "path",
	Usage:    "Show the absolute path of an installed SDK",
	Action:   pathCmd,
	Category: CategorySDK,
}

func pathCmd(ctx *cli.Context) error {
	args := ctx.Args().First()
	if args == "" {
		return cli.Exit("sdk name is required", 1)
	}

	argArr := strings.Split(args, "@")
	if len(argArr) != 2 {
		return cli.Exit("invalid arguments, expected format: <sdk>@<version>", 1)
	}

	name := strings.ToLower(argArr[0])
	version := base.Version(argArr[1])

	manager := internal.NewSdkManager()
	defer manager.Close()

	sdk, err := manager.LookupSdk(name)
	if err != nil {
		fmt.Println("notfound")
		return nil
	}

	if sdk.CheckExists(version) {
		fmt.Println(sdk.VersionPath(version))
	} else {
		fmt.Println("notfound")
	}

	return nil
}