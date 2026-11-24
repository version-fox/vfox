//go:build windows

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

	"github.com/version-fox/vfox/internal/env"
)

// When a bash/zsh hook is active, env.Paths.String() emits a POSIX-style PATH
// (":" separated, "/c/..." drive letters) meant for the shell. exec passes that
// PATH to a native Windows process, whose lookPathInEnv splits on ";", so the
// POSIX PATH becomes a single unusable entry and no SDK bin dir is searched.
func TestBuildExecPathIsNativeOnWindowsWithBashHook(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", ""} {
		t.Run("shell="+shell, func(t *testing.T) {
			t.Setenv(env.HookFlag, shell)

			binDir := t.TempDir()
			tool := filepath.Join(binDir, "vfox-probe.exe")
			if err := os.WriteFile(tool, []byte("probe"), 0o755); err != nil {
				t.Fatal(err)
			}

			paths := env.NewPaths(env.EmptyPaths)
			paths.Add(binDir)

			pathValue := buildExecPath(paths)
			got, err := lookPathInEnv("vfox-probe.exe", pathValue)
			if err != nil {
				t.Fatalf("shell=%q: SDK tool not found in exec PATH %q: %v", shell, pathValue, err)
			}
			if got != tool {
				t.Fatalf("found %q, want %q", got, tool)
			}
		})
	}
}

func TestLookPathInEnvResolvesPATHEXT(t *testing.T) {
	binDir := t.TempDir()
	otherDir := t.TempDir()
	sdkCmd := filepath.Join(binDir, "etcd.CMD")
	otherCmd := filepath.Join(otherDir, "etcd.CMD")
	for _, p := range []string{sdkCmd, otherCmd} {
		if err := os.WriteFile(p, []byte("@echo off\r\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	pathValue := binDir + string(os.PathListSeparator) + otherDir
	got, err := lookPathInEnvWithExt("etcd", pathValue, ".COM;.EXE;.BAT;.CMD")
	if err != nil {
		t.Fatalf("lookPathInEnvWithExt returned error: %v", err)
	}
	if !strings.EqualFold(got, sdkCmd) {
		t.Fatalf("found %q, want SDK entry %q", got, sdkCmd)
	}
}
