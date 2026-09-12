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

package sdk

import (
	"fmt"
	"os"
	"path/filepath"
)

// acquireInstallLock serializes attempts across SDK instances and processes.
// The file is kept after Close: unlinking it could let another process lock a
// different inode. Closing the descriptor (including process exit) releases it.
func acquireInstallLock(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create installation root: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("open installation lock: %w", err)
	}
	if err := lockInstallFile(file); err != nil {
		_ = file.Close() // No lock was acquired; preserve the locking error.
		return nil, fmt.Errorf("installation is in progress or cannot be locked: %w", err)
	}
	return file, nil
}
