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

func execExpectedLine(args string) string {
	return "sdk:args=" + args
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
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}
	toolDir := filepath.Join(root, "tools")
	if err := os.MkdirAll(toolDir, 0755); err != nil {
		t.Fatal(err)
	}

	if runtime.GOOS == "windows" {
		if err := os.WriteFile(filepath.Join(binDir, "etcd.cmd"), []byte("@echo off\r\necho sdk:args=%*\r\n"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(toolDir, "etcd.cmd"), []byte("@echo off\r\necho decoy-tools:args=%*\r\n"), 0755); err != nil {
			t.Fatal(err)
		}
		return
	}

	if err := os.WriteFile(filepath.Join(binDir, "etcd"), []byte("#!/bin/sh\nprintf 'sdk:args=%s\\n' \"$*\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(toolDir, "etcd"), []byte("#!/bin/sh\nprintf 'decoy-tools:args=%s\\n' \"$*\"\n"), 0755); err != nil {
		t.Fatal(err)
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
		{name: "-- with --version", args: []string{"exec", "etcd@3.5.12", "--", "etcd", "--version"}, want: execExpectedLine("--version")},
		{name: "without -- with --version", args: []string{"exec", "etcd@3.5.12", "etcd", "--version"}, want: execExpectedLine("--version")},
		{name: "without -- with --help", args: []string{"exec", "etcd@3.5.12", "etcd", "--help"}, want: execExpectedLine("--help")},
		{name: "without -- with -help", args: []string{"exec", "etcd@3.5.12", "etcd", "-help"}, want: execExpectedLine("-help")},
		{name: "without -- with -v -V -h", args: []string{"exec", "etcd@3.5.12", "etcd", "-v", "-V", "-h"}, want: execExpectedLine("-v -V -h")},
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

	t.Run("trailing --generate-shell-completion emits empty output without running target with or without --", func(t *testing.T) {
		probeDir := t.TempDir()
		marker := filepath.Join(probeDir, "ran.txt")
		var probeCmd string
		if runtime.GOOS == "windows" {
			probeCmd = filepath.Join(probeDir, "probe.cmd")
			if err := os.WriteFile(probeCmd, []byte("@echo off\r\necho ran>\""+marker+"\"\r\necho sdk:args=%*\r\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		} else {
			probeCmd = filepath.Join(probeDir, "probe")
			if err := os.WriteFile(probeCmd, []byte("#!/bin/sh\necho ran >\""+marker+"\"\nprintf 'sdk:args=%s\\n' \"$*\"\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		}

		for _, tc := range []struct {
			name string
			args []string
		}{
			{name: "exec without --", args: []string{"exec", "etcd@3.5.12", probeCmd, "--generate-shell-completion"}},
			{name: "exec with --", args: []string{"exec", "etcd@3.5.12", "--", probeCmd, "--generate-shell-completion"}},
			{name: "x alias with --", args: []string{"x", "etcd@3.5.12", "--", probeCmd, "--generate-shell-completion"}},
			{name: "x alias without --", args: []string{"x", "etcd@3.5.12", probeCmd, "--generate-shell-completion"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				stdout, stderr, code := runExecCLI(t, root, tc.args...)
				if code != 0 {
					t.Fatalf("vfox %v exited %d (stderr:\n%s\nstdout:\n%s)", tc.args, code, stderr, stdout)
				}
				if stderr != "" {
					t.Fatalf("vfox %v wrote unexpected stderr:\n%s", tc.args, stderr)
				}
				if stdout != "" {
					t.Fatalf("vfox %v stdout = %q, want empty", tc.args, stdout)
				}
				if _, err := os.Stat(marker); !os.IsNotExist(err) {
					t.Fatalf("vfox %v invoked target script %q", tc.args, marker)
				}
			})
		}
	})

	t.Run("root --debug before exec still enables debug while preserving target flags", func(t *testing.T) {
		stdout, stderr, code := runExecCLI(t, root, "--debug", "exec", "etcd@3.5.12", "etcd", "--version")
		if code != 0 {
			t.Fatalf("vfox --debug exec exited %d (stderr:\n%s\nstdout:\n%s)", code, stderr, stdout)
		}
		if !strings.Contains(stdout, "Looking up SDK: etcd") {
			t.Fatalf("stdout missing debug log output:\n%s", stdout)
		}
		if !strings.Contains(stdout, execExpectedLine("--version")) {
			t.Fatalf("stdout missing target output %q:\n%s", execExpectedLine("--version"), stdout)
		}
		if strings.Contains(stderr, internal.RuntimeVersion) {
			t.Fatalf("stderr leaked vfox version %q:\n%s", internal.RuntimeVersion, stderr)
		}
	})

	t.Run("root --debug=false before exec keeps debug off", func(t *testing.T) {
		stdout, stderr, code := runExecCLI(t, root, "--debug=false", "exec", "etcd@3.5.12", "etcd", "put")
		if code != 0 {
			t.Fatalf("vfox --debug=false exec exited %d (stderr:\n%s\nstdout:\n%s)", code, stderr, stdout)
		}
		if strings.Contains(stdout, "Looking up SDK: etcd") {
			t.Fatalf("stdout unexpectedly contained debug log output:\n%s", stdout)
		}
		if got := strings.TrimSpace(stdout); got != execExpectedLine("put") {
			t.Fatalf("stdout = %q, want %q (stderr:\n%s)", got, execExpectedLine("put"), stderr)
		}
	})

	t.Run("single sdk without -- runs command path containing @", func(t *testing.T) {
		scopedDir := filepath.Join(root, "node_modules", "@foo", "bin")
		if err := os.MkdirAll(scopedDir, 0755); err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS == "windows" {
			if err := os.WriteFile(filepath.Join(scopedDir, "run.cmd"), []byte("@echo off\r\necho local:args=%*\r\n"), 0755); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.WriteFile(filepath.Join(scopedDir, "run"), []byte("#!/bin/sh\nprintf 'local:args=%s\\n' \"$*\"\n"), 0755); err != nil {
				t.Fatal(err)
			}
		}
		stdout, stderr, code := runExecCLI(t, root, "exec", "etcd@3.5.12", "./node_modules/@foo/bin/run", "arg")
		if code != 0 {
			t.Fatalf("vfox exec ./node_modules/@foo/bin/run exited %d (stderr:\n%s\nstdout:\n%s)", code, stderr, stdout)
		}
		if got := strings.TrimSpace(stdout); got != "local:args=arg" {
			t.Fatalf("stdout = %q, want %q", got, "local:args=arg")
		}
	})

	for _, tc := range []struct {
		name      string
		cmdArgs   []string
		wantDebug bool
	}{
		{name: "exec --debug", cmdArgs: []string{"exec", "--debug", "etcd@3.5.12", "etcd", "put"}, wantDebug: true},
		{name: "exec -debug", cmdArgs: []string{"exec", "-debug", "etcd@3.5.12", "etcd", "put"}, wantDebug: true},
		{name: "exec -debug=true", cmdArgs: []string{"exec", "-debug=true", "etcd@3.5.12", "etcd", "put"}, wantDebug: true},
		{name: "exec --debug=1", cmdArgs: []string{"exec", "--debug=1", "etcd@3.5.12", "etcd", "put"}, wantDebug: true},
		{name: "exec --debug=TRUE", cmdArgs: []string{"exec", "--debug=TRUE", "etcd@3.5.12", "etcd", "put"}, wantDebug: true},
		{name: "x -debug alias", cmdArgs: []string{"x", "-debug", "etcd@3.5.12", "etcd", "put"}, wantDebug: true},
		{name: "exec --debug=0", cmdArgs: []string{"exec", "--debug=0", "etcd@3.5.12", "etcd", "put"}, wantDebug: false},
		{name: "exec --debug=false", cmdArgs: []string{"exec", "--debug=false", "etcd@3.5.12", "etcd", "put"}, wantDebug: false},
	} {
		t.Run(tc.name+" between exec and sdk matches debug parsing", func(t *testing.T) {
			stdout, stderr, code := runExecCLI(t, root, tc.cmdArgs...)
			if code != 0 {
				t.Fatalf("vfox %v exited %d (stderr:\n%s\nstdout:\n%s)", tc.cmdArgs, code, stderr, stdout)
			}
			if got := strings.Contains(stdout, "Looking up SDK: etcd"); got != tc.wantDebug {
				t.Fatalf("debug log present = %v, want %v (stdout:\n%s)", got, tc.wantDebug, stdout)
			}
			if !strings.Contains(stdout, execExpectedLine("put")) {
				t.Fatalf("stdout missing target output %q:\n%s", execExpectedLine("put"), stdout)
			}
		})
	}

	t.Run("exec --debug before -- still preserves --debug after command", func(t *testing.T) {
		stdout, stderr, code := runExecCLI(t, root, "exec", "--debug", "etcd@3.5.12", "--", "etcd", "--debug", "--version")
		if code != 0 {
			t.Fatalf("vfox exec --debug -- exited %d (stderr:\n%s\nstdout:\n%s)", code, stderr, stdout)
		}
		if !strings.Contains(stdout, "Looking up SDK: etcd") {
			t.Fatalf("stdout missing debug log output:\n%s", stdout)
		}
		if !strings.Contains(stdout, execExpectedLine("--debug --version")) {
			t.Fatalf("stdout missing target output %q:\n%s", execExpectedLine("--debug --version"), stdout)
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

	t.Run("separator notes distinguish missing or misplaced -- from explicit sdk@version errors", func(t *testing.T) {
		for _, tc := range []struct {
			name            string
			args            []string
			wantErrContains []string
			wantErrAbsent   string
		}{
			{
				name:            "installed sdk without version before --",
				args:            []string{"exec", "etcd@3.5.12", "etcd", "--", "--version"},
				wantErrContains: []string{"no version configured for etcd", "first '--'"},
			},
			{
				name:            "uninstalled command word before --",
				args:            []string{"exec", "etcd@3.5.12", "npm", "run", "build", "--", "--watch"},
				wantErrContains: []string{"npm not supported", "first '--'"},
			},
			{
				name:            "explicit missing sdk@version after -- omits misplaced-separator note",
				args:            []string{"exec", "etcd@3.5.12", "nosuch@1", "--", "etcd", "put"},
				wantErrContains: []string{"nosuch not supported"},
				wantErrAbsent:   "pass '--' before the command",
			},
			{
				name:            "missing sdk before -- with plain command states separator rule without pass instruction",
				args:            []string{"exec", "etcd@3.5.12", "nosuch", "--", "etcd", "put"},
				wantErrContains: []string{"nosuch not supported", "first '--'", `"nosuch"`},
				wantErrAbsent:   "pass '--' before the command",
			},
			{
				name:            "configured sdk then missing sdk before -- states separator rule without pass instruction",
				args:            []string{"exec", "etcd@3.5.12", "etcd", "--", "etcd", "put"},
				wantErrContains: []string{"no version configured for etcd", "first '--'", `"etcd"`},
				wantErrAbsent:   "pass '--' before the command",
			},
			{
				name:            "misplaced -- after command with plain arg states separator rule",
				args:            []string{"exec", "etcd@3.5.12", "etcd", "put", "--", "key"},
				wantErrContains: []string{"no version configured for etcd", "first '--'", `"etcd"`},
				wantErrAbsent:   "pass '--' before the command",
			},
			{
				name:            "missing sdk before -- when command itself passes -- omits pass instruction",
				args:            []string{"exec", "etcd@3.5.12", "nosuch", "--", "etcd", "put", "--", "key"},
				wantErrContains: []string{"nosuch not supported", "first '--'", `"nosuch"`},
				wantErrAbsent:   "pass '--' before the command",
			},
			{
				name:            "missing sdk before -- when command takes --save omits pass instruction",
				args:            []string{"exec", "etcd@3.5.12", "nosuch", "--", "npm", "install", "--save", "foo"},
				wantErrContains: []string{"nosuch not supported", "first '--'", `"nosuch"`},
				wantErrAbsent:   "pass '--' before the command",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				stdout, stderr, code := runExecCLI(t, root, tc.args...)
				if code == 0 {
					t.Fatalf("expected non-zero exit code for %v, got 0 (stdout:\n%s)", tc.args, stdout)
				}
				for _, want := range tc.wantErrContains {
					if !strings.Contains(stderr, want) {
						t.Fatalf("stderr missing %q:\n%s", want, stderr)
					}
				}
				if tc.wantErrAbsent != "" && strings.Contains(stderr, tc.wantErrAbsent) {
					t.Fatalf("stderr unexpectedly contained %q:\n%s", tc.wantErrAbsent, stderr)
				}
			})
		}
	})

	t.Run("command not found explains separator rules when misplaced or omitted", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(root, ".vfox.toml"), []byte("[tools]\netcd = \"3.5.12\"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(filepath.Join(root, ".vfox.toml")) })

		alphaPluginDir := filepath.Join(root, "shared", "plugin", "alpha")
		writeCLIPluginFile(t, alphaPluginDir, "main.lua", `PLUGIN = {name="alpha", version="0.1.0", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {} end
function PLUGIN:EnvKeys(ctx) return {} end
`)
		t.Cleanup(func() { _ = os.RemoveAll(alphaPluginDir) })

		for _, tc := range []struct {
			name            string
			args            []string
			wantErrContains []string
			wantErrAbsent   string
		}{
			{
				name:            "configured sdk repeated before -- turns flag into command",
				args:            []string{"exec", "etcd", "etcd", "--", "--version"},
				wantErrContains: []string{"command not found: --version", "first '--'", `"etcd"`},
			},
			{
				name:            "unseparated known sdk command notes single-sdk parsing",
				args:            []string{"exec", "etcd", "alpha", "put"},
				wantErrContains: []string{"command not found: alpha", `"etcd"`, `"alpha"`, "pass '--'"},
			},
			{
				name:            "unseparated pinned sdk with typo command omits multi-sdk note",
				args:            []string{"exec", "etcd@3.5.12", "etcdx", "put"},
				wantErrContains: []string{"command not found: etcdx"},
				wantErrAbsent:   "note:",
			},
			{
				name:            "unseparated unknown command omits multi-sdk note",
				args:            []string{"exec", "etcd", "nosuch", "put"},
				wantErrContains: []string{"command not found: nosuch"},
				wantErrAbsent:   "note:",
			},
			{
				name:            "separated missing non-flag command omits separator note",
				args:            []string{"exec", "etcd@3.5.12", "--", "nosuchcmd"},
				wantErrContains: []string{"command not found: nosuchcmd"},
				wantErrAbsent:   "note:",
			},
			{
				name:            "separated missing flag-like command with single sdk omits separator note",
				args:            []string{"exec", "etcd@3.5.12", "--", "--version"},
				wantErrContains: []string{"command not found: --version"},
				wantErrAbsent:   "note:",
			},
			{
				name:            "separated missing short flag command with single sdk omits separator note",
				args:            []string{"exec", "etcd@3.5.12", "--", "-x"},
				wantErrContains: []string{"command not found: -x"},
				wantErrAbsent:   "note:",
			},
			{
				name:            "unseparated single-word missing command omits multi-sdk note",
				args:            []string{"exec", "etcd@3.5.12", "nosuchcmd"},
				wantErrContains: []string{"command not found: nosuchcmd"},
				wantErrAbsent:   "note:",
			},
			{
				name:            "unseparated dash command reports command not found instead of unexpected flag",
				args:            []string{"exec", "etcd@3.5.12", "-"},
				wantErrContains: []string{"command not found: -"},
				wantErrAbsent:   "note:",
			},
			{
				name:            "configured sdk then missing sdk before -- omits misplaced-separator note",
				args:            []string{"exec", "etcd", "nosuch", "--", "etcd", "put"},
				wantErrContains: []string{"nosuch not supported", "first '--'", `"nosuch"`},
				wantErrAbsent:   "pass '--' before the command",
			},
			{
				name:            "configured sdk with misplaced -- after put states separator rule",
				args:            []string{"exec", "etcd", "etcd", "put", "--", "key"},
				wantErrContains: []string{"put not supported", "first '--'", `"put"`},
				wantErrAbsent:   "pass '--' before the command",
			},
			{
				name:            "unconfigured second plugin before -- when command takes -- omits pass instruction",
				args:            []string{"exec", "etcd@3.5.12", "alpha", "--", "etcd", "put", "--", "key"},
				wantErrContains: []string{"no version configured for alpha", "first '--'", `"alpha"`},
				wantErrAbsent:   "pass '--' before the command",
			},
			{
				name:            "unseparated dot-slash command omits multi-sdk note",
				args:            []string{"exec", "etcd@3.5.12", "./etcd", "arg"},
				wantErrContains: []string{"command not found: ./etcd"},
				wantErrAbsent:   "pass '--' before the command",
			},
			{
				name:            "unseparated parent-path command omits multi-sdk note",
				args:            []string{"exec", "etcd@3.5.12", "../plugin/etcd", "arg"},
				wantErrContains: []string{"command not found: ../plugin/etcd"},
				wantErrAbsent:   "pass '--' before the command",
			},
			{
				name:            "versioned trailing sdk before -- omits untreated-command note",
				args:            []string{"exec", "etcd@3.5.12", "etcd@3.5.12", "--", "--version"},
				wantErrContains: []string{"command not found: --version"},
				wantErrAbsent:   "was treated as an SDK instead of the command",
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				stdout, stderr, code := runExecCLI(t, root, tc.args...)
				if code == 0 {
					t.Fatalf("expected non-zero exit code for %v, got 0 (stdout:\n%s)", tc.args, stdout)
				}
				for _, want := range tc.wantErrContains {
					if !strings.Contains(stderr, want) {
						t.Fatalf("stderr missing %q:\n%s", want, stderr)
					}
				}
				if tc.wantErrAbsent != "" && strings.Contains(stderr, tc.wantErrAbsent) {
					t.Fatalf("stderr unexpectedly contained %q:\n%s", tc.wantErrAbsent, stderr)
				}
			})
		}
	})

	t.Run("command -- passes through when -- separates sdk and command", func(t *testing.T) {
		stdout, stderr, code := runExecCLI(t, root, "exec", "etcd@3.5.12", "--", "etcd", "put", "--", "key")
		if code != 0 {
			t.Fatalf("vfox exec -- etcd put -- key exited %d (stderr:\n%s\nstdout:\n%s)", code, stderr, stdout)
		}
		if got := strings.TrimSpace(stdout); got != execExpectedLine("put -- key") {
			t.Fatalf("stdout = %q, want %q (stderr:\n%s)", got, execExpectedLine("put -- key"), stderr)
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
		for _, flag := range []string{"--help", "-help", "-h"} {
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
