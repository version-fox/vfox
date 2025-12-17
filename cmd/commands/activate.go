/*
 *    Copyright 2025 Han Li and contributors
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

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/shell"
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
		// Try to auto-detect shell if not provided
		if detected := shell.GetShellName(); detected != "" {
			name = detected
			fmt.Fprintf(os.Stderr, "Warning: No shell specified, auto-detected: %s\n", name)
			fmt.Fprintf(os.Stderr, "To avoid this warning, specify shell explicitly:\n")
			fmt.Fprintf(os.Stderr, "  eval \"$(vfox activate %s)\"\n\n", name)
		} else {
			fmt.Fprintf(os.Stderr, "Error: shell name is required and auto-detection failed\n")
			fmt.Fprintf(os.Stderr, "\nUsage:\n")
			fmt.Fprintf(os.Stderr, "  eval \"$(vfox activate <shell>)\"\n\n")
			fmt.Fprintf(os.Stderr, "Examples:\n")
			fmt.Fprintf(os.Stderr, "  eval \"$(vfox activate zsh)\"    # For Zsh\n")
			fmt.Fprintf(os.Stderr, "  eval \"$(vfox activate bash)\"   # For Bash\n")
			fmt.Fprintf(os.Stderr, "  eval \"$(vfox activate fish)\"   # For Fish\n\n")
			fmt.Fprintf(os.Stderr, "Configuration:\n")
			fmt.Fprintf(os.Stderr, "  Add this line to your ~/.zshrc, ~/.bashrc, or ~/.config/fish/config.fish:\n")
			fmt.Fprintf(os.Stderr, "    eval \"$(vfox activate <shell>)\"\n\n")
			fmt.Fprintf(os.Stderr, "Note: Place this in your shell's interactive config file, not .zprofile\n")
			return cli.Exit("", 1)
		}
	}
	manager := internal.NewSdkManager()
	defer manager.Close()

	sdkEnvs, err := manager.GlobalEnvKeys()
	if err != nil {
		return err
	}

	// Note: This step must be the first.
	// the Paths will handle the path format of GitBash which is different from other shells.
	// So we need to set the env.HookFlag first to let the Paths know
	// which shell we are using.
	_ = os.Setenv(env.HookFlag, name)

	exportEnvs := sdkEnvs.ToExportEnvs()

	// Current shell is not for user use, it will be killed after the activation.
	// If we setup hook in this shell, it will cause all terminal launched by IDE could not be hooked properly.
	if !env.IsIDEEnvironmentResolution() {
		exportEnvs[env.HookFlag] = &name
		exportEnvs[internal.HookCurTmpPath] = &manager.PathMeta.CurTmpPath
	}

	logger.Debugf("export envs: %+v", exportEnvs)

	path := manager.PathMeta.ExecutablePath
	path = strings.Replace(path, "\\", "/", -1)
	s := shell.NewShell(name)
	if s == nil {
		return fmt.Errorf("unknown target shell %s", name)
	}

	exportStr := s.Export(exportEnvs)
	str, err := s.Activate(
		shell.ActivateConfig{
			SelfPath: path,
			Args:     cmd.Args().Tail(),
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
		SelfPath   string
		EnvContent string
	}{
		SelfPath:   path,
		EnvContent: exportStr,
	}
	return hookTemplate.Execute(cmd.Writer, tmpCtx)
}
