//go:build darwin || linux

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
	"github.com/pterm/pterm"
	"os"
	"os/exec"
)

type unixProcess struct{}

var process = unixProcess{}

func GetProcess() Process {
	return process
}

func (u unixProcess) Open(shell Shell) error {
	//shellPath := os.Getenv("SHELL")
	command := exec.Command(shell.Name())
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		pterm.Printf("Failed to start shell, err:%s\n", err.Error())
		return err
	}
	return nil
}
