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
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// CreateDirSymlink creates a directory junction on Windows systems
// Junctions don't require administrator privileges
func CreateDirSymlink(target, link string) error {
	// Ensure target is an absolute path
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}

	// Ensure link parent directory exists
	linkParent := filepath.Dir(link)
	if err := os.MkdirAll(linkParent, 0755); err != nil {
		return err
	}

	// If link already exists, remove it first
	if _, err := os.Stat(link); err == nil {
		if err := os.RemoveAll(link); err != nil {
			return err
		}
	}

	// Convert to UTF-16
	targetPtr, err := windows.UTF16PtrFromString(targetAbs)
	if err != nil {
		return err
	}
	linkPtr, err := windows.UTF16PtrFromString(link)
	if err != nil {
		return err
	}

	// Create the junction using golang.org/x/sys/windows
	return windows.CreateJunction(targetPtr, linkPtr)
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
