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

package util

import (
	"errors"
	"os/exec"
	"runtime"
)

var (
	// ErrClipboardNotSupported is returned when the OS doesn't support clipboard operations
	ErrClipboardNotSupported = errors.New("clipboard not supported on this OS")
	// ErrClipboardUtilityNotFound is returned when clipboard utility is not available
	ErrClipboardUtilityNotFound = errors.New("clipboard utility not found")
)

// CopyToClipboard copies the given text to the system clipboard
// Returns nil if successful, error otherwise
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel as fallback
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			// No clipboard utility available
			return ErrClipboardUtilityNotFound
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		// Unsupported OS
		return ErrClipboardNotSupported
	}

	if cmd == nil {
		return ErrClipboardNotSupported
	}

	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if _, err := in.Write([]byte(text)); err != nil {
		return err
	}

	if err := in.Close(); err != nil {
		return err
	}

	return cmd.Wait()
}
