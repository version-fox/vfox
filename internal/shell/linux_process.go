//go:build linux

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
	"os"
	"os/exec"
	"strconv"
)

type linuxProcess struct{}

var process = linuxProcess{}

func GetProcess() Process {
	return process
}

func (u linuxProcess) Open(pid int) error {
	//shellPath := os.Getenv("SHELL")
	path, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
	if err != nil {
		return fmt.Errorf("open a new shell failed, err:%w", err)
	}
	command := exec.Command(path)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("open a new shell failed, err:%w", err)
	}
	return nil
}
