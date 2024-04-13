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
	"path/filepath"
)

func FileSave(file string, data []byte) error {
	err := os.WriteFile(file, data, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

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
