package commands

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/shell"
	"os"
)

var Cd = &cli.Command{
	Name:  "cd",
	Usage: "Launch a shell in the VFOX_HOME or SDK directory",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "plugin",
			Aliases: []string{"p"},
			Usage:   "Launch a shell in the plugin directory",
		},
	},
	Action: cdCmd,
}

func cdCmd(ctx *cli.Context) error {
	var dir string

	manager := internal.NewSdkManager()
	if ctx.Args().Len() == 0 {
		dir = manager.PathMeta.HomePath
	} else {
		sdkName := ctx.Args().First()
		sdk, err := manager.LookupSdk(sdkName)
		if err != nil {
			return err
		}
		if ctx.Bool("plugin") {
			dir = sdk.Plugin.Path
		} else {
			current := sdk.Current()
			if current == "" {
				return fmt.Errorf("no current version of %s", sdkName)
			}
			sdkPackage, err := sdk.GetLocalSdkPackage(current)
			if err != nil {
				return err
			}
			dir = sdkPackage.Main.Path
		}
	}
	manager.Close()

	err := os.Chdir(dir)
	if err != nil {
		return err
	}
	return shell.GetProcess().Open(env.GetPid())
}
