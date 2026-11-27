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
	"encoding/json"
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

// postInstallLocksEnv lists the version locks of the installations whose
// PostInstall is running, in this process and in the vfox that started it, as a
// JSON array. An install that would wait for one of them runs inside that
// PostInstall, and the wait could only end when it times out. The list is JSON
// rather than a path list because a lock path can contain the path list separator,
// a ':' in VFOX_HOME on Unix or a ';' on Windows. Each path is stored as bytes,
// which encoding/json writes as base64, because it would replace bytes that are
// not valid UTF-8 in a string, and the path would no longer match.
const postInstallLocksEnv = "__VFOX_POST_INSTALL_LOCKS"

// versionLockPath is the lock held while version is being installed.
func (b *impl) versionLockPath(version Version) string {
	return filepath.Join(b.InstallPath, ".v-"+string(version)+".lock")
}

// withPostInstallLock runs postInstall with lockPath added to postInstallLocksEnv,
// so the commands it starts inherit it, and restores the variable afterwards.
func withPostInstallLock(lockPath string, postInstall func() error) error {
	old, had := os.LookupEnv(postInstallLocksEnv)
	var paths [][]byte
	for _, path := range append(postInstallLocks(), lockPath) {
		paths = append(paths, []byte(path))
	}
	// A list of byte slices always encodes.
	value, _ := json.Marshal(paths)
	if err := os.Setenv(postInstallLocksEnv, string(value)); err != nil {
		return fmt.Errorf("set %s: %w", postInstallLocksEnv, err)
	}
	defer func() {
		if had {
			_ = os.Setenv(postInstallLocksEnv, old)
		} else {
			_ = os.Unsetenv(postInstallLocksEnv)
		}
	}()
	return postInstall()
}

// postInstallLocks returns the locks listed in postInstallLocksEnv. A value that is
// not a JSON array of base64 strings was not set by vfox and lists none.
func postInstallLocks() []string {
	var paths [][]byte
	if err := json.Unmarshal([]byte(os.Getenv(postInstallLocksEnv)), &paths); err != nil {
		return nil
	}
	var locks []string
	for _, path := range paths {
		locks = append(locks, string(path))
	}
	return locks
}

// inPostInstallOf reports whether lockPath belongs to an installation whose
// PostInstall is running this one.
func inPostInstallOf(lockPath string) bool {
	for _, path := range postInstallLocks() {
		if filepath.Clean(path) == lockPath {
			return true
		}
	}
	return false
}
