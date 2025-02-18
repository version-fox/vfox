//go:build windows

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
)

type windowsProcess struct{}

var process = windowsProcess{}

func GetProcess() Process {
	return process
}

func (w windowsProcess) Open(pid int) error {
	// On Windows, os.FindProcess does not actually find the process.
	// So, we use this workaround to get the parent process name.
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH", "/FO", "CSV")
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	cmd = exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid), "get", "ExecutablePath", "/format:list")
	output, err = cmd.Output()
	if err != nil {
		return err
	}
	path := strings.TrimPrefix(strings.TrimSpace(string(output)), "ExecutablePath=")
	command := exec.Command(path)
	command.Env = os.Environ()
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("open a new shell failed, err:%w", err)
	}
	return nil
}
