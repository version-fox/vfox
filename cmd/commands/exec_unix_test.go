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
	"testing"
)

func TestLookPathInEnvSkipsNonExecutableUnixFiles(t *testing.T) {
	sdkBin := t.TempDir()
	fallbackBin := t.TempDir()

	nonExec := filepath.Join(sdkBin, "nonexec-tool")
	if err := os.WriteFile(nonExec, []byte("#!/bin/sh\necho bad\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fallbackTool := filepath.Join(fallbackBin, "nonexec-tool")
	if err := os.WriteFile(fallbackTool, []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	pathValue := sdkBin + string(os.PathListSeparator) + fallbackBin
	got, err := lookPathInEnvWithExt("nonexec-tool", pathValue, "")
	if err != nil {
		t.Fatalf("lookPathInEnvWithExt returned error: %v", err)
	}
	if got != fallbackTool {
		t.Fatalf("found %q, want executable fallback %q", got, fallbackTool)
	}

	if err := os.Remove(fallbackTool); err != nil {
		t.Fatal(err)
	}
	if got, err := lookPathInEnvWithExt("nonexec-tool", sdkBin, ""); err == nil {
		t.Fatalf("lookPathInEnvWithExt unexpectedly returned non-executable file %q", got)
	}
}
