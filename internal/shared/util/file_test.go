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

package util

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestMovePathFallsBackOnCrossDeviceRename(t *testing.T) {
	oldRenamePath := renamePath
	renamePath = func(oldPath, newPath string) error {
		return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: syscall.EXDEV}
	}
	t.Cleanup(func() {
		renamePath = oldRenamePath
	})

	srcDir := filepath.Join(t.TempDir(), "src")
	dstDir := filepath.Join(t.TempDir(), "dst")
	nestedDir := filepath.Join(srcDir, "nested")

	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("failed to create source directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "main.lua"), []byte("print('hello')\n"), 0644); err != nil {
		t.Fatalf("failed to write root file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "config.txt"), []byte("ok"), 0600); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	if err := MovePath(srcDir, dstDir); err != nil {
		t.Fatalf("MovePath returned error: %v", err)
	}

	if _, err := os.Stat(srcDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected source to be removed, got err=%v", err)
	}

	content, err := os.ReadFile(filepath.Join(dstDir, "main.lua"))
	if err != nil {
		t.Fatalf("failed to read moved root file: %v", err)
	}
	if string(content) != "print('hello')\n" {
		t.Fatalf("unexpected root file content: %q", content)
	}

	info, err := os.Stat(filepath.Join(dstDir, "nested", "config.txt"))
	if err != nil {
		t.Fatalf("failed to stat moved nested file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("expected nested file mode 0600, got %o", info.Mode().Perm())
	}
}

func TestMovePathReturnsNonCrossDeviceErrors(t *testing.T) {
	oldRenamePath := renamePath
	renamePath = func(oldPath, newPath string) error {
		return &os.LinkError{Op: "rename", Old: oldPath, New: newPath, Err: os.ErrPermission}
	}
	t.Cleanup(func() {
		renamePath = oldRenamePath
	})

	srcFile := filepath.Join(t.TempDir(), "plugin.lua")
	dstFile := filepath.Join(t.TempDir(), "plugin-copy.lua")

	if err := os.WriteFile(srcFile, []byte("print('hello')\n"), 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	err := MovePath(srcFile, dstFile)
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected permission error, got %v", err)
	}

	if _, statErr := os.Stat(srcFile); statErr != nil {
		t.Fatalf("expected source file to remain in place, got %v", statErr)
	}
	if _, statErr := os.Stat(dstFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected destination file to be absent, got %v", statErr)
	}
}
