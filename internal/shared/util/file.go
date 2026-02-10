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
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

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

// copyDir recursively copies a directory tree
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory with same permissions
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
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
		} else {
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
			// Preserve file permissions
			if info, err := os.Stat(srcPath); err == nil {
				os.Chmod(dstPath, info.Mode())
			}
		}
	}

	return nil
}

// Rename renames (moves) a file or directory, handling cross-drive operations on Windows.
// It first attempts a direct rename using os.Rename. If that fails (e.g., cross-drive on Windows),
// it falls back to copy-and-delete.
func Rename(src, dst string) error {
	// First try a direct rename (fast, works on same filesystem)
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// If rename failed, check if source exists
	srcInfo, statErr := os.Stat(src)
	if statErr != nil {
		return statErr
	}

	// Copy source to destination
	if srcInfo.IsDir() {
		if err := copyDir(src, dst); err != nil {
			return err
		}
	} else {
		if err := CopyFile(src, dst); err != nil {
			return err
		}
		// Preserve file permissions
		if err := os.Chmod(dst, srcInfo.Mode()); err != nil {
			return err
		}
	}

	// Remove source after successful copy
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

// MkSymlink Create a symbolic link
func MkSymlink(oldname, newname string) (err error) {
	if runtime.GOOS == "windows" {
		// Create a symbolic link on Windows
		// https://superuser.com/questions/1020821/how-can-i-create-a-symbolic-link-on-windows-10
		if err = exec.Command("cmd", "/c", "mklink", "/j", newname, oldname).Run(); err == nil {
			return nil
		}
	}
	return os.Symlink(oldname, newname)
}
