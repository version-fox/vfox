//go:build windows

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
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func RequestPermission() error {
	isAdmin, err := isAdmin()
	if err != nil {
		return err
	}

	if !isAdmin {
		if err = runAsAdmin(); err != nil {
			return err
		}
	}
	return nil
}

func isAdmin() (bool, error) {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err != nil {
		if os.IsPermission(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func runAsAdmin() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	verb := "runas"
	cwd, err := syscall.UTF16PtrFromString(".")
	if err != nil {
		return err
	}
	arg, err := syscall.UTF16PtrFromString(SelfUpgradeName)
	if err != nil {
		return err
	}
	run := windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")
	ret, _, callErr := run.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(verb))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(exePath))),
		uintptr(unsafe.Pointer(arg)),
		uintptr(unsafe.Pointer(cwd)),
		1,
	)
	if err := validateElevationLaunchResult(ret); err != nil {
		if callErr != windows.ERROR_SUCCESS {
			return fmt.Errorf("%w: %v", err, callErr)
		}
		return err
	}
	os.Exit(0)
	return nil
}

func validateElevationLaunchResult(code uintptr) error {
	if code > 32 {
		return nil
	}

	switch code {
	case 0:
		return fmt.Errorf("failed to request administrator privileges: out of memory")
	case 2:
		return fmt.Errorf("failed to request administrator privileges: file not found")
	case 3:
		return fmt.Errorf("failed to request administrator privileges: path not found")
	case 5:
		return fmt.Errorf("failed to request administrator privileges: access denied or elevation was canceled")
	case 8:
		return fmt.Errorf("failed to request administrator privileges: not enough memory")
	case 32:
		return fmt.Errorf("failed to request administrator privileges: DLL not found")
	default:
		return fmt.Errorf("failed to request administrator privileges: ShellExecuteW returned code %d", code)
	}
}
