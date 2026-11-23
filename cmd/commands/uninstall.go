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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
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
	resolvedVersion := manager.ResolveVersionName(name, source, manager.ResolveVersion(name, version))
	// If the global scope asks for this version, Uninstall switches it to the
	// first of these that can be used, after its checks and before it removes
	// anything. A version only a project or session asks for is left alone.
	fallbacks := fallbackVersions(source, otherVersions(source.InstalledList(), resolvedVersion))
	uninstallErr := source.Uninstall(resolvedVersion, fallbacks...)
	if uninstallErr != nil && !errors.Is(uninstallErr, sdk.ErrGlobalNotUnset) {
		return uninstallErr
	}
	// An alias must not outlive the version it names.
	if removed, err := manager.RemoveAliasesFor(name, resolvedVersion); err != nil {
		pterm.Printf("%s\n", pterm.LightYellow(fmt.Sprintf("Could not update the aliases of %s@%s: %v", name, resolvedVersion, err)))
	} else if len(removed) > 0 {
		pterm.Printf("Removed the alias(es) %s of %s@%s.\n", strings.Join(removed, ", "), name, resolvedVersion)
	}
	if len(source.InstalledList()) == 0 {
		_ = os.RemoveAll(source.Metadata().SdkInstalledPath)
	}
	return uninstallErr
}

// otherVersions returns the entries of versions that are not version.
func otherVersions(versions []sdk.Version, version sdk.Version) []sdk.Version {
	var others []sdk.Version
	for _, v := range versions {
		if v != version {
			others = append(others, v)
		}
	}
	return others
}

// fallbackVersions orders the installed versions that may take over when the
// version in use is uninstalled: installations of a version first, then copies
// made with install --as, which were made for one purpose. Within each, the
// order of remaining is kept.
func fallbackVersions(s sdk.Sdk, remaining []sdk.Version) []sdk.Version {
	var versions, copies []sdk.Version
	for _, v := range remaining {
		if inst, err := s.Installation(v); err == nil && inst.Receipt.IsCopy() {
			copies = append(copies, v)
		} else {
			versions = append(versions, v)
		}
	}
	return append(versions, copies...)
}
