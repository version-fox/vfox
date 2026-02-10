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
	"os"
	"path/filepath"
	"testing"
)

func TestRenameFile(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "vfox-rename-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	srcFile := filepath.Join(tmpDir, "test-src.txt")
	content := "test content"
	if err := os.WriteFile(srcFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test renaming file
	dstFile := filepath.Join(tmpDir, "test-dst.txt")
	if err := Rename(srcFile, dstFile); err != nil {
		t.Fatalf("Rename failed: %v", err)
	}

	// Verify source no longer exists
	if FileExists(srcFile) {
		t.Error("Source file still exists after rename")
	}

	// Verify destination exists with correct content
	if !FileExists(dstFile) {
		t.Error("Destination file does not exist")
	}
	readContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}
	if string(readContent) != content {
		t.Errorf("Content mismatch: expected %q, got %q", content, string(readContent))
	}
}

func TestRenameDirectory(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "vfox-rename-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test directory structure
	srcDir := filepath.Join(tmpDir, "test-src-dir")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create nested structure
	subDir := filepath.Join(srcDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	file1 := filepath.Join(srcDir, "file1.txt")
	file2 := filepath.Join(subDir, "file2.txt")
	content1 := "content1"
	content2 := "content2"

	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Test renaming directory
	dstDir := filepath.Join(tmpDir, "test-dst-dir")
	if err := Rename(srcDir, dstDir); err != nil {
		t.Fatalf("Rename directory failed: %v", err)
	}

	// Verify source no longer exists
	if FileExists(srcDir) {
		t.Error("Source directory still exists after rename")
	}

	// Verify destination exists with correct structure
	if !FileExists(dstDir) {
		t.Error("Destination directory does not exist")
	}

	// Check file1
	dstFile1 := filepath.Join(dstDir, "file1.txt")
	if !FileExists(dstFile1) {
		t.Error("file1.txt does not exist in destination")
	}
	readContent1, err := os.ReadFile(dstFile1)
	if err != nil {
		t.Fatalf("Failed to read file1: %v", err)
	}
	if string(readContent1) != content1 {
		t.Errorf("Content mismatch in file1: expected %q, got %q", content1, string(readContent1))
	}

	// Check file2 in subdirectory
	dstFile2 := filepath.Join(dstDir, "subdir", "file2.txt")
	if !FileExists(dstFile2) {
		t.Error("file2.txt does not exist in destination subdirectory")
	}
	readContent2, err := os.ReadFile(dstFile2)
	if err != nil {
		t.Fatalf("Failed to read file2: %v", err)
	}
	if string(readContent2) != content2 {
		t.Errorf("Content mismatch in file2: expected %q, got %q", content2, string(readContent2))
	}
}

func TestRenameNonExistent(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "vfox-rename-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "nonexistent.txt")
	dstFile := filepath.Join(tmpDir, "destination.txt")

	// Test renaming non-existent file should fail
	err = Rename(srcFile, dstFile)
	if err == nil {
		t.Error("Expected error when renaming non-existent file, got nil")
	}
}
