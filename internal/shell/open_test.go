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

package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The child processes model a shell -> vfox shim -> vfox process chain.
// Reopening the shim prints its executable name, so the test can distinguish
// it from the actual shell without launching an interactive shell.
func TestMain(m *testing.M) {
	mode := os.Getenv("VFOX_SHELL_PROCESS_TEST")
	if mode == "" {
		os.Exit(m.Run())
	}
	if len(os.Args) == 1 {
		fmt.Println(filepath.Base(os.Args[0]))
		os.Exit(0)
	}
	if mode == "child" {
		if err := Open(os.Getppid()); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	next := "child"
	if mode == "shell" {
		next = "wrapper"
	}
	command := exec.Command(os.Getenv("VFOX_SHELL_TEST_BINARY"), "helper")
	command.Env = append(os.Environ(), "VFOX_SHELL_PROCESS_TEST="+next)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestOpenSkipsVfoxShim(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	shellName := "fixture-shell" + filepath.Ext(executable)
	shellDir := filepath.Join(t.TempDir(), "shell install 中文")
	if err := os.Mkdir(shellDir, 0755); err != nil {
		t.Fatal(err)
	}
	shellPath := filepath.Join(shellDir, shellName)
	data, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shellPath, data, 0755); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"direct", "shell"} {
		t.Run(mode, func(t *testing.T) {
			command := exec.Command(shellPath, "helper")
			command.Env = append(os.Environ(), "VFOX_SHELL_PROCESS_TEST="+mode, "VFOX_SHELL_TEST_BINARY="+executable)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("open shell: %v\n%s", err, output)
			}
			if got := strings.TrimSpace(string(output)); got != shellName {
				t.Fatalf("opened %q, want the parent shell %q instead of the vfox shim", got, shellName)
			}
		})
	}
}
