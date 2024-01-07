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

package commands

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shell"
	"html/template"
	"os"
	"strings"
)

var Activate = &cli.Command{
	Name:   "activate",
	Hidden: true,
	Action: activateCmd,
}

func activateCmd(ctx *cli.Context) error {
	name := ctx.Args().First()
	if name == "" {
		return cli.Exit("shell name is required", 1)
	}
	manager := sdk.NewSdkManager()
	defer manager.Close()
	temp, err := sdk.NewTemp(manager.TempPath, os.Getppid())
	if err != nil {
		return fmt.Errorf("create temp file failed: %w", err)
	}
	// Clean up the old temp files, before today.
	go temp.RemoveOldFile()
	record, err := env.NewRecord(manager.ConfigPath)
	if err != nil {
		return err
	}
	envKeys, err := manager.EnvKeys(record)
	path := manager.ExecutablePath
	path = strings.Replace(path, "\\", "/", -1)
	tmpCtx := struct {
		SelfPath string
	}{
		SelfPath: path,
	}
	s := shell.NewShell(name)
	if s == nil {
		return fmt.Errorf("unknow target shell %s", name)
	}
	exportStr := s.Export(envKeys)
	str, err := s.Activate()
	if err != nil {
		return err
	}
	script := exportStr + "\n" + str
	hookTemplate, err := template.New("hook").Parse(script)
	if err != nil {
		return nil
	}
	return hookTemplate.Execute(ctx.App.Writer, tmpCtx)
}
