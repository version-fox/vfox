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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

var renamePath = os.Rename

func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	err = dstFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

// MovePath moves a file or directory to the target path.
// If a cross-device rename fails, it falls back to copy and remove.
func MovePath(src, dst string) error {
	if err := renamePath(src, dst); err == nil {
		return nil
	} else if !isCrossDeviceRenameError(err) {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if err := copyDir(src, dst); err != nil {
			return err
		}
	} else {
		if err := copySingleFile(src, dst, info.Mode()); err != nil {
			return err
		}
	}

	return os.RemoveAll(src)
}

// MoveFiles Move a folder or file to a specified directory
func MoveFiles(src, targetDir string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		files, err := os.ReadDir(src)
		if err != nil {
			return err
		}

		for _, file := range files {
			oldPath := filepath.Join(src, file.Name())
			newPath := filepath.Join(targetDir, file.Name())
			err = os.Rename(oldPath, newPath)
			if err != nil {
				return err
			}
		}
	} else {
		newPath := filepath.Join(targetDir, filepath.Base(src))
		err = os.Rename(src, newPath)
		if err != nil {
			return err
		}
	}
	return nil
}

// ChangeModeIfNot Change the permission mode of a file if it is not the same as the specified mode
func ChangeModeIfNot(src string, mode os.FileMode) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.Mode() != mode {
		err = os.Chmod(src, mode)
		if err != nil {
			return err
		}
	}
	return nil
}

// IsExecutable Check if a file is executable
func IsExecutable(src string) bool {
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(src))
		return ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".ps1"
	} else {
		info, err := os.Stat(src)
		if err != nil {
			return false
		}
		return info.Mode()&0111 != 0
	}
}

// MkSymlink Create a symbolic link. On Windows this uses PowerShell's
// New-Item -ItemType Junction, invoked via -EncodedCommand, so paths
// containing spaces, quotes, or backslashes survive the cmd-line layer.
// (The previous implementation shelled out to `cmd.exe /c mklink /j`,
// which Go's exec.Command escapes in a way cmd.exe misinterprets.)
func MkSymlink(oldname, newname string) (err error) {
	if !FileExists(oldname) {
		return fmt.Errorf("source directory '%s' does not exist", oldname)
	}
	if FileExists(newname) {
		return fmt.Errorf("destination '%s' already exists", newname)
	}
	if IsWindows() {
		script := fmt.Sprintf("New-Item -ItemType Junction -Path %s -Target %s -Force",
			psSingleQuote(newname), psSingleQuote(oldname))
		out, err := RunPowerShellScript(script)
		if err != nil {
			return fmt.Errorf("failed to create junction '%s' -> '%s': %w (%s)", newname, oldname, err, strings.TrimSpace(out))
		}
		return nil
	}
	err = os.Symlink(oldname, newname)
	if err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	return nil
}

func isCrossDeviceRenameError(err error) bool {
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		err = linkErr.Err
	}

	if errors.Is(err, syscall.EXDEV) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "cross-device link") ||
		strings.Contains(msg, "different disk drive") ||
		strings.Contains(msg, "not same device")
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}

		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if err := copySingleFile(srcPath, dstPath, fileInfo.Mode()); err != nil {
			return err
		}
	}

	return nil
}

func copySingleFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if err := CopyFile(src, dst); err != nil {
		return err
	}
	return ChangeModeIfNot(dst, mode)
}
