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

package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/cmd/commands"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/logger"
)

func Execute(args []string) {
	newCmd().Execute(args)
}

type cmd struct {
	app     *cli.App
	version string
}

func (c *cmd) Execute(args []string) {
	if err := c.app.Run(args); err != nil {
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
		Action: func(ctx *cli.Context, b bool) error {
			println(version)
			return nil
		},
	}

	app := &cli.App{}
	app.EnableBashCompletion = true
	app.Name = "vfox"
	app.Usage = "vfox is a tool for runtime version management."
	app.UsageText = "vfox [command] [command options]"
	app.Copyright = "Copyright 2024 Han Li. All rights reserved."
	app.Version = version
	app.Description = "vfox is a cross-platform version manager, extendable via plugins. It allows you to quickly install and switch between different environment you need via the command line."
	app.Suggest = true
	app.BashComplete = func(ctx *cli.Context) {
		for _, command := range ctx.App.Commands {
			_, _ = fmt.Fprintln(ctx.App.Writer, command.Name)
		}
	}

	debugFlags := &cli.BoolFlag{
		Name:  "debug",
		Usage: "show debug information",
		Action: func(ctx *cli.Context, b bool) error {
			logger.SetLevel(logger.DebugLevel)
			return nil
		},
	}

	app.Flags = []cli.Flag{
		debugFlags,
	}
	app.Commands = []*cli.Command{
		commands.Info,
		commands.Install,
		commands.Current,
		commands.Use,
		commands.List,
		commands.Uninstall,
		commands.Available,
		commands.Search,
		commands.Update,
		commands.Remove,
		commands.Add,
		commands.Activate,
		commands.Env,
		commands.Config,
	}

	return &cmd{app: app, version: version}
}
