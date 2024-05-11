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
	"fmt"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/shim"
	"os"
	"strings"
	"text/template"

	"github.com/version-fox/vfox/internal/toolset"

	"github.com/version-fox/vfox/internal"

	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/shell"
)

var Activate = &cli.Command{
	Name:     "activate",
	Hidden:   true,
	Action:   activateCmd,
	Category: CategorySDK,
}

func activateCmd(ctx *cli.Context) error {
	name := ctx.Args().First()
	if name == "" {
		return cli.Exit("shell name is required", 1)
	}
	manager := internal.NewSdkManager()
	defer manager.Close()

	workToolVersion, err := toolset.NewToolVersion(manager.PathMeta.WorkingDirectory)
	if err != nil {
		return err
	}

	if err = manager.ParseLegacyFile(func(sdkname, version string) {
		if _, ok := workToolVersion.Record[sdkname]; !ok {
			workToolVersion.Record[sdkname] = version
		}
	}); err != nil {
		return err
	}
	homeToolVersion, err := toolset.NewToolVersion(manager.PathMeta.HomePath)
	if err != nil {
		return err
	}
	envKeys, err := manager.EnvKeys(toolset.MultiToolVersions{
		workToolVersion,
		homeToolVersion,
	})
	if err != nil {
		return err
	}
	exportEnvs := make(env.Vars)
	for k, v := range envKeys.Variables {
		exportEnvs[k] = v
	}

	// generate shims for current shell
	if envKeys.Paths.Len() > 0 {
		logger.Debugf("Generate shims for current shell, path: %s\n", manager.PathMeta.ShellShimsPath)
		bins := envKeys.Paths.ToBinPaths()
		for _, bin := range bins.Slice() {
			binShim := shim.NewShim(bin, manager.PathMeta.ShellShimsPath)
			if err = binShim.Generate(); err != nil {
				continue
			}
		}
	}

	_ = os.Setenv(env.HookFlag, name)
	exportEnvs[env.HookFlag] = &name
	osPaths := env.NewPaths(env.OsPaths)
	osPaths.AddWithIndex(0, manager.PathMeta.ShellShimsPath)
	osPathsStr := osPaths.String()
	exportEnvs["PATH"] = &osPathsStr

	path := manager.PathMeta.ExecutablePath
	path = strings.Replace(path, "\\", "/", -1)
	s := shell.NewShell(name)
	if s == nil {
		return fmt.Errorf("unknow target shell %s", name)
	}
	exportStr := s.Export(exportEnvs)
	str, err := s.Activate()
	if err != nil {
		return err
	}
	hookTemplate, err := template.New("hook").Parse(str)
	if err != nil {
		return nil
	}
	tmpCtx := struct {
		SelfPath   string
		EnvContent string
	}{
		SelfPath:   path,
		EnvContent: exportStr,
	}
	return hookTemplate.Execute(ctx.App.Writer, tmpCtx)
}
