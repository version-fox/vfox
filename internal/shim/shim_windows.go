//go:build windows

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

package shim

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/util"
)

const shimFileContent = `
path = "%s"
`

const cmdShimContent = `
@echo off
call "%s" %%*
`
const ps1ShimContent = `
param (
    [Parameter(Position=0, ValueFromRemainingArguments=$true)]
    $params
)

& '%s' $params
`

//go:embed binary/shim.exe
var shim []byte

// Clear removes the generated shim.
func (s *Shim) Clear() (err error) {
	filename := filepath.Base(s.BinaryPath)
	ext := filepath.Ext(filename)
	shimName := filename[:len(filename)-len(ext)] + ".shim"
	shimBinary := filepath.Join(s.OutputPath, filename)

	if util.FileExists(shimBinary) {
		if err = os.Remove(shimBinary); err != nil {
			return
		}
	}
	shimFile := filepath.Join(s.OutputPath, shimName)
	if util.FileExists(shimFile) {
		if err = os.Remove(shimFile); err != nil {
			return
		}
	}
	return nil
}

// Generate generates the shim.
func (s *Shim) Generate() error {
	if err := s.Clear(); err != nil {
		logger.Debugf("Clear shim failed: %s", err)
		return err
	}
	filename := filepath.Base(s.BinaryPath)
	stat, err := os.Stat(s.BinaryPath)
	if err != nil {
		return err
	}
	targetPath := filepath.Join(s.OutputPath, filename)
	ext := filepath.Ext(filename)
	if ext == ".cmd" {
		if err = os.WriteFile(targetPath, []byte(fmt.Sprintf(cmdShimContent, s.BinaryPath)), stat.Mode()); err != nil {
			return fmt.Errorf("failed to gnerate shim: %w", err)
		}
		return nil
	} else if ext == ".ps1" {
		if err = os.WriteFile(targetPath, []byte(fmt.Sprintf(ps1ShimContent, s.BinaryPath)), stat.Mode()); err != nil {
			return fmt.Errorf("failed to gnerate shim: %w", err)
		}
		return nil
	}
	logger.Debugf("Write shim binary to %s", targetPath)
	if err = os.WriteFile(targetPath, shim, stat.Mode()); err != nil {
		return fmt.Errorf("failed to gnerate shim: %w", err)
	}
	shimName := filename[:len(filename)-len(ext)] + ".shim"
	shimFile := filepath.Join(s.OutputPath, shimName)
	logger.Debugf("Write shim file to %s", shimFile)
	if err = os.WriteFile(shimFile, []byte(fmt.Sprintf(shimFileContent, s.BinaryPath)), stat.Mode()); err != nil {
		return fmt.Errorf("failed to gnerate shim: %w", err)
	}
	return nil
}
