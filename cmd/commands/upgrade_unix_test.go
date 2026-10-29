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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckUpgradeDirWritable(t *testing.T) {
	t.Run("writable directory", func(t *testing.T) {
		dir := t.TempDir()
		if err := checkUpgradeDirWritable(dir, filepath.Join(dir, "vfox")); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected the probe file to be removed, found %v", entries)
		}
	})

	t.Run("read-only directory", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root can write to a read-only directory")
		}
		dir := t.TempDir()
		if err := os.Chmod(dir, 0555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(dir, 0755) })
		exePath := filepath.Join(dir, "vfox")

		err := checkUpgradeDirWritable(dir, exePath)
		if err == nil {
			t.Fatal("expected an error for a read-only directory")
		}
		for _, want := range []string{dir + " is not writable", "sudo " + exePath + " upgrade"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("expected the error to contain %q, got %q", want, err.Error())
			}
		}
	})
}
