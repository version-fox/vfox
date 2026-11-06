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

	"golang.org/x/sys/unix"
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

	// install.sh moves the binary into /usr/local/bin with sudo, and it stays
	// owned by the user who unpacked it. The binary is writable, the directory
	// is not, so a check on the binary would pass here.
	t.Run("writable binary in a read-only directory", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root can write to a read-only directory")
		}
		dir := t.TempDir()
		exePath := filepath.Join(dir, "vfox")
		if err := os.WriteFile(exePath, []byte("vfox"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dir, 0555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(dir, 0755) })

		f, err := os.OpenFile(exePath, os.O_WRONLY, 0)
		if err != nil {
			t.Fatalf("the binary itself should be writable: %v", err)
		}
		f.Close()

		if err := checkUpgradeDirWritable(dir, exePath); err == nil {
			t.Fatal("expected an error, the directory is read-only")
		}
	})

	// Replacing the binary only needs the directory, so its own mode does not matter.
	t.Run("read-only binary in a writable directory", func(t *testing.T) {
		dir := t.TempDir()
		exePath := filepath.Join(dir, "vfox")
		if err := os.WriteFile(exePath, []byte("vfox"), 0555); err != nil {
			t.Fatal(err)
		}
		if err := checkUpgradeDirWritable(dir, exePath); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	// The root directory is root:root 755, like /usr/local/bin: the owner write
	// bit is set, but this user is not the owner. t.TempDir() is always ours,
	// so it can't show this.
	t.Run("directory owned by someone else", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root can write anywhere")
		}
		if unix.Access("/", unix.W_OK) == nil {
			t.Skip("/ is writable by this user")
		}
		if err := checkUpgradeDirWritable("/", "/vfox"); err == nil {
			t.Fatal("expected an error for a directory this user does not own")
		}
	})
}
