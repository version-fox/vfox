//go:build !windows

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
	"github.com/version-fox/vfox/internal/logger"
	"github.com/version-fox/vfox/internal/util"
	"os"
	"path/filepath"
)

// Clear removes the generated shim.
func (s *Shim) Clear() error {
	name := filepath.Base(s.BinaryPath)
	targetShim := filepath.Join(s.OutputPath, name)
	if !util.FileExists(targetShim) {
		return nil
	}
	return os.Remove(targetShim)
}

// Generate generates the shim.
func (s *Shim) Generate() error {
	if err := s.Clear(); err != nil {
		logger.Debugf("Clear shim failed: %s", err)
		return err
	}
	name := filepath.Base(s.BinaryPath)
	targetShim := filepath.Join(s.OutputPath, name)
	logger.Debugf("Create shim from %s to %s", s.BinaryPath, targetShim)
	if util.FileExists(targetShim) {
		_ = os.Remove(targetShim)
	}
	if err := os.Symlink(s.BinaryPath, targetShim); err != nil {
		logger.Debugf("Create symlink failed: %s", err)
		return err
	}
	return nil
}
