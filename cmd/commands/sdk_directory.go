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
 *
 */

package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/sdk"
)

// IncludePath prints the include directory of an SDK version, for build tools.
var IncludePath = &cli.Command{
	Name:      "include-path",
	Usage:     "Show the include directory of an SDK version",
	ArgsUsage: "<sdk>[@<version>]",
	UsageText: "Without @<version>, the version that `vfox env` puts on PATH here is used. The directory is the one the plugin reports.",
	Action:    sdkDirectoryCmd("include"),
	Category:  CategorySDK,
}

// LibPath prints the lib directory of an SDK version, for build tools.
var LibPath = &cli.Command{
	Name:      "lib-path",
	Usage:     "Show the lib directory of an SDK version",
	ArgsUsage: "<sdk>[@<version>]",
	UsageText: "Without @<version>, the version that `vfox env` puts on PATH here is used. The directory is the one the plugin reports.",
	Action:    sdkDirectoryCmd("lib"),
	Category:  CategorySDK,
}

// sdkDirectoryCmd returns the action of a command that prints one directory the
// plugin reports for an installed version, and nothing else, so that a Makefile
// can use the output as it is.
func sdkDirectoryCmd(name string) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		sdkName, typed, pinned := strings.Cut(cmd.Args().First(), "@")
		if sdkName == "" || (pinned && typed == "") {
			return cli.Exit("invalid arguments", 1)
		}
		manager, err := internal.NewSdkManager()
		if err != nil {
			return err
		}
		defer manager.Close()

		s, err := manager.LookupSdk(sdkName)
		if err != nil {
			return fmt.Errorf("%s not supported, error: %w", sdkName, err)
		}
		version := sdk.Version(typed)
		if pinned {
			version = manager.ResolveVersionName(sdkName, s, version)
		} else {
			version, err = installedVersionInEffect(manager, s)
			if err != nil {
				return fmt.Errorf("cannot tell which version applies here: %w", err)
			}
			if version == "" {
				return fmt.Errorf("no installed version of %s is set here; name one as %s@<version>", sdkName, sdkName)
			}
		}
		if !s.CheckRuntimeExist(version) {
			return fmt.Errorf("%s@%s is not installed", sdkName, version)
		}
		dir, err := s.ReportedPath(version, name)
		if err != nil {
			return fmt.Errorf("%s@%s: %w", sdkName, version, err)
		}
		if dir == "" {
			return fmt.Errorf("%s@%s has no %s directory: the plugin does not report one", sdkName, version, name)
		}
		fmt.Println(dir)
		return nil
	}
}
