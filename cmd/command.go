/*
 *    Copyright 2024 [lihan aooohan@gmail.com]
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
	"strings"

	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/sdk"
)

func newInfo(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
		Name:  "info",
		Usage: "show sdk info",
		Action: func(ctx *cli.Context) error {
			args := ctx.Args().First()
			if args == "" {
				return cli.Exit("invalid arguments", 1)
			}
			return manager.Info(args)
		},
	}
}

func newAdd(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
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
	}
}

func newAvailable(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
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
	}
}

func newRemove(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
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
	}

}
func newUpdate(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
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
	}
}

func newInstall(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
		Name:    "install",
		Aliases: []string{"i"},
		Usage:   "install a version of sdk",
		Action:  sdkVersionParser(manager.Install),
	}
}

func newUninstall(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
		Name:    "uninstall",
		Aliases: []string{"un"},
		Usage:   "uninstall a version of sdk",
		Action:  sdkVersionParser(manager.Uninstall),
	}
}

func newSearch(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
		Name:  "search",
		Usage: "search a version of sdk",
		Action: func(ctx *cli.Context) error {
			sdkName := ctx.Args().First()
			return manager.Search(sdkName)
		},
	}
}

func newUse(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
		Name:    "use",
		Aliases: []string{"u"},
		Usage:   "use a version of sdk",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "global",
				Aliases: []string{"g"},
				Usage:   "used with the global environment",
			},
			&cli.StringFlag{
				Name:    "project",
				Aliases: []string{"p"},
				Usage:   "used with the current directory",
			},
			&cli.StringFlag{
				Name:    "session",
				Aliases: []string{"s"},
				Usage:   "used with the current shell session",
			},
		},
		Action: func(ctx *cli.Context) error {
			sdkArg := ctx.Args().First()
			arg := sdk.Arg{}
			if len(sdkArg) != 0 {
				argArr := strings.Split(sdkArg, "@")
				if len(argArr) <= 1 {
					arg.Name = argArr[0]
				} else {
					arg.Name = argArr[0]
					arg.Version = argArr[1]
				}
			}
			scope := sdk.Session
			if ctx.IsSet("global") {
				scope = sdk.Global
			} else if ctx.IsSet("project") {
				scope = sdk.Project
			} else {
				scope = sdk.Session
			}
			// TODO Consider how to handle exceptions appropriately. Print directly or return?
			_ = manager.Use(arg, scope)
			return nil
		},
	}
}

func newList(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "list all versions of the target sdk",
		Action: func(ctx *cli.Context) error {
			sdkName := ctx.Args().First()
			return manager.List(sdk.Arg{Name: sdkName})
		},
	}
}

func newCurrent(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
		Name:      "current",
		Aliases:   []string{"c"},
		Usage:     "show current version of the targeted sdk",
		UsageText: "show current version of all SDK's if no parameters are passed",
		Action: func(ctx *cli.Context) error {
			sdkName := ctx.Args().First()
			return manager.Current(sdkName)
		},
	}
}

func newActivate(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
		Name: "activate",
		Action: func(ctx *cli.Context) error {
			shellName := ctx.Args().First()
			if shellName == "" {
				return fmt.Errorf("shell name is required")
			}
			return manager.Activate(ctx.App.Writer, shellName)
		},
	}
}

func newEnv(manager *sdk.Manager) *cli.Command {
	return &cli.Command{
		Name: "env",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "shell",
				Aliases: []string{"s"},
				Usage:   "shell",
			},
		},
		Action: func(ctx *cli.Context) error {
			shellName := ctx.String("shell")
			return manager.Env(ctx.App.Writer, shellName)
		},
	}
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
