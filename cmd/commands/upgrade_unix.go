//go:build !windows

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

package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

// RequestPermission checks that the directory holding the vfox binary is writable.
// The upgrade stages the download there and renames the new binary over the old one,
// so it needs write access to the directory, not just to the binary.
func RequestPermission() error {
	exePath, err := os.Executable()
	if err != nil {
		return cli.Exit("Failed to get executable path: "+err.Error(), 1)
	}
	return checkUpgradeDirWritable(filepath.Dir(exePath), exePath)
}

// checkUpgradeDirWritable creates and removes a file in dir, which also catches
// read-only mounts and ACLs that the directory's mode bits would not show.
func checkUpgradeDirWritable(dir, exePath string) error {
	probe, err := os.CreateTemp(dir, ".vfox_upgrade_check_*")
	if err != nil {
		reason := err
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			reason = pathErr.Err
		}
		msg := fmt.Sprintf("Cannot upgrade vfox: %s is not writable (%v).", dir, reason)
		// sudo only helps with permissions, not with a read-only filesystem
		if errors.Is(err, fs.ErrPermission) {
			msg += fmt.Sprintf("\nRe-run the upgrade with write access to it, for example: sudo %s upgrade", exePath)
		}
		return cli.Exit(msg, 1)
	}
	probe.Close()
	return os.Remove(probe.Name())
}
