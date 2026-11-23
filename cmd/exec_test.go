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

package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal"
)

func TestExecCLIProcess(t *testing.T) {
	if os.Getenv("VFOX_EXEC_CLI_PROCESS") != "1" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		os.Exit(2)
	}
	os.Args = append([]string{"vfox"}, os.Args[separator+1:]...)
	Execute(os.Args)
	os.Exit(0)
}

func runExecCLI(t *testing.T, root string, args ...string) (string, string, int) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, executable, append([]string{"-test.run=^TestExecCLIProcess$", "--"}, args...)...)
	command.Dir = root
	toolDir := filepath.Join(root, "tools")
	command.Env = append(os.Environ(),
		"VFOX_EXEC_CLI_PROCESS=1",
		"HOME="+filepath.Join(root, "user"),
		"USERPROFILE="+filepath.Join(root, "user"),
		"VFOX_HOME="+filepath.Join(root, "shared"),
		"PATH="+toolDir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err = command.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if ctx.Err() != nil {
		t.Fatal("exec CLI hung")
	}
	return stdout.String(), stderr.String(), code
}

func setupExecFakeSDK(t *testing.T, root string) {
	t.Helper()
	pluginDir := filepath.Join(root, "shared", "plugin", "etcd")
	writeCLIPluginFile(t, pluginDir, "main.lua", `PLUGIN = {name="etcd", version="0.2.3", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="3.5.12"}} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`)
	binDir := filepath.Join(root, "shared", "cache", "etcd", "v-3.5.12", "etcd-3.5.12", "bin")
	toolDir := filepath.Join(root, "tools")
	for _, dir := range []string{binDir, toolDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		toolName := "etcd"
		toolContent := "#!/bin/sh\nprintf 'args=%s\\n' \"$*\"\n"
		if runtime.GOOS == "windows" {
			toolName = "etcd.cmd"
			toolContent = "@echo off\r\necho args=%*\r\n"
		}
		if err := os.WriteFile(filepath.Join(dir, toolName), []byte(toolContent), 0755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestExecForwardsFlagsAfterCommandWithoutSeparator(t *testing.T) {
	root := t.TempDir()
	setupExecFakeSDK(t, root)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "-- with --version", args: []string{"exec", "etcd@3.5.12", "--", "etcd", "--version"}, want: "args=--version"},
		{name: "without -- with --version", args: []string{"exec", "etcd@3.5.12", "etcd", "--version"}, want: "args=--version"},
		{name: "without -- with --help", args: []string{"exec", "etcd@3.5.12", "etcd", "--help"}, want: "args=--help"},
		{name: "without -- with -v -V -h", args: []string{"exec", "etcd@3.5.12", "etcd", "-v", "-V", "-h"}, want: "args=-v -V -h"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code := runExecCLI(t, root, tc.args...)
			if code != 0 {
				t.Fatalf("vfox %v exited %d (stderr:\n%s\nstdout:\n%s)", tc.args, code, stderr, stdout)
			}
			if got := strings.TrimSpace(stdout); got != tc.want {
				t.Fatalf("stdout = %q, want %q (stderr:\n%s)", got, tc.want, stderr)
			}
			if strings.Contains(stderr, internal.RuntimeVersion) {
				t.Fatalf("stderr leaked vfox version %q:\n%s", internal.RuntimeVersion, stderr)
			}
			if strings.Contains(stderr, "No help topic") {
				t.Fatalf("stderr contained help topic error:\n%s", stderr)
			}
		})
	}

	t.Run("root --debug before exec still enables debug while preserving target flags", func(t *testing.T) {
		stdout, stderr, code := runExecCLI(t, root, "--debug", "exec", "etcd@3.5.12", "etcd", "--version")
		if code != 0 {
			t.Fatalf("vfox --debug exec exited %d (stderr:\n%s\nstdout:\n%s)", code, stderr, stdout)
		}
		if !strings.Contains(stdout, "Looking up SDK: etcd") {
			t.Fatalf("stdout missing debug log output:\n%s", stdout)
		}
		if !strings.Contains(stdout, "args=--version") {
			t.Fatalf("stdout missing target output %q:\n%s", "args=--version", stdout)
		}
		if strings.Contains(stderr, internal.RuntimeVersion) {
			t.Fatalf("stderr leaked vfox version %q:\n%s", internal.RuntimeVersion, stderr)
		}
	})

	t.Run("exec --debug between exec and sdk enables debug and runs command", func(t *testing.T) {
		stdout, stderr, code := runExecCLI(t, root, "exec", "--debug", "etcd@3.5.12", "etcd", "put")
		if code != 0 {
			t.Fatalf("vfox exec --debug exited %d (stderr:\n%s\nstdout:\n%s)", code, stderr, stdout)
		}
		if !strings.Contains(stdout, "Looking up SDK: etcd") {
			t.Fatalf("stdout missing debug log output:\n%s", stdout)
		}
		if !strings.Contains(stdout, "args=put") {
			t.Fatalf("stdout missing target output %q:\n%s", "args=put", stdout)
		}
	})

	t.Run("exec --debug before -- still preserves --debug after command", func(t *testing.T) {
		stdout, stderr, code := runExecCLI(t, root, "exec", "--debug", "etcd@3.5.12", "--", "etcd", "--debug", "--version")
		if code != 0 {
			t.Fatalf("vfox exec --debug -- exited %d (stderr:\n%s\nstdout:\n%s)", code, stderr, stdout)
		}
		if !strings.Contains(stdout, "Looking up SDK: etcd") {
			t.Fatalf("stdout missing debug log output:\n%s", stdout)
		}
		if !strings.Contains(stdout, "args=--debug --version") {
			t.Fatalf("stdout missing target output %q:\n%s", "args=--debug --version", stdout)
		}
	})

	for _, flag := range []string{"-v", "-V", "--version"} {
		t.Run("exec "+flag+" between exec and sdk fails naming the flag", func(t *testing.T) {
			stdout, stderr, code := runExecCLI(t, root, "exec", flag, "etcd@3.5.12", "etcd", "put")
			if code == 0 {
				t.Fatalf("expected non-zero exit code for `vfox exec %s`, got 0 (stdout:\n%s)", flag, stdout)
			}
			if !strings.Contains(stderr, flag) {
				t.Fatalf("stderr did not name flag %q:\n%s", flag, stderr)
			}
			if strings.Contains(stderr, "multiple SDKs") {
				t.Fatalf("stderr gave misleading multi-SDK message:\n%s", stderr)
			}
			if !strings.Contains(stderr, "usage: vfox exec") {
				t.Fatalf("stderr missing usage line:\n%s", stderr)
			}
			if strings.Contains(stderr, internal.RuntimeVersion) || strings.Contains(stdout, internal.RuntimeVersion) {
				t.Fatalf("unexpected vfox version %q printed (stdout=%q stderr=%q)", internal.RuntimeVersion, stdout, stderr)
			}
		})
	}

	t.Run("first -- separates sdks and names word parsed as sdk on failure", func(t *testing.T) {
		for _, tc := range []struct {
			name     string
			args     []string
			wantWord string
			wantErr  string
		}{
			{
				name:     "installed sdk without version before --",
				args:     []string{"exec", "etcd@3.5.12", "etcd", "--", "--version"},
				wantWord: "etcd",
				wantErr:  "no version configured for etcd",
			},
			{
				name:     "uninstalled command word before --",
				args:     []string{"exec", "etcd@3.5.12", "npm", "run", "build", "--", "--watch"},
				wantWord: "npm",
				wantErr:  "npm not supported",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				stdout, stderr, code := runExecCLI(t, root, tc.args...)
				if code == 0 {
					t.Fatalf("expected non-zero exit code for %v, got 0 (stdout:\n%s)", tc.args, stdout)
				}
				if !strings.Contains(stderr, tc.wantErr) {
					t.Fatalf("stderr missing %q naming %q:\n%s", tc.wantErr, tc.wantWord, stderr)
				}
				if !strings.Contains(stderr, "first '--'") {
					t.Fatalf("stderr missing first '--' separator explanation:\n%s", stderr)
				}
			})
		}
	})

	t.Run("command -- passes through when -- separates sdk and command", func(t *testing.T) {
		stdout, stderr, code := runExecCLI(t, root, "exec", "etcd@3.5.12", "--", "etcd", "put", "--", "key")
		if code != 0 {
			t.Fatalf("vfox exec -- etcd put -- key exited %d (stderr:\n%s\nstdout:\n%s)", code, stderr, stdout)
		}
		if got := strings.TrimSpace(stdout); got != "args=put -- key" {
			t.Fatalf("stdout = %q, want %q (stderr:\n%s)", got, "args=put -- key", stderr)
		}
	})

	t.Run("flag right after sdk without -- returns usage and does not print vfox version", func(t *testing.T) {
		stdout, stderr, code := runExecCLI(t, root, "exec", "clang", "--version")
		if code == 0 {
			t.Fatalf("expected non-zero exit code for `vfox exec clang --version`, got 0 (stdout:\n%s)", stdout)
		}
		if !strings.Contains(stderr, "usage: vfox exec") {
			t.Fatalf("stderr missing usage line:\n%s", stderr)
		}
		if strings.Contains(stderr, internal.RuntimeVersion) || strings.Contains(stdout, internal.RuntimeVersion) {
			t.Fatalf("unexpected vfox version %q printed (stdout=%q stderr=%q)", internal.RuntimeVersion, stdout, stderr)
		}
	})

	t.Run("exec --help prints command help including global options", func(t *testing.T) {
		for _, flag := range []string{"--help", "-h"} {
			t.Run(flag, func(t *testing.T) {
				stdout, stderr, code := runExecCLI(t, root, "exec", flag)
				if code != 0 {
					t.Fatalf("vfox exec %s exited %d (stderr:\n%s\nstdout:\n%s)", flag, code, stderr, stdout)
				}
				for _, want := range []string{
					"Execute a command in vfox managed environment",
					"vfox exec [options]",
					"OPTIONS:",
					"GLOBAL OPTIONS:",
					"--debug",
					"--version, -v, -V",
					"first '--'",
				} {
					if !strings.Contains(stdout, want) {
						t.Fatalf("exec %s missing %q in output:\n%s", flag, want, stdout)
					}
				}
			})
		}
	})

	t.Run("root --version still prints vfox version", func(t *testing.T) {
		stdout, stderr, code := runExecCLI(t, root, "--version")
		if code != 0 {
			t.Fatalf("vfox --version exited %d (stderr:\n%s)", code, stderr)
		}
		if !strings.Contains(stdout+stderr, "vfox version "+internal.RuntimeVersion) {
			t.Fatalf("vfox --version output missing version %q: stdout=%q stderr=%q", internal.RuntimeVersion, stdout, stderr)
		}
	})
}
