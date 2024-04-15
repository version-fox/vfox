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
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
)

var Uninstall = &cli.Command{
	Name:     "uninstall",
	Aliases:  []string{"un"},
	Usage:    "Uninstall a version of the target SDK",
	Action:   uninstallCmd,
	Category: CategorySDK,
}

func uninstallCmd(ctx *cli.Context) error {
	sdkArg := ctx.Args().First()
	if sdkArg == "" {
		return cli.Exit("sdk name is required", 1)
	}
	manager := internal.NewSdkManager()
	defer manager.Close()
	argArr := strings.Split(sdkArg, "@")
	argsLen := len(argArr)
	if argsLen != 2 {
		return cli.Exit("sdk version is invalid", 1)
	}

	name := strings.ToLower(argArr[0])
	version := internal.Version(argArr[1])

	source, err := manager.LookupSdk(name)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", name, err)
	}
	cv := source.Current()
	if err = source.Uninstall(version); err != nil {
		return err
	}
	remainVersion := source.List()
	if len(remainVersion) == 0 {
		_ = os.RemoveAll(source.InstallPath)
		return nil
	}
	if cv == version {
		pterm.Println("Auto switch to the other version.")
		firstVersion := remainVersion[0]
		return source.Use(firstVersion, internal.Global)
	}
	return nil
}
