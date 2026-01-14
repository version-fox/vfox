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

	// Build arguments string from os.Args (skip the executable name)
	// This ensures all flags like --debug are passed through
	args := ""
	if len(os.Args) > 1 {
		// Join all arguments starting from index 1
		for i, arg := range os.Args[1:] {
			if i > 0 {
				args += " "
			}
			// Quote arguments that contain spaces
			if len(arg) > 0 && (arg[0] == '-' || !containsSpace(arg)) {
				args += arg
			} else {
				args += "\"" + arg + "\""
			}
		}
	}

	verb := "runas"
	cwd, _ := syscall.UTF16PtrFromString(".")
	arg, _ := syscall.UTF16PtrFromString(args)
	run := windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteW")
	ret, _, _ := run.Call(
		0,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(verb))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(exePath))),
		uintptr(unsafe.Pointer(arg)),
		uintptr(unsafe.Pointer(cwd)),
		1,
	)
	// ShellExecuteW returns a value > 32 on success
	if ret <= 32 {
		return syscall.Errno(ret)
	}
	os.Exit(0)
	return nil
}

func containsSpace(s string) bool {
	for _, c := range s {
		if c == ' ' {
			return true
		}
	}
	return false
}
