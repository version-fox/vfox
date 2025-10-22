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

package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/cmd/commands"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/logger"
)

func Execute(args []string) {
	newCmd().Execute(args)
}

type cmd struct {
	app     *cli.Command
	version string
}

func (c *cmd) Execute(args []string) {
	if err := c.app.Run(context.Background(), args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newCmd() *cmd {
	version := internal.RuntimeVersion
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"v", "V"},
		Usage:   "print version",
		Action: func(ctx context.Context, cmd *cli.Command, b bool) error {
			println(version)
			return nil
		},
	}

	app := &cli.Command{}
	app.EnableShellCompletion = true
	app.Name = "vfox"
	app.Usage = "vfox is a tool for runtime version management."
	app.UsageText = "vfox [command] [command options]"
	app.Copyright = "Copyright 2025 Han Li. All rights reserved."
	app.Version = version
	app.Description = "vfox is a cross-platform version manager, extendable via plugins. It allows you to quickly install and switch between different environment you need via the command line."
	app.Suggest = true
	app.ShellComplete = func(ctx context.Context, cmd *cli.Command) {
		for _, command := range cmd.Commands {
			_, _ = fmt.Fprintln(cmd.Writer, command.Name)
		}
	}

	debugFlags := &cli.BoolFlag{
		Name:  "debug",
		Usage: "show debug information",
		Action: func(ctx context.Context, cmd *cli.Command, b bool) error {
			logger.SetLevel(logger.DebugLevel)
			return nil
		},
	}

	debugFd := &cli.IntFlag{
		Name:   "debug-fd",
		Usage:  "show debug information in file descriptor (1=stdout, 2=stderr)",
		Hidden: true,
		Action: func(ctx context.Context, cmd *cli.Command, fd int) error {
			logger.SetLevel(logger.DebugLevel)
			switch fd {
			case 1:
				logger.SetOutput(os.Stdout)
			case 2:
				logger.SetOutput(os.Stderr)
			default:
				return fmt.Errorf("unsupported file descriptor: %d, only 1 (stdout) and 2 (stderr) are supported", fd)
			}
			return nil
		},
	}

	app.Flags = []cli.Flag{
		debugFlags,
		debugFd,
	}
	app.Commands = []*cli.Command{
		commands.Info,
		commands.Install,
		commands.Current,
		commands.Use,
		commands.Unuse,
		commands.List,
		commands.Uninstall,
		commands.Available,
		commands.Search,
		commands.Update,
		commands.Upgrade,
		commands.Remove,
		commands.Add,
		commands.Activate,
		commands.Env,
		commands.Config,
		commands.Cd,
	}

	return &cmd{app: app, version: version}
}
