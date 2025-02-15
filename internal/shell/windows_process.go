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

	"github.com/StackExchange/wmi"
	"github.com/version-fox/vfox/internal/env"
)

type Win32_Process struct {
	ExecutablePath string
	CommandLine    string
	ProcessId      uint32
}

type windowsProcess struct{}

var process = windowsProcess{}

func GetProcess() Process {
	return process
}

func (w windowsProcess) Open(pid int) error {
	// Check if shell has hooks configured
	if !env.IsHookEnv() {
		return handleNoHookFallback(pid)
	}

	// Query WMI for process info
	var processes []Win32_Process
	query := fmt.Sprintf("SELECT ExecutablePath FROM Win32_Process WHERE ProcessId = %d", pid)
	if err := wmi.Query(query, &processes); err != nil {
		return fmt.Errorf("WMI query failed: %w", err)
	}

	if len(processes) == 0 {
		return fmt.Errorf("process with PID %d not found", pid)
	}

	// Get executable path
	path := processes[0].ExecutablePath
	if path == "" {
		return fmt.Errorf("executable path not found for PID %d", pid)
	}

	// Launch new shell process with proper environment
	cmd := exec.Command(path)
	cmd.Env = os.Environ()
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to launch shell: %w", err)
	}

	return nil
}

func handleNoHookFallback(pid int) error {
	// Fall back to global scope if no hooks
	fmt.Println("Warning: The current shell lacks hook support. Switching to global scope.")
	return nil
}
