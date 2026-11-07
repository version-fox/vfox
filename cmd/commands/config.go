package commands

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/version-fox/vfox/internal/config"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
)

var Config = &cli.Command{
	Name:  "config",
	Usage: "Setup, view config",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "list all config",
		},
		&cli.BoolFlag{
			Name:    "unset",
			Aliases: []string{"un"},
			Usage:   "remove a config from your config file",
		},
	},
	Action: configCmd,
}

func configCmd(ctx context.Context, cmd *cli.Command) error {
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()
	runtimeEnvContext := manager.RuntimeEnvContext
	conf := reflect.ValueOf(runtimeEnvContext.UserConfig)

	if cmd.Bool("list") {
		configList("", conf)
		return nil
	}

	args := cmd.Args()
	if args.Len() == 0 {
		return cli.ShowSubcommandHelp(cmd)
	}

	key := args.First()
	unset := cmd.Bool("unset")
	if !unset && args.Len() == 1 {
		configGet(conf, strings.Split(key, "."))
		return nil
	}

	// What is listed and read above is the merged config. What is written is
	// the user's own file, and in it only the key that is set or unset: the
	// merged config holds the defaults and the shared values too, and writing
	// them would turn them into the user's own.
	file, err := config.LoadFile(runtimeEnvContext.PathMeta.User.Config)
	if err != nil {
		return err
	}
	if unset {
		removed, err := file.Unset(key)
		if err != nil {
			return err
		}
		if !removed {
			return nil // the file does not have the key, so there is nothing to remove
		}
	} else if err := file.Set(key, args.Get(1)); err != nil {
		return err
	}
	return file.Save()
}

func configList(prefix string, v reflect.Value) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for i := 0; i < v.NumField(); i++ {
		key := v.Type().Field(i).Tag.Get("yaml")
		value := v.Field(i)
		if (value.Kind() == reflect.Ptr && value.Elem().Kind() == reflect.Struct) || value.Kind() == reflect.Struct {
			configList(prefix+key+".", value)
		} else {
			fmt.Printf(prefix+key+" = %v\n", value.Interface())
		}
	}
}

func configGet(v reflect.Value, keys []string) {
	var foundCount int
	for _, key := range keys {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).Tag.Get("yaml") == key {
				v = v.Field(i)
				foundCount = foundCount + 1
				break
			}
		}
	}
	if foundCount == len(keys) {
		if (v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Struct) || v.Kind() == reflect.Struct {
			configList(strings.Join(keys, ".")+".", v)
		} else {
			fmt.Printf("%v\n", v.Interface())
		}
	}
}
