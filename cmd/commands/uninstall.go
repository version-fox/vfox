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
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/sdk"
)

var Uninstall = &cli.Command{
	Name:     "uninstall",
	Aliases:  []string{"un"},
	Usage:    "Uninstall a version of the target SDK",
	Action:   uninstallCmd,
	Category: CategorySDK,
}

func uninstallCmd(ctx context.Context, cmd *cli.Command) error {
	sdkArg := cmd.Args().First()
	if sdkArg == "" {
		return cli.Exit("sdk name is required", 1)
	}
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()
	argArr := strings.Split(sdkArg, "@")
	argsLen := len(argArr)
	if argsLen != 2 {
		return cli.Exit("sdk version is invalid", 1)
	}

	name := strings.ToLower(argArr[0])
	version := sdk.Version(argArr[1])

	source, err := manager.LookupSdk(name)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w", name, err)
	}
	resolvedVersion := manager.ResolveAlias(name, source, manager.ResolveVersion(name, version))
	cv := source.Current()
	if err = source.Uninstall(resolvedVersion); err != nil {
		return err
	}
	// An alias must not outlive the version it names.
	if removed, err := manager.RemoveAliasesFor(name, resolvedVersion); err != nil {
		pterm.Printf("%s\n", pterm.LightYellow(fmt.Sprintf("Could not update the aliases of %s@%s: %v", name, resolvedVersion, err)))
	} else if len(removed) > 0 {
		pterm.Printf("Removed the alias(es) %s of %s@%s.\n", strings.Join(removed, ", "), name, resolvedVersion)
	}
	remainVersion := source.InstalledList()
	if len(remainVersion) == 0 {
		_ = os.RemoveAll(source.Metadata().SdkInstalledPath)
		return nil
	}
	if cv == resolvedVersion {
		pterm.Println("Auto switch to the other version.")
		firstVersion := remainVersion[0]
		return source.Use(firstVersion, env.Global)
	}
	return nil
}
