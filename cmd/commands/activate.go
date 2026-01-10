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
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/pathmeta"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/env/shell"
	"github.com/version-fox/vfox/internal/shared/logger"
)

var Activate = &cli.Command{
	Name:     "activate",
	Hidden:   true,
	Action:   activateCmd,
	Category: CategorySDK,
}

func activateCmd(ctx context.Context, cmd *cli.Command) error {
	name := cmd.Args().First()
	if name == "" {
		return cli.Exit("shell name is required", 1)
	}
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()

	// Load Project and Global configs
	chain, err := manager.RuntimeEnvContext.LoadVfoxTomlChainByScopes(env.Global, env.Project)
	if err != nil {
		return err
	}

	// Project > Global
	envs, err := manager.EnvKeys(chain)

	if err != nil {
		return err
	}

	runtimeEnvContext := manager.RuntimeEnvContext

	// Note: This step must be the first.
	// the Paths will handle the path format of GitBash which is different from other shells.
	// So we need to set the env.HookFlag first to let the Paths know
	// which shell we are using.
	_ = os.Setenv(env.HookFlag, name)
	// TODO：deprecated
	_ = os.Setenv(pathmeta.HookCurTmpPath, runtimeEnvContext.PathMeta.Working.SessionSdkDir)

	//threeLayerPaths := generatePATH(runtimeEnvContext.PathMeta)

	exportEnvs := make(env.Vars)
	for k, v := range envs.Variables {
		exportEnvs[k] = v
	}
	//pathStr := threeLayerPaths.String()
	//exportEnvs["PATH"] = &pathStr

	logger.Debugf("export envs: %+v", exportEnvs)

	path := runtimeEnvContext.PathMeta.Executable
	path = strings.Replace(path, "\\", "/", -1)
	s := shell.NewShell(name)
	if s == nil {
		return fmt.Errorf("unknown target shell %s", name)
	}

	exportStr := s.Export(exportEnvs)
	str, err := s.Activate(
		shell.ActivateConfig{
			SelfPath:       path,
			Args:           cmd.Args().Tail(),
			EnablePidCheck: env.IsMultiplexerEnvironment(),
		},
	)
	if err != nil {
		return err
	}
	hookTemplate, err := template.New("hook").Parse(str)
	if err != nil {
		return nil
	}
	tmpCtx := struct {
		SelfPath       string
		EnvContent     string
		EnablePidCheck bool
	}{
		SelfPath:       path,
		EnvContent:     exportStr,
		EnablePidCheck: env.IsMultiplexerEnvironment(),
	}
	return hookTemplate.Execute(cmd.Writer, tmpCtx)
}
