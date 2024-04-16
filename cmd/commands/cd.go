package commands

import (
	"fmt"
	"github.com/twpayne/go-shell"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
	"os"
	"os/exec"
)

var CD = &cli.Command{
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
	userShell, _ := shell.CurrentUserShell()
	cmd := exec.Command(userShell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	manager := internal.NewSdkManager()
	if ctx.Args().Len() == 0 {
		cmd.Dir = manager.PathMeta.HomePath
	} else {
		sdkName := ctx.Args().First()
		sdk, err := manager.LookupSdk(sdkName)
		if err != nil {
			return err
		}
		if ctx.Bool("plugin") {
			cmd.Dir = sdk.Plugin.Path
		} else {
			current := sdk.Current()
			if current == "" {
				return fmt.Errorf("no current version of %s", sdkName)
			}
			sdkPackage, err := sdk.GetLocalSdkPackage(current)
			if err != nil {
				return err
			}
			cmd.Dir = sdkPackage.Main.Path
		}
	}
	manager.Close()

	return cmd.Run()
}
