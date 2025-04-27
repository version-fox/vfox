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

package shell

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/logger"
)

type ActivateConfig struct {
	SelfPath string
	Args     []string
}

type Shell interface {
	// Activate generates a shell script to be placed in the shell's configuration file, which will set up initial
	// environment variables and set a hook to update the environment variables when needed.
	Activate(config ActivateConfig) (string, error)

	// Export generates a string that can be used by the shell to set or unset the given environment variables. (The
	// input specifies environment variables to be unset by giving them a nil value.)
	Export(envs env.Vars) string
}

func NewShell(name string) Shell {
	switch strings.ToLower(name) {
	case "bash":
		return Bash
	case "zsh":
		return Zsh
	case "pwsh":
		return Pwsh
	case "fish":
		return Fish
	case "clink":
		return Clink
	case "nushell":
		return Nushell
	}
	return nil
}

func Open(pid int) error {
	logger.Debugf("open a new shell: %d\n", pid)
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return fmt.Errorf("open a new shell failed, err:%w", err)
	}

	cmdSlice, err := p.CmdlineSlice()
	if err != nil {
		return fmt.Errorf("open a new shell failed, err:%w", err)
	}

	if len(cmdSlice) == 0 {
		return fmt.Errorf("open a new shell failed, err: cannot find the command of the process: %d", pid)
	}

	// dev case
	if len(cmdSlice) > 1 && cmdSlice[0] == "go" && cmdSlice[1] == "run" {
		return fmt.Errorf("You are running the command in development mode, please use the binary file instead")
	}

	logger.Debugf("open a new shell: %s\n", strings.Join(cmdSlice, " "))
	command := exec.Command(cmdSlice[0], cmdSlice[1:]...)
	command.Env = os.Environ()
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("open a new shell failed, err:%w", err)
	}
	return nil
}
