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
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal/sdk"
	"strings"
)

var Install = &cli.Command{
	Name:    "install",
	Aliases: []string{"i"},
	Usage:   "install a version of sdk",
	Action:  installCmd,
}

func installCmd(ctx *cli.Context) error {
	sdkArg := ctx.Args().First()
	if sdkArg == "" {
		return cli.Exit("sdk name is required", 1)
	}
	argArr := strings.Split(sdkArg, "@")
	argsLen := len(argArr)
	manager := sdk.NewSdkManager()
	defer manager.Close()
	if argsLen > 2 {
		return cli.Exit("sdk version is invalid", 1)
	} else {
		var name string
		var version sdk.Version
		if argsLen == 2 {
			name = strings.ToLower(argArr[0])
			version = sdk.Version(argArr[1])
		} else {
			name = strings.ToLower(argArr[0])
			version = ""
		}
		source, err := manager.LookupSdk(name)
		if err != nil {
			return fmt.Errorf("%s not supported, error: %w", name, err)
		}
		return source.Install(version)
	}
}
