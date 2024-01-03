//go:build windows

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

package shell

import (
	"fmt"
	"github.com/pterm/pterm"
	"os"
	"os/exec"
	"strings"
)

func (i *Shell) ReOpen() error {
	command := exec.Command(i.ShellPath)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		pterm.Printf("Failed to start shell, err:%s\n", err.Error())
		return err
	}
	return nil
}

func NewShell() (*Shell, error) {
	ppid := os.Getppid()

	// On Windows, os.FindProcess does not actually find the process.
	// So, we use this workaround to get the parent process name.
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", ppid), "/NH", "/FO", "CSV")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	fields := strings.Split(string(output), ",")
	parentProcessName := strings.Trim(fields[0], "\" ")
	cmd = exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", ppid), "get", "ExecutablePath", "/format:list")
	output, err = cmd.Output()
	if err != nil {
		return nil, err
	}
	path := strings.TrimPrefix(strings.TrimSpace(string(output)), "ExecutablePath=")
	return &Shell{
		Type:       Type(parentProcessName),
		ShellPath:  path,
		ConfigPath: "",
	}, nil
}
