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
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/sdk"
)

var Link = &cli.Command{
	Name:      "link",
	Aliases:   []string{"ln"},
	Usage:     "Use an SDK that is already on this machine as a version",
	ArgsUsage: "<sdk>@<version> <path>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Point an already linked version to a different directory",
		},
	},
	Action:   linkCmd,
	Category: CategorySDK,
}

func linkCmd(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 2 {
		return cli.Exit("usage: vfox link <sdk>@<version> <path>", 1)
	}
	name, version, err := parseLinkSpec(cmd.Args().Get(0))
	if err != nil {
		return err
	}
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	source, err := manager.LookupSdk(name)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w (add its plugin first: vfox add %s)", name, err, name)
	}
	return source.Link(version, cmd.Args().Get(1), sdk.LinkOptions{Force: cmd.Bool("force")})
}

// parseLinkSpec splits <sdk>@<version>. The version is used exactly as written:
// it is a name the user picks, not something the plugin resolves.
func parseLinkSpec(spec string) (string, sdk.Version, error) {
	name, version, ok := strings.Cut(spec, "@")
	if !ok || name == "" || version == "" || strings.Contains(version, "@") {
		return "", "", fmt.Errorf("invalid parameter %q. format: <sdk-name>@<version>", spec)
	}
	return strings.ToLower(name), sdk.Version(version), nil
}
