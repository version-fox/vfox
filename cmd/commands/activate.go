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
	"github.com/version-fox/vfox/internal/toolset"
	"os"
	"strings"
	"text/template"

	"github.com/version-fox/vfox/internal"

	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/shell"
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
	manager := internal.NewSdkManager()
	defer manager.Close()
	tvs, err := toolset.NewMultiToolVersions([]string{
		manager.PathMeta.HomePath,
		manager.PathMeta.WorkingDirectory,
	})
	if err != nil {
		return err
	}
	envKeys, err := manager.EnvKeys(tvs)
	if err != nil {
		return err
	}
	exportEnvs := make(env.Vars)
	for k, v := range envKeys.Variables {
		exportEnvs[k] = v
	}

	_ = os.Setenv(env.HookFlag, name)
	exportEnvs[env.HookFlag] = &name
	allPaths := envKeys.Paths.Merge(env.NewPaths(env.OsPaths))
	pathStr := allPaths.String()
	exportEnvs["PATH"] = &pathStr

	// filter vfox sdk path
	homePath := manager.PathMeta.HomePath
	previousPaths := env.NewPaths(env.EmptyPaths)
	for _, p := range allPaths.Slice() {
		if strings.HasPrefix(p, homePath) {
			previousPaths.Add(p)
		}
	}
	prePathsStr := previousPaths.String()
	exportEnvs[env.PreviousPathsFlag] = &prePathsStr

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
