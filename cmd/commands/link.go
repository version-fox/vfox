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
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v3"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/util"
)

var Link = &cli.Command{
	Name:      "link",
	Aliases:   []string{"ln"},
	Usage:     "Use an SDK that is already on this machine as a version",
	ArgsUsage: "<sdk>@<version> <path> | --detect <sdk>",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "force",
			Aliases: []string{"f"},
			Usage:   "Point an already linked version to a different directory",
		},
		&cli.BoolFlag{
			Name:  "detect",
			Usage: "Let the plugin look for SDKs on this machine and link what it finds",
		},
		&cli.BoolFlag{
			Name:    "yes",
			Aliases: []string{"y"},
			Usage:   "With --detect, link everything found without asking",
		},
	},
	Action:   linkCmd,
	Category: CategorySDK,
}

func linkCmd(ctx context.Context, cmd *cli.Command) error {
	if cmd.Bool("detect") {
		return linkDetected(cmd)
	}
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

func linkDetected(cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return cli.Exit("usage: vfox link --detect <sdk>", 1)
	}
	name := strings.ToLower(cmd.Args().First())
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	source, err := manager.LookupSdk(name)
	if err != nil {
		return fmt.Errorf("%s not supported, error: %w (add its plugin first: vfox add %s)", name, err, name)
	}
	found, err := source.Detect()
	if errors.Is(err, sdk.ErrDetectNotSupported) {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err != nil {
		return err
	}

	candidates, skipped := detectCandidates(source, found)
	for _, line := range skipped {
		pterm.Println(pterm.LightYellow("Skipping " + line))
	}
	if len(candidates) == 0 {
		pterm.Println("No new " + name + " installations found.")
		return nil
	}
	pterm.Println("Found:")
	for _, candidate := range candidates {
		line := fmt.Sprintf("  %s@%s  %s", name, candidate.Version, candidate.Path)
		if candidate.Note != "" {
			line += "  (" + candidate.Note + ")"
		}
		pterm.Println(line)
	}

	if !cmd.Bool("yes") {
		if util.IsNonInteractiveTerminal() {
			return cli.Exit("Use the -y flag to link what was found in non-interactive environments", 1)
		}
		confirmed, _ := pterm.DefaultInteractiveConfirm.
			WithDefaultValue(true).
			Show(fmt.Sprintf("Link %d %s installation(s)?", len(candidates), name))
		if !confirmed {
			return nil
		}
	}

	var failed []string
	for _, candidate := range candidates {
		if err := source.Link(candidate.Version, candidate.Path, sdk.LinkOptions{}); err != nil {
			pterm.Println(pterm.Red(fmt.Sprintf("Could not link %s@%s: %v", name, candidate.Version, err)))
			failed = append(failed, string(candidate.Version))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to link %s", strings.Join(failed, ", "))
	}
	return nil
}

// detectCandidates drops what is already registered: a name that is taken, or a
// directory some version already links to. It explains each omission.
func detectCandidates(s sdk.Sdk, found []sdk.DetectedRuntime) (candidates []sdk.DetectedRuntime, skipped []string) {
	linkedTargets := make(map[string]sdk.Version)
	for _, version := range s.InstalledList() {
		if installation, err := s.Installation(version); err == nil && installation.Target != "" {
			linkedTargets[filepath.Clean(installation.Target)] = version
		}
	}
	for _, candidate := range found {
		if s.CheckRuntimeExist(candidate.Version) {
			skipped = append(skipped, fmt.Sprintf("%s: a version of that name is already installed", candidate.Version))
			continue
		}
		if version, linked := linkedTargets[filepath.Clean(candidate.Path)]; linked {
			skipped = append(skipped, fmt.Sprintf("%s: %s is already linked as %s", candidate.Version, candidate.Path, version))
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates, skipped
}
