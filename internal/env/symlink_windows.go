//go:build windows

/*
 *
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
 *
 */

package env

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

// CreateDirSymlink creates a directory junction on Windows systems
// Junctions don't require administrator privileges
func CreateDirSymlink(target, link string) error {
	// Ensure target is an absolute path
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("failed to resolve target path: %w", err)
	}

	// Ensure link parent directory exists
	linkParent := filepath.Dir(link)
	if err := os.MkdirAll(linkParent, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// If link already exists, remove it first
	// Use Lstat instead of Stat to check the link itself, not its target
	if fileInfo, err := os.Lstat(link); err == nil {
		// Remove existing file/directory/symlink
		if fileInfo.IsDir() {
			if err := os.RemoveAll(link); err != nil {
				return fmt.Errorf("failed to remove existing directory: %w", err)
			}
		} else {
			if err := os.Remove(link); err != nil {
				return fmt.Errorf("failed to remove existing file: %w", err)
			}
		}
	}

	// Create the directory for the junction
	if err := os.Mkdir(link, 0777); err != nil {
		return fmt.Errorf("failed to create junction directory: %w", err)
	}

	success := false
	defer func() {
		if !success {
			// Clean up on failure
			os.Remove(link)
		}
	}()

	// Open the directory with GENERIC_WRITE access
	linkPtr, err := windows.UTF16PtrFromString(link)
	if err != nil {
		return fmt.Errorf("failed to convert link path: %w", err)
	}

	handle, err := windows.CreateFile(
		linkPtr,
		windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		// FILE_FLAG_OPEN_REPARSE_POINT tells Windows we want the junction itself, not its target
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return fmt.Errorf("failed to open directory: %w", err)
	}
	defer windows.CloseHandle(handle)

	// Create reparse point data
	rp := winio.ReparsePoint{
		Target:       targetAbs,
		IsMountPoint: true,
	}

	data := winio.EncodeReparsePoint(&rp)

	// Set the reparse point
	var bytesReturned uint32
	err = windows.DeviceIoControl(
		handle,
		windows.FSCTL_SET_REPARSE_POINT,
		&data[0],
		uint32(len(data)),
		nil,
		0,
		&bytesReturned,
		nil,
	)

	if err != nil {
		return fmt.Errorf("failed to set reparse point: %w", err)
	}

	success = true
	return nil
}

// RemoveDirSymlink removes a directory junction on Windows systems
func RemoveDirSymlink(link string) error {
	return os.RemoveAll(link)
}

// IsDirSymlink checks if the given path is a directory junction or symlink
func IsDirSymlink(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	// Check for reparse point (junction or symlink)
	return fi.Mode()&os.ModeSymlink != 0
}

// ReadDirSymlink reads the target of a directory junction or symlink
func ReadDirSymlink(link string) (string, error) {
	return os.Readlink(link)
}
