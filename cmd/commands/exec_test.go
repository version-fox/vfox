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
	"slices"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

// buildExecPath must join entries with the native path list separator no
// matter which shell hook is active. On Windows, env.Paths.String() emits a
// POSIX-style, ":"-joined PATH for bash/zsh hooks, meant for the hook shell;
// a process launched by exec cannot consume it. Splitting the exec PATH on
// os.PathListSeparator must always reproduce the original entries, so this
// check exercises the separator logic on every platform.
func TestBuildExecPathUsesNativeSeparator(t *testing.T) {
	for _, shell := range []string{"", "bash", "zsh"} {
		t.Run("shell="+shell, func(t *testing.T) {
			t.Setenv(env.HookFlag, shell)

			paths := env.NewPaths(env.EmptyPaths)
			first := filepath.Join(t.TempDir(), "one")
			second := filepath.Join(t.TempDir(), "two")
			paths.Add(first)
			paths.Add(second)

			got := strings.Split(buildExecPath(paths), string(os.PathListSeparator))
			if !slices.Equal(got, paths.Slice()) {
				t.Fatalf("buildExecPath = %q, want %q", got, paths.Slice())
			}
		})
	}
}

func TestParseExecInvocationPreservesCommandFlagsWithoutSeparator(t *testing.T) {
	t.Run("accepts command flags without --", func(t *testing.T) {
		for _, tc := range []struct {
			name        string
			args        []string
			wantSpecs   []execSDKSpec
			wantCommand string
			wantArgs    []string
		}{
			{
				name:        "single sdk without -- and --version",
				args:        []string{"etcd@3.5.12", "etcd", "--version"},
				wantSpecs:   []execSDKSpec{{Name: "etcd", Version: "3.5.12"}},
				wantCommand: "etcd",
				wantArgs:    []string{"--version"},
			},
			{
				name:        "single sdk without -- and --help",
				args:        []string{"etcd@3.5.12", "etcd", "--help"},
				wantSpecs:   []execSDKSpec{{Name: "etcd", Version: "3.5.12"}},
				wantCommand: "etcd",
				wantArgs:    []string{"--help"},
			},
			{
				name:        "single sdk without -- and multiple vfox-like flags",
				args:        []string{"etcd@3.5.12", "etcd", "-v", "-V", "-h", "--debug"},
				wantSpecs:   []execSDKSpec{{Name: "etcd", Version: "3.5.12"}},
				wantCommand: "etcd",
				wantArgs:    []string{"-v", "-V", "-h", "--debug"},
			},
			{
				name:        "single sdk with -- and --version",
				args:        []string{"etcd@3.5.12", "--", "etcd", "--version"},
				wantSpecs:   []execSDKSpec{{Name: "etcd", Version: "3.5.12"}},
				wantCommand: "etcd",
				wantArgs:    []string{"--version"},
			},
			{
				name:        "multiple sdks with -- and flags",
				args:        []string{"nodejs@20", "python@3.12", "--", "python", "--help", "--version"},
				wantSpecs:   []execSDKSpec{{Name: "nodejs", Version: "20"}, {Name: "python", Version: "3.12"}},
				wantCommand: "python",
				wantArgs:    []string{"--help", "--version"},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				specs, cmdName, cmdArgs, err := parseExecInvocation(tc.args)
				if err != nil {
					t.Fatalf("parseExecInvocation(%q) unexpected error: %v", tc.args, err)
				}
				if !slices.Equal(specs, tc.wantSpecs) {
					t.Fatalf("specs = %#v, want %#v", specs, tc.wantSpecs)
				}
				if cmdName != tc.wantCommand {
					t.Fatalf("command = %q, want %q", cmdName, tc.wantCommand)
				}
				if !slices.Equal(cmdArgs, tc.wantArgs) {
					t.Fatalf("cmdArgs = %q, want %q", cmdArgs, tc.wantArgs)
				}
			})
		}
	})

	t.Run("rejects missing command or flag before command", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			args []string
		}{
			{name: "no args", args: nil},
			{name: "only sdk", args: []string{"etcd@3.5.12"}},
			{name: "flag instead of command --version", args: []string{"clang", "--version"}},
			{name: "flag instead of command --help", args: []string{"etcd@3.5.12", "--help"}},
			{name: "leading --version flag", args: []string{"--version"}},
			{name: "leading flag before --", args: []string{"--version", "etcd", "--", "echo"}},
			{name: "separator with no sdk", args: []string{"--", "etcd"}},
			{name: "separator with no command", args: []string{"etcd@3.5.12", "--"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				if _, _, _, err := parseExecInvocation(tc.args); err == nil {
					t.Fatalf("parseExecInvocation(%q) succeeded unexpectedly", tc.args)
				}
			})
		}
	})
}
