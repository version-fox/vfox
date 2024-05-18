/*
 *    Copyright 2024 Han Li and contributors
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
	fileInfo, err := os.Lstat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		return false
	}

	if fileInfo.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(filename)
		if err != nil {
			return false
		}
		_, err = os.Stat(target)
		if err != nil {
			if os.IsNotExist(err) {
				return false
			}
			return false
		}
		return true
	}

	return true
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
