// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// msix-runner runs the packaged CLI without the CI runner's elevation.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func runCommand(cmd *exec.Cmd) error {
	cmd.Env = os.Environ()
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func normalUserToken(token windows.Token) (windows.Token, error) {
	if linked, err := token.GetLinkedToken(); err == nil {
		return linked, nil
	}
	// Hosted runners may have no split UAC token. SAFER removes Administrator
	// and Power User rights while retaining the same user and logon session.
	advapi := windows.NewLazySystemDLL("advapi32.dll")
	var level windows.Handle
	const saferScopeUser, saferNormalUser, saferLevelOpen = 2, 0x20000, 1
	ok, _, err := advapi.NewProc("SaferCreateLevel").Call(
		saferScopeUser, saferNormalUser, saferLevelOpen, uintptr(unsafe.Pointer(&level)), 0)
	if ok == 0 {
		return 0, fmt.Errorf("create normal-user SAFER level: %w", err)
	}
	defer advapi.NewProc("SaferCloseLevel").Call(uintptr(level))
	var restricted windows.Token
	ok, _, err = advapi.NewProc("SaferComputeTokenFromLevel").Call(
		uintptr(level), 0, uintptr(unsafe.Pointer(&restricted)), 0, 0)
	if ok == 0 {
		return 0, fmt.Errorf("create normal-user token: %w", err)
	}
	medium, err := windows.CreateWellKnownSid(windows.WinMediumLabelSid)
	if err == nil {
		label := windows.Tokenmandatorylabel{Label: windows.SIDAndAttributes{
			Sid: medium, Attributes: windows.SE_GROUP_INTEGRITY,
		}}
		err = windows.SetTokenInformation(restricted, windows.TokenIntegrityLevel,
			(*byte)(unsafe.Pointer(&label)), label.Size())
	}
	if err != nil {
		restricted.Close()
		return 0, fmt.Errorf("set medium integrity: %w", err)
	}
	return restricted, nil
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: msix-runner <command> [arguments...]")
	}
	token := windows.GetCurrentProcessToken()
	if os.Args[1] == "--child" {
		if token.IsElevated() {
			return errors.New("test child still has an elevated token")
		}
		if len(os.Args) < 3 {
			return errors.New("missing child command")
		}
		if os.Args[2] == "--check-token" {
			fmt.Println("Non-elevated child token verified")
			return nil
		}
		return runCommand(exec.Command(os.Args[2], os.Args[3:]...))
	}

	child := exec.Command(os.Args[0], append([]string{"--child"}, os.Args[1:]...)...)
	if token.IsElevated() {
		linked, err := normalUserToken(token)
		if err != nil {
			return err
		}
		defer linked.Close()
		if linked.IsElevated() {
			return errors.New("normal-user token is elevated")
		}
		child.SysProcAttr = &syscall.SysProcAttr{Token: syscall.Token(linked)}
	}
	return runCommand(child)
}

func main() {
	if err := run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
