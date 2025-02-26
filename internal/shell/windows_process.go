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

	"golang.org/x/sys/windows"
)

type windowsProcess struct{}

var process = windowsProcess{}

func GetProcess() Process {
	return process
}

func (w windowsProcess) Open(pid int) error {
	hProcess, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return err
	}
	defer windows.CloseHandle(hProcess)

	var exePath [windows.MAX_PATH]uint16
	size := uint32(len(exePath))
	if err := windows.QueryFullProcessImageName(hProcess, 0, &exePath[0], &size); err != nil {
		return err
	}

	path := windows.UTF16ToString(exePath[:size])
	command := exec.Command(path)
	command.Env = os.Environ()
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	fmt.Println(path)
	fmt.Println(command.String())
	if err := command.Run(); err != nil {
		return fmt.Errorf("open a new shell failed, err:%w", err)
	}
	return nil
}
