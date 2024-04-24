package commands

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal"
	"reflect"
	"strconv"
	"strings"
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
			Usage:   "remove a config",
		},
	},
	Action: configCmd,
}

func configCmd(ctx *cli.Context) error {
	manager := internal.NewSdkManager()
	defer manager.Close()
	config := reflect.ValueOf(manager.Config)

	if ctx.Bool("list") {
		configList("", config)
		return nil
	}

	args := ctx.Args()
	if args.Len() == 0 {
		return ctx.App.Run([]string{"CMD", "config", "-h"})
	}

	keys := strings.Split(args.First(), ".")
	unset := ctx.Bool("unset")
	if !unset && args.Len() == 1 {
		configGet(config, keys)
		return nil
	}

	value := args.Get(1)
	if unset {
		value = ""
	}
	configSet(config, keys, value)
	return manager.Config.SaveConfig(manager.PathMeta.HomePath)
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
			if value.Kind() == reflect.String && value.IsZero() {
				continue
			}
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

func configSet(v reflect.Value, keys []string, value string) {
	key := keys[0]
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Tag.Get("yaml") == key {
			if len(keys) > 1 {
				configSet(v.Field(i), keys[1:], value)
			} else {
				switch v.Field(i).Kind() {
				case reflect.Bool:
					parseBool, _ := strconv.ParseBool(value)
					v.Field(i).SetBool(parseBool)
				default:
					v.Field(i).SetString(value)
				}
			}
			break
		}
	}
}
