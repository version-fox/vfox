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

package commands

import (
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/logger"
	"strconv"
)

var example string = `
Example: 
config --unset-all
config --unset proxy.url
config --unset proxy.enable
config --unset storage.sdk-path
config --unset registry.address

config proxy.url http://x.com
config proxy.url ""

config proxy.enable true
config proxy.enable false

config storage.sdk-path D:/app/vfox/sdk
config storage.sdk-path ""

config registry.address http://x.com
config registry.address ""
`

var Config = &cli.Command{
	Name:        "config",
	Usage:       "Config operation, list, storage etc.",
	Description: example,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "unset",
			Usage: "unset someone config",
		},
		&cli.BoolFlag{
			Name:  "unset-all",
			Usage: "unset all config",
		},
	},
	Subcommands: []*cli.Command{
		{
			Name:    "list",
			Aliases: []string{"ls"},
			Usage:   "list all config info",
			Action:  ConfigListCmd,
		},
		{
			Name:   "path",
			Usage:  "show the config location",
			Action: ConfigPathCmd,
		},
		{
			Name:   "proxy.enable",
			Usage:  "Update proxy enable, true or false",
			Action: proxyEnableCmd,
		},
		{
			Name:   "proxy.url",
			Usage:  "Update proxy url",
			Action: proxyUrlCmd,
		},
		{
			Name:   "storage.sdk-path",
			Usage:  "storage sdk path",
			Action: storageSdkPathCmd,
		},
		{
			Name:   "registry.address",
			Usage:  "Update registry address",
			Action: registryAddressCmd,
		},
	},
	Category: ConfigPlugin,
	Action:   configCmd,
}

func configCmd(ctx *cli.Context) error {
	names := ctx.FlagNames()

	manager := internal.NewSdkManager()
	if names != nil && names[0] == "unset" {
		value := ctx.String("unset")

		switch value {
		case "proxy.enable":
			manager.Config.Proxy.Enable = config.EmptyProxy.Enable
		case "proxy.url":
			manager.Config.Proxy.Url = config.EmptyProxy.Url
		case "storage.sdk-path":
			manager.Config.Storage.SdkPath = config.EmptyStorage.SdkPath
		case "registry.address":
			manager.Config.Registry.Address = config.EmptyRegistry.Address
		}

		config.SaveConfig(manager.Config)
	} else if names != nil && names[0] == "unset-all" {
		manager.Config.Proxy = config.EmptyProxy
		manager.Config.Storage = config.EmptyStorage
		manager.Config.Registry = config.EmptyRegistry
		config.SaveConfig(manager.Config)
	} else {
		logger.Error("vfox config, you do not input any command or flag!")
	}
	return nil
}

// ConfigListCmd config list all
func ConfigListCmd(ctx *cli.Context) error {
	ctx.Args().First()

	manager := internal.NewSdkManager()
	defer manager.Close()
	manager.PrintConfigInfo()
	return nil
}

func ConfigPathCmd(ctx *cli.Context) error {
	ctx.Args().First()

	manager := internal.NewSdkManager()
	defer manager.Close()

	logger.Info(manager.PathMeta.HomePath)
	return nil
}

func proxyEnableCmd(ctx *cli.Context) error {
	args := ctx.Args()
	l := args.Len()
	if l < 1 {
		return cli.Exit("invalid arguments", 1)
	}

	manager := internal.NewSdkManager()
	defer manager.Close()

	var err error
	if args.First() != "" {
		enable, _ := strconv.ParseBool(args.First())
		err = manager.UpdateConfigProxyEnable(enable)
	}

	if err != nil {
		return err
	}

	return err
}

func proxyUrlCmd(ctx *cli.Context) error {
	args := ctx.Args()
	l := args.Len()
	if l < 1 {
		return cli.Exit("invalid arguments", 1)
	}

	manager := internal.NewSdkManager()
	defer manager.Close()

	err := manager.UpdateConfigProxyUrl(args.First())
	if err != nil {
		return err
	}

	return err
}

// storageSdkPathCmd config such as storage sdkPath
func storageSdkPathCmd(ctx *cli.Context) error {
	args := ctx.Args()
	l := args.Len()
	if l < 1 {
		return cli.Exit("invalid arguments", 1)
	}

	manager := internal.NewSdkManager()
	defer manager.Close()

	var err error
	if args.First() != "" {
		err = manager.UpdateConfigStorageSdkPath(args.First())
	}

	if err != nil {
		return err
	}

	return err
}

func registryAddressCmd(ctx *cli.Context) error {
	args := ctx.Args()
	l := args.Len()
	if l < 1 {
		return cli.Exit("invalid arguments", 1)
	}

	manager := internal.NewSdkManager()
	defer manager.Close()

	var err error
	err = manager.UpdateRegistryAddress(args.First())

	if err != nil {
		return err
	}

	return err
}
