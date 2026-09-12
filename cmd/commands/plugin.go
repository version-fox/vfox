/*
 *    Copyright 2026 Han Li and contributors
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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/version-fox/vfox/internal"
)

var Plugin = &cli.Command{
	Name:     "plugin",
	Usage:    "Test and debug a local plugin",
	Category: CategoryPlugin,
	Commands: []*cli.Command{
		{
			Name: "test", Usage: "Run Lua test files (offline by default)", ArgsUsage: "<directory>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "file", Usage: "Run one test file, relative to the plugin directory"},
				&cli.BoolFlag{Name: "online", Usage: "Allow HTTP requests when no response handler is configured"},
				&cli.DurationFlag{Name: "timeout", Value: 30 * time.Second, Usage: "Lua and HTTP deadline per file"},
			},
			Action: pluginTestCmd,
		},
		{
			Name: "run", Usage: "Invoke one hook (online by default)", ArgsUsage: "<directory> <hook>",
			Flags: []cli.Flag{
				&cli.StringFlag{Name: "input", Value: "{}", Usage: "Hook context as a JSON object"},
				&cli.StringFlag{Name: "input-file", Usage: "Read hook context from a JSON file"},
				&cli.BoolFlag{Name: "offline", Usage: "Reject HTTP requests"},
				&cli.StringFlag{Name: "os", Usage: "Override RUNTIME.osType (e.g. linux, darwin, windows)"},
				&cli.StringFlag{Name: "arch", Usage: "Override RUNTIME.archType (e.g. amd64, arm64)"},
				&cli.BoolFlag{Name: "json", Usage: "Write only the typed JSON result to stdout"},
				&cli.DurationFlag{Name: "timeout", Value: 60 * time.Second, Usage: "Lua and HTTP deadline"},
			},
			Action: pluginRunCmd,
		},
	},
}

func pluginTestCmd(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return fmt.Errorf("usage: vfox plugin test <directory> [options]")
	}
	if cmd.Duration("timeout") <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return internal.TestPlugin(ctx, cmd.Args().First(), internal.PluginTestOptions{
		File: cmd.String("file"), Online: cmd.Bool("online"), Timeout: cmd.Duration("timeout"),
		Output: cmd.Writer, Diagnostics: cmd.ErrWriter,
	})
}

func pluginRunCmd(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 2 {
		return fmt.Errorf("usage: vfox plugin run <directory> <hook> [options]")
	}
	if cmd.Duration("timeout") <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if cmd.IsSet("input") && cmd.IsSet("input-file") {
		return fmt.Errorf("use either --input or --input-file")
	}
	input := []byte(cmd.String("input"))
	if cmd.IsSet("input-file") {
		var err error
		input, err = os.ReadFile(cmd.String("input-file"))
		if err != nil {
			return fmt.Errorf("read hook input: %w", err)
		}
	}
	hook := cmd.Args().Get(1)
	result, err := internal.RunPluginHook(ctx, cmd.Args().First(), hook, input, internal.PluginRunOptions{
		OS: cmd.String("os"), Arch: cmd.String("arch"), Offline: cmd.Bool("offline"),
		Timeout: cmd.Duration("timeout"), Diagnostics: cmd.ErrWriter,
	})
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(cmd.Writer)
	if !cmd.Bool("json") {
		if _, err := fmt.Fprintf(cmd.Writer, "%s:\n", hook); err != nil {
			return err
		}
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(result)
}
