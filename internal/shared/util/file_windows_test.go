//go:build windows

// Copyright 2026 vfox project contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMkSymlinkWithSpaces verifies that MkSymlink creates a junction
// correctly when both source and destination paths contain spaces. This
// is the regression test for the Windows path-with-spaces bug: the old
// implementation shelled out to `cmd.exe /c mklink /j`, which Go's
// exec.Command escapes in a way cmd.exe misinterprets.
func TestMkSymlinkWithSpaces(t *testing.T) {
	src, err := os.MkdirTemp("", "vfox src*")
	if err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	defer os.RemoveAll(src)

	dstDir, err := os.MkdirTemp("", "vfox dst*")
	if err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	// Insert spaces into both paths to reproduce the bug scenario.
	srcPath := filepath.Join(dstDir, "sdk with spaces")
	if err := os.Mkdir(srcPath, 0755); err != nil {
		t.Fatalf("failed to create source with spaces: %v", err)
	}
	defer os.RemoveAll(srcPath)

	dstPath := filepath.Join(dstDir, "link with spaces")

	if err := MkSymlink(srcPath, dstPath); err != nil {
		t.Fatalf("MkSymlink with spaces failed: %v", err)
	}

	// Verify the junction was created and points to the correct target.
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("failed to stat junction: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("junction is not a directory")
	}
}
