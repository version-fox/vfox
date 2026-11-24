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
	"runtime"
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
			{
				name:        "single sdk without -- and slash path containing @",
				args:        []string{"etcd@3.5.12", "./node_modules/@foo/bin/run", "arg"},
				wantSpecs:   []execSDKSpec{{Name: "etcd", Version: "3.5.12"}},
				wantCommand: "./node_modules/@foo/bin/run",
				wantArgs:    []string{"arg"},
			},
			{
				name:        "single sdk without -- and backslash path containing @",
				args:        []string{"etcd@3.5.12", `.\node_modules\@foo\bin\run.cmd`, "arg"},
				wantSpecs:   []execSDKSpec{{Name: "etcd", Version: "3.5.12"}},
				wantCommand: `.\node_modules\@foo\bin\run.cmd`,
				wantArgs:    []string{"arg"},
			},
			{
				name:        "single sdk without -- and dash command",
				args:        []string{"etcd@3.5.12", "-", "arg"},
				wantSpecs:   []execSDKSpec{{Name: "etcd", Version: "3.5.12"}},
				wantCommand: "-",
				wantArgs:    []string{"arg"},
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
			name            string
			args            []string
			wantErrContains string
		}{
			{name: "no args", args: nil, wantErrContains: "usage: vfox exec"},
			{name: "only sdk", args: []string{"etcd@3.5.12"}, wantErrContains: "usage: vfox exec"},
			{name: "flag instead of command --version", args: []string{"clang", "--version"}, wantErrContains: "--version"},
			{name: "flag instead of command --help", args: []string{"etcd@3.5.12", "--help"}, wantErrContains: "--help"},
			{name: "leading --version flag", args: []string{"--version"}, wantErrContains: "usage: vfox exec"},
			{name: "leading flag before --", args: []string{"--version", "etcd", "--", "echo"}, wantErrContains: "--version"},
			{name: "flag in sdk list before --", args: []string{"etcd@3.5.12", "-v", "--", "etcd"}, wantErrContains: "-v"},
			{name: "separator with no sdk", args: []string{"--", "etcd"}, wantErrContains: "usage: vfox exec"},
			{name: "separator with no command", args: []string{"etcd@3.5.12", "--"}, wantErrContains: "usage: vfox exec"},
			{name: "two sdk specs without separator", args: []string{"etcd@3.5.12", "golang@1.25.6", "etcd"}, wantErrContains: "multiple SDKs require '--'"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, _, _, err := parseExecInvocation(tc.args)
				if err == nil {
					t.Fatalf("parseExecInvocation(%q) succeeded unexpectedly", tc.args)
				}
				if tc.wantErrContains != "" && !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Fatalf("parseExecInvocation(%q) error = %q, want substring %q", tc.args, err.Error(), tc.wantErrContains)
				}
			})
		}
	})
}

func TestParseExecLeadingArgs(t *testing.T) {
	t.Run("consumes leading --debug and --help flags only before sdk", func(t *testing.T) {
		for _, tc := range []struct {
			name          string
			args          []string
			wantRemaining []string
			wantOpts      execOptions
		}{
			{
				name:          "leading --debug before sdk",
				args:          []string{"--debug", "etcd@3.5.12", "etcd", "put"},
				wantRemaining: []string{"etcd@3.5.12", "etcd", "put"},
				wantOpts:      execOptions{Debug: true, DebugSet: true},
			},
			{
				name:          "leading -debug before sdk",
				args:          []string{"-debug", "etcd@3.5.12", "etcd", "put"},
				wantRemaining: []string{"etcd@3.5.12", "etcd", "put"},
				wantOpts:      execOptions{Debug: true, DebugSet: true},
			},
			{
				name:          "leading -debug=true before sdk",
				args:          []string{"-debug=true", "etcd@3.5.12", "etcd", "put"},
				wantRemaining: []string{"etcd@3.5.12", "etcd", "put"},
				wantOpts:      execOptions{Debug: true, DebugSet: true},
			},
			{
				name:          "leading --debug=1 before sdk",
				args:          []string{"--debug=1", "etcd@3.5.12", "etcd", "put"},
				wantRemaining: []string{"etcd@3.5.12", "etcd", "put"},
				wantOpts:      execOptions{Debug: true, DebugSet: true},
			},
			{
				name:          "leading --debug=TRUE before sdk",
				args:          []string{"--debug=TRUE", "etcd@3.5.12", "etcd", "put"},
				wantRemaining: []string{"etcd@3.5.12", "etcd", "put"},
				wantOpts:      execOptions{Debug: true, DebugSet: true},
			},
			{
				name:          "leading --debug=0 before sdk",
				args:          []string{"--debug=0", "etcd@3.5.12", "etcd", "--debug"},
				wantRemaining: []string{"etcd@3.5.12", "etcd", "--debug"},
				wantOpts:      execOptions{Debug: false, DebugSet: true},
			},
			{
				name:          "leading --debug=false before sdk",
				args:          []string{"--debug=false", "etcd@3.5.12", "etcd", "--debug"},
				wantRemaining: []string{"etcd@3.5.12", "etcd", "--debug"},
				wantOpts:      execOptions{Debug: false, DebugSet: true},
			},
			{
				name:          "leading -debug=false before sdk",
				args:          []string{"-debug=false", "etcd@3.5.12", "etcd", "--debug"},
				wantRemaining: []string{"etcd@3.5.12", "etcd", "--debug"},
				wantOpts:      execOptions{Debug: false, DebugSet: true},
			},
			{
				name:          "leading --help",
				args:          []string{"--help"},
				wantRemaining: nil,
				wantOpts:      execOptions{ShowHelp: true},
			},
			{
				name:          "leading -help",
				args:          []string{"-help"},
				wantRemaining: nil,
				wantOpts:      execOptions{ShowHelp: true},
			},
			{
				name:          "leading --debug then -h",
				args:          []string{"--debug", "-h"},
				wantRemaining: nil,
				wantOpts:      execOptions{ShowHelp: true, Debug: true, DebugSet: true},
			},
			{
				name:          "leading --debug then -help",
				args:          []string{"--debug", "-help"},
				wantRemaining: nil,
				wantOpts:      execOptions{ShowHelp: true, Debug: true, DebugSet: true},
			},
			{
				name:          "stops at sdk so trailing flags pass to command",
				args:          []string{"etcd@3.5.12", "etcd", "-v", "-help", "--debug", "--help"},
				wantRemaining: []string{"etcd@3.5.12", "etcd", "-v", "-help", "--debug", "--help"},
				wantOpts:      execOptions{},
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				remaining, opts, err := parseExecLeadingArgs(tc.args)
				if err != nil {
					t.Fatalf("parseExecLeadingArgs(%q) unexpected error: %v", tc.args, err)
				}
				if !slices.Equal(remaining, tc.wantRemaining) {
					t.Fatalf("remaining = %q, want %q", remaining, tc.wantRemaining)
				}
				if opts != tc.wantOpts {
					t.Fatalf("opts = %+v, want %+v", opts, tc.wantOpts)
				}
			})
		}
	})

	t.Run("rejects unsupported leading flags and names the flag", func(t *testing.T) {
		for _, tc := range []struct {
			name        string
			args        []string
			wantFlag    string
			wantContain string
		}{
			{name: "-v before sdk", args: []string{"-v", "etcd@3.5.12", "etcd", "put"}, wantFlag: "-v", wantContain: "vfox --version"},
			{name: "-V before sdk", args: []string{"-V", "etcd@3.5.12", "etcd", "put"}, wantFlag: "-V", wantContain: "vfox --version"},
			{name: "--version before sdk", args: []string{"--version", "etcd@3.5.12", "etcd", "put"}, wantFlag: "--version", wantContain: "vfox --version"},
			{name: "--debug=invalid before sdk", args: []string{"--debug=invalid", "etcd@3.5.12", "etcd"}, wantFlag: "--debug=invalid"},
			{name: "--unknown after --debug", args: []string{"--debug", "--unknown", "etcd@3.5.12", "etcd"}, wantFlag: "--unknown"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				_, _, err := parseExecLeadingArgs(tc.args)
				if err == nil {
					t.Fatalf("parseExecLeadingArgs(%q) succeeded unexpectedly", tc.args)
				}
				if !strings.Contains(err.Error(), tc.wantFlag) {
					t.Fatalf("error = %q, want flag name %q", err.Error(), tc.wantFlag)
				}
				if tc.wantContain != "" && !strings.Contains(err.Error(), tc.wantContain) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tc.wantContain)
				}
				if !strings.Contains(err.Error(), "usage: vfox exec") {
					t.Fatalf("error = %q, want usage line", err.Error())
				}
			})
		}
	})
}

func TestParseExecArgsBetweenSDKAndCommand(t *testing.T) {
	for _, tc := range []struct {
		name        string
		args        []string
		wantSpecs   []execSDKSpec
		wantCommand string
		wantArgs    []string
		wantOpts    execOptions
		wantSep     bool
	}{
		{
			name:        "single sdk then --debug then command",
			args:        []string{"etcd@3.5.12", "--debug", "etcd", "put"},
			wantSpecs:   []execSDKSpec{{Name: "etcd", Version: "3.5.12"}},
			wantCommand: "etcd",
			wantArgs:    []string{"put"},
			wantOpts:    execOptions{Debug: true, DebugSet: true},
		},
		{
			name:        "single sdk then -debug then command with trailing --debug",
			args:        []string{"etcd@3.5.12", "-debug", "etcd", "--debug"},
			wantSpecs:   []execSDKSpec{{Name: "etcd", Version: "3.5.12"}},
			wantCommand: "etcd",
			wantArgs:    []string{"--debug"},
			wantOpts:    execOptions{Debug: true, DebugSet: true},
		},
		{
			name:        "single sdk then --debug=false overrides leading --debug",
			args:        []string{"--debug", "etcd@3.5.12", "--debug=false", "etcd", "put"},
			wantSpecs:   []execSDKSpec{{Name: "etcd", Version: "3.5.12"}},
			wantCommand: "etcd",
			wantArgs:    []string{"put"},
			wantOpts:    execOptions{Debug: false, DebugSet: true},
		},
		{
			name:        "multiple sdks then --debug before --",
			args:        []string{"etcd@3.5.12", "golang@1.25.6", "--debug", "--", "etcd", "--debug"},
			wantSpecs:   []execSDKSpec{{Name: "etcd", Version: "3.5.12"}, {Name: "golang", Version: "1.25.6"}},
			wantCommand: "etcd",
			wantArgs:    []string{"--debug"},
			wantOpts:    execOptions{Debug: true, DebugSet: true},
			wantSep:     true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			specs, cmdName, cmdArgs, opts, hasSep, err := parseExecArgs(tc.args)
			if err != nil {
				t.Fatalf("parseExecArgs(%q) unexpected error: %v", tc.args, err)
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
			if opts != tc.wantOpts {
				t.Fatalf("opts = %+v, want %+v", opts, tc.wantOpts)
			}
			if hasSep != tc.wantSep {
				t.Fatalf("hasSep = %v, want %v", hasSep, tc.wantSep)
			}
		})
	}
}

func TestLookPathInEnvWithDotEntry(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix '.' PATH entry lookup test")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "etcd")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	got, err := lookPathInEnvWithExt("etcd", "."+string(os.PathListSeparator)+"/nonexistent", "")
	if err != nil {
		t.Fatalf("lookPathInEnvWithExt returned error: %v", err)
	}
	want := "." + string(os.PathSeparator) + "etcd"
	if got != want {
		t.Fatalf("lookPathInEnvWithExt = %q, want %q", got, want)
	}
	if err := executeCommand("etcd", []string{"hi"}, map[string]string{"PATH": "."}); err != nil {
		t.Fatalf("executeCommand with PATH=. failed: %v", err)
	}
}
