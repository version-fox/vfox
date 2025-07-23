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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/base"
)

var Path = &cli.Command{
	Name:    "path",
	Usage:   "Get the absolute path of the target SDK",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "json",
			Usage: "Output in JSON format",
		},
	},
	Action:   pathCmd,
	Category: CategorySDK,
}

func pathCmd(ctx *cli.Context) error {
	sdkArg := ctx.Args().First()
	if len(sdkArg) == 0 {
		return fmt.Errorf("invalid parameter. format: <sdk-name>[@<version>]")
	}

	var (
		name    string
		version base.Version
	)
	argArr := strings.Split(sdkArg, "@")
	if len(argArr) <= 1 {
		return fmt.Errorf("version is required. format: <sdk-name>@<version>")
	} else {
		name = argArr[0]
		version = base.Version(argArr[1])
	}

	manager := internal.NewSdkManager()
	defer manager.Close()

	source, err := manager.LookupSdk(name)
	if err != nil {
		if ctx.Bool("json") {
			result := map[string]string{
				"path":  "",
				"found": "false",
			}
			jsonOutput, _ := json.Marshal(result)
			fmt.Println(string(jsonOutput))
			return nil
		}
		fmt.Println("notfound")
		return nil
	}

	if !source.CheckExists(version) {
		if ctx.Bool("json") {
			result := map[string]string{
				"path":  "",
				"found": "false",
			}
			jsonOutput, _ := json.Marshal(result)
			fmt.Println(string(jsonOutput))
			return nil
		}
		fmt.Println("notfound")
		return nil
	}

	path := source.VersionPath(version)
	if ctx.Bool("json") {
		result := map[string]string{
			"path":  path,
			"found": "true",
		}
		jsonOutput, _ := json.Marshal(result)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println(path)
	}

	return nil
}