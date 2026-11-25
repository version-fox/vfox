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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// acquireInstallLock serializes attempts across SDK instances and processes.
// The file is kept after Close: unlinking it could let another process lock a
// different inode. Closing the descriptor (including process exit) releases it.
func acquireInstallLock(path string) (*os.File, error) {
	file, err := openInstallLock(path)
	if err != nil {
		return nil, err
	}
	if err := lockInstallFile(file); err != nil {
		_ = file.Close() // No lock was acquired; preserve the locking error.
		return nil, fmt.Errorf("installation is in progress or cannot be locked: %w", err)
	}
	return file, nil
}

func openInstallLock(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create installation root: %w", err)
	}
	// The file is created writable by the group as far as the umask allows, so that
	// members of a group sharing VFOX_HOME can open it. One that this user cannot
	// write, as another user may have created it, is opened for reading: flock and
	// LockFileEx lock a descriptor that is open for reading too, and nothing is
	// written to the file.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if errors.Is(err, fs.ErrPermission) {
		file, err = os.Open(path)
	}
	if err != nil {
		return nil, fmt.Errorf("open installation lock: %w", err)
	}
	return file, nil
}

// errInstallLockStillHeld is returned by waitForInstallLock when the lock is still
// held by another installation after installLockWait.
var errInstallLockStillHeld = errors.New("another installation still holds the lock")

// How long waitForInstallLock waits for another installation, and how often it
// tries the lock meanwhile. Variables so that tests can shorten them.
var (
	installLockWait = 10 * time.Minute
	installLockPoll = 100 * time.Millisecond
)

// waitForInstallLock is acquireInstallLock for a lock that another installation
// holds while it runs: it waits up to installLockWait for that installation, and
// calls waiting once when it starts to wait. Only a lock held by someone else is
// waited for; any other error is returned at once.
func waitForInstallLock(path string, waiting func()) (*os.File, error) {
	file, err := openInstallLock(path)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(installLockWait)
	for {
		err := lockInstallFile(file)
		if err == nil {
			return file, nil
		}
		if !installLockHeld(err) {
			_ = file.Close() // No lock was acquired; preserve the locking error.
			return nil, fmt.Errorf("lock installation: %w", err)
		}
		if time.Now().After(deadline) {
			_ = file.Close()
			return nil, errInstallLockStillHeld
		}
		if waiting != nil {
			waiting()
			waiting = nil
		}
		time.Sleep(installLockPoll)
	}
}

// sdkInstallLockName is the lock in an SDK's installation root that lets one
// installation of the SDK run at a time. Like the lock of each version, it stays
// in place after the installation.
const sdkInstallLockName = ".install.lock"
