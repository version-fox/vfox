//go:build darwin || linux

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
)

// CreateDirSymlink creates a directory symbolic link on Unix systems
func CreateDirSymlink(target, link string) error {
	// If link already exists, remove it first
	if _, err := os.Lstat(link); err == nil {
		if err := os.Remove(link); err != nil {
			return err
		}
	}
	return os.Symlink(target, link)
}

// RemoveDirSymlink removes a directory symbolic link on Unix systems
func RemoveDirSymlink(link string) error {
	return os.Remove(link)
}

// IsDirSymlink checks if the given path is a directory symbolic link
func IsDirSymlink(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0
}

// ReadDirSymlink reads the target of a directory symbolic link
func ReadDirSymlink(link string) (string, error) {
	return os.Readlink(link)
}
