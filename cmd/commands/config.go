package commands

import (
"context"
	"errors"
	"fmt"
	"github.com/version-fox/vfox/internal/config"
	"reflect"
	"strconv"
	"strings"
	"time"

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
			Usage:   "remove a config",
		},
	},
	Action: configCmd,
}

func configCmd(ctx context.Context, cmd *cli.Command) error {
	manager := internal.NewSdkManager()
	defer manager.Close()
	conf := reflect.ValueOf(manager.Config)

	if cmd.Bool("list") {
		configList("", conf)
		return nil
	}

	args := cmd.Args()
	if args.Len() == 0 {
		return cmd.Run(ctx, []string{"CMD", "config", "-h"})
	}

	keys := strings.Split(args.First(), ".")
	unset := cmd.Bool("unset")
	if !unset && args.Len() == 1 {
		configGet(conf, keys)
		return nil
	}

	var value any
	value = args.Get(1)
	if unset {
		value = defaultConfig(keys)
	}
	err := configSet(conf, keys, value)
	if err != nil {
		return err
	}
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

func configSet(v reflect.Value, keys []string, value any) error {
	key := keys[0]
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Tag.Get("yaml") == key {
			if len(keys) > 1 {
				err := configSet(v.Field(i), keys[1:], value)
				if err != nil {
					return err
				}
			} else {
				switch v.Field(i).Kind() {
				case reflect.String:
					v.Field(i).SetString(fmt.Sprintf("%v", value))
				case reflect.Bool:
					value := fmt.Sprintf("%v", value)
					parseBool, _ := strconv.ParseBool(value)
					v.Field(i).SetBool(parseBool)
				case reflect.Int64:
					value := fmt.Sprintf("%v", value)
					if v.Field(i).Type() == reflect.TypeOf(config.CacheDuration(0)) {
						if value == "-1" {
							v.Field(i).SetInt(-1)
						} else {
							duration, err := time.ParseDuration(strings.ToLower(value))
							if err != nil {
								return err
							}
							v.Field(i).SetInt(int64(duration))
						}
					} else {
						atoi, err := strconv.Atoi(value)
						if err != nil {
							return err
						}
						v.Field(i).SetInt(int64(atoi))
					}
				case reflect.Ptr:
					if _, ok := value.(string); ok {
						return fmt.Errorf("key does not contain a section: %v", key)
					}
					v.Field(i).Set(reflect.ValueOf(value))
				default:
					return errors.New("unsupported configuration type")
				}
			}
			break
		}
	}
	return nil
}

func defaultConfig(keys []string) any {
	v := reflect.ValueOf(config.DefaultConfig)
	for _, key := range keys {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).Tag.Get("yaml") == key {
				v = v.Field(i)
				break
			}
		}
	}
	return v.Interface()
}
