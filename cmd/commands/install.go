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
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
)

var Install = &cli.Command{
	Name:     "install",
	Aliases:  []string{"i"},
	Usage:    "Install a version of the target SDK",
	Action:   installCmd,
	Category: CategorySDK,
}

func installCmd(ctx *cli.Context) error {
	sdkArg := ctx.Args().First()
	if sdkArg == "" {
		return cli.Exit("sdk name is required", 1)
	}
	argArr := strings.Split(sdkArg, "@")
	argsLen := len(argArr)
	manager := internal.NewSdkManager()
	defer manager.Close()
	if argsLen > 2 {
		return cli.Exit("sdk version is invalid", 1)
	} else {
		var name string
		var version internal.Version
		if argsLen == 2 {
			name = strings.ToLower(argArr[0])
			version = internal.Version(argArr[1])
		} else {
			name = strings.ToLower(argArr[0])
			version = ""
		}
		source, err := manager.LookupSdkWithInstall(name)
		if err != nil {
			return err
		}
		return source.Install(version)
	}
}
