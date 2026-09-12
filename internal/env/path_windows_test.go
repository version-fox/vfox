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

package env

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsPathFormatByShell(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "pwsh", "clink", "nushell", ""} {
		t.Run(shell, func(t *testing.T) {
			t.Setenv(HookFlag, shell)
			paths := NewPaths(EmptyPaths)
			paths.Add(`C:\Program Files\Git\bin`)
			paths.Add(`D:\中文 目录\bin`)
			want := `C:\Program Files\Git\bin;D:\中文 目录\bin`
			if shell == "bash" || shell == "zsh" {
				want = "/c/Program Files/Git/bin:/d/中文 目录/bin"
			}
			if got := paths.String(); got != want {
				t.Fatalf("PATH = %q, want %q", got, want)
			}
		})
	}
}

// Run on Windows with MSYS2 zsh on PATH and VFOX_TEST_MSYS_ZSH=1.
func TestMSYSZshPATH(t *testing.T) {
	if os.Getenv("VFOX_TEST_MSYS_ZSH") != "1" {
		t.Skip("requires MSYS2 zsh")
	}
	zsh, err := exec.LookPath("zsh")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(HookFlag, "zsh")
	toolDir := filepath.Join(t.TempDir(), "中文 tools")
	if err := os.Mkdir(toolDir, 0755); err != nil {
		t.Fatal(err)
	}
	// A native executable checks that both Unicode and spaces survive PATH export.
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolDir, "vfox-path-probe.exe"), data, 0755); err != nil {
		t.Fatal(err)
	}
	paths := NewPaths(EmptyPaths)
	paths.Add(toolDir)
	paths.Merge(NewPaths(OsPaths))
	cmd := exec.Command(zsh, "-fc", `export PATH="$VFOX_TEST_PATH"; command -v ls && command -v vfox-path-probe`)
	cmd.Env = append(os.Environ(), "VFOX_TEST_PATH="+paths.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("zsh PATH lookup: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "vfox-path-probe") {
		t.Fatalf("tool absent from zsh PATH: %s", out)
	}
}
