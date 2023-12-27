/*
 *    Copyright 2023 [lihan aooohan@gmail.com]
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

package main

import (
	"github.com/pterm/pterm"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/sdk"
)

const Version = "0.0.1"

func main() {
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"v", "V"},
		Usage:   "print version",
		Action: func(ctx *cli.Context, b bool) error {
			println(Version)
			return nil
		},
	}

	manager := sdk.NewSdkManager()
	defer manager.Close()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)

	go func() {
		<-signals
		manager.Close()
		os.Exit(0)
	}()

	app := &cli.App{
		EnableBashCompletion: true,
	}
	app.Name = "VersionFox"
	app.Usage = "VersionFox is a tool for sdk version management"
	app.UsageText = "vfox [command] [command options]"
	app.Copyright = "Copyright 2023 Han Li. All rights reserved."
	app.Version = Version
	app.Description = "VersionFox is a tool for sdk version management, which allows you to quickly install and use different versions of targeted sdk via the command line."
	app.Suggest = true
	app.Commands = []*cli.Command{
		{
			Name:  "info",
			Usage: "show info of plugin",
			Action: func(ctx *cli.Context) error {
				args := ctx.Args().First()
				if args == "" {
					return cli.Exit("invalid arguments", 1)
				}
				return manager.Info(args)
			},
		},
		{
			Name:  "add",
			Usage: "add a plugin of sdk",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "source",
					Aliases: []string{"s"},
					Usage:   "plugin source",
				},
				&cli.StringFlag{
					Name:  "alias",
					Usage: "plugin alias",
				},
			},
			Action: func(ctx *cli.Context) error {
				sdkName := ctx.Args().First()
				source := ctx.String("source")
				alias := ctx.String("alias")
				return manager.Add(sdkName, source, alias)
			},
		},
		{
			Name:  "available",
			Usage: "show available plugins",
			Action: func(ctx *cli.Context) error {
				categoryName := ctx.Args().First()
				categories, err := manager.Available()
				if err != nil {
					return err
				}
				data := pterm.TableData{
					{"NAME", "VERSION", "AUTHOR", "DESCRIPTION"},
				}
				for _, category := range categories {
					if len(categoryName) > 0 {
						if categoryName != category.Name {
							continue
						}
					}
					for _, p := range category.Plugins {
						desc := p.Desc
						if len(desc) == 0 {
							desc = "-"
						} else if len(desc) > 100 {
							desc = desc[:100] + "..."
						}
						data = append(data, []string{category.Name + "/" + p.Filename, p.Version, p.Author, desc})
					}
				}

				_ = pterm.DefaultTable.
					WithHasHeader().
					WithSeparator("\t ").
					WithData(data).Render()
				pterm.Printf("Please use %s to install plugin\n", pterm.LightBlue("vfox add <plugin name>"))
				return nil
			},
		},
		{
			Name:  "remove",
			Usage: "remove a plugin of sdk",
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()
				l := args.Len()
				if l < 1 {
					return cli.Exit("invalid arguments", 1)
				}
				return manager.Remove(args.First())
			},
		},
		{
			Name:  "update",
			Usage: "update specified plug-ins",
			Action: func(ctx *cli.Context) error {
				args := ctx.Args()
				l := args.Len()
				if l < 1 {
					return cli.Exit("invalid arguments", 1)
				}
				return manager.Update(args.First())
			},
		},

		{
			Name:    "install",
			Aliases: []string{"i"},
			Usage:   "install a version of sdk",
			Action:  sdkVersionParser(manager.Install),
		},
		{
			Name:    "uninstall",
			Aliases: []string{"un"},
			Usage:   "uninstall a version of sdk",
			Action:  sdkVersionParser(manager.Uninstall),
		},
		{
			Name:  "search",
			Usage: "search a version of sdk",
			Action: func(ctx *cli.Context) error {
				sdkName := ctx.Args().First()
				return manager.Search(sdkName)
			},
		},
		{
			Name:    "use",
			Aliases: []string{"u"},
			Usage:   "use a version of sdk",
			Action:  sdkVersionParser(manager.Use),
		},
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "list all versions of the target sdk",
			Action: func(ctx *cli.Context) error {
				sdkName := ctx.Args().First()
				return manager.List(sdk.Arg{Name: sdkName})
			},
		},
		{
			Name:      "current",
			Aliases:   []string{"c"},
			Usage:     "show current version of the targeted sdk",
			UsageText: "show current version of all SDK's if no parameters are passed",
			Action: func(ctx *cli.Context) error {
				sdkName := ctx.Args().First()
				return manager.Current(sdkName)
			},
		},
	}

	_ = app.Run(os.Args)
}

func sdkVersionParser(operation func(arg sdk.Arg) error) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		sdkArg := ctx.Args().First()
		if sdkArg == "" {
			return cli.Exit("sdk name is required", 1)
		}
		argArr := strings.Split(sdkArg, "@")
		argsLen := len(argArr)
		if argsLen > 2 {
			return cli.Exit("sdk version is invalid", 1)
		} else if argsLen == 2 {
			return operation(sdk.Arg{
				Name:    strings.ToLower(argArr[0]),
				Version: argArr[1],
			})
		} else {
			return operation(sdk.Arg{
				Name:    strings.ToLower(argArr[0]),
				Version: "",
			})
		}
	}
}
