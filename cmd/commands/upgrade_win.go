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
	"strings"
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
		quotedArgs := make([]string, 0, len(os.Args)-1)
		for _, arg := range os.Args[1:] {
			quotedArgs = append(quotedArgs, escapeArg(arg))
		}
		args = strings.Join(quotedArgs, " ")
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
	// Exit the current process since we've successfully launched the elevated one
	// This function never returns normally after successful elevation
	os.Exit(0)
	return nil // unreachable but required for compilation
}

// escapeArg escapes a command-line argument according to Windows rules.
// Based on https://docs.microsoft.com/en-us/archive/blogs/twistylittlepassagesallalike/everyone-quotes-command-line-arguments-the-wrong-way
func escapeArg(arg string) string {
	// If the argument doesn't contain special characters, return as-is
	if !strings.ContainsAny(arg, " \t\n\"") {
		return arg
	}

	// Build the escaped argument
	var b strings.Builder
	b.WriteByte('"')

	for i := 0; i < len(arg); {
		// Count consecutive backslashes
		backslashes := 0
		for i < len(arg) && arg[i] == '\\' {
			backslashes++
			i++
		}

		if i >= len(arg) {
			// Backslashes at the end need to be doubled (they precede the closing quote)
			b.WriteString(strings.Repeat("\\", backslashes*2))
			break
		}

		if arg[i] == '"' {
			// Backslashes before a quote need to be doubled, and the quote needs to be escaped
			b.WriteString(strings.Repeat("\\", backslashes*2))
			b.WriteString("\\\"")
			i++
		} else {
			// Regular backslashes (not before a quote) are literal
			b.WriteString(strings.Repeat("\\", backslashes))
			b.WriteByte(arg[i])
			i++
		}
	}

	b.WriteByte('"')
	return b.String()
}
