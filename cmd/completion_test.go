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
)

func TestCompletionScriptsUseCurrentCliFlag(t *testing.T) {
	t.Parallel()

	scriptPaths := []string{
		filepath.Join("..", "completions", "bash_autocomplete"),
		filepath.Join("..", "completions", "zsh_autocomplete"),
		filepath.Join("..", "completions", "powershell_autocomplete.ps1"),
		filepath.Join("..", "completions", "vfox.fish"),
	}

	for _, scriptPath := range scriptPaths {
		scriptPath := scriptPath

		t.Run(filepath.Base(scriptPath), func(t *testing.T) {
			t.Parallel()

			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("read completion script %q: %v", scriptPath, err)
			}

			script := string(content)
			if strings.Contains(script, "--generate-bash-completion") {
				t.Fatalf("completion script %q still references deprecated completion flag", scriptPath)
			}
			if !strings.Contains(script, "--generate-shell-completion") {
				t.Fatalf("completion script %q does not reference current completion flag", scriptPath)
			}
		})
	}
}

func TestGenerateShellCompletionFlagProducesSuggestions(t *testing.T) {
	t.Parallel()

	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(executable, "-test.run=^TestShellCompletionProcess$", "--", "--generate-shell-completion")
	command.Env = append(os.Environ(), "VFOX_COMPLETION_TEST_PROCESS=1", "SHELL=fish")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("run shell completion: %v\n%s", err, output)
	}

	suggestions := string(output)
	if !strings.Contains(suggestions, "install") {
		t.Fatalf("shell completion output missing install command: %q", suggestions)
	}
	if !strings.Contains(suggestions, "use") {
		t.Fatalf("shell completion output missing use command: %q", suggestions)
	}
}

// TestShellCompletionProcess runs the real CLI in a separate process so that
// urfave/cli sees the same os.Args as it would when invoked by a shell.
func TestShellCompletionProcess(t *testing.T) {
	if os.Getenv("VFOX_COMPLETION_TEST_PROCESS") != "1" {
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

func TestFishCompletion(t *testing.T) {
	script, err := filepath.Abs(filepath.Join("..", "completions", "vfox.fish"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Fatal(err)
	}
	fish, err := exec.LookPath("fish")
	if err != nil {
		t.Skip("fish is not installed; install fish to run native completion tests")
	}
	dir := installCompletionTestCLI(t)
	// Fish can be started from Zsh: completion output must not inherit its
	// colon-separated description format.
	t.Setenv("SHELL", "/bin/zsh")

	for _, tt := range []struct {
		name    string
		line    string
		want    string
		absent  []string
		noCalls bool
	}{
		{name: "commands", line: "vfox ", want: "install", absent: []string{"activate", "completion"}},
		{name: "partial command", line: "vfox unins", want: "uninstall"},
		{name: "global flag", line: "vfox --de", want: "--debug"},
		{name: "double dash flag prefix", line: "vfox --", want: "--help"},
		{name: "install flag", line: "vfox install --y", want: "--yes"},
		{name: "command alias", line: "vfox i --a", want: "--all"},
		{name: "scope flag", line: "vfox use --g", want: "--global"},
		{name: "global option before command", line: "vfox --debug use --s", want: "--session"},
		{name: "flag after SDK argument", line: "vfox use nodejs --p", want: "--project"},
		{name: "quoted argument", line: "vfox info 'sdk with spaces' --f", want: "--format"},
		{name: "after separator", line: "vfox exec nodejs -- echo --", noCalls: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			trace := filepath.Join(t.TempDir(), "calls")
			t.Setenv("VFOX_COMPLETION_TEST_TRACE", trace)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, fish, "--no-config", "-c", `source $argv[1]; complete --do-complete "$argv[2]"`, "--", script, tt.line)
			command.Dir = dir
			var stdout, stderr bytes.Buffer
			command.Stdout = &stdout
			command.Stderr = &stderr
			if err := command.Run(); err != nil {
				t.Fatalf("fish completion: %v\n%s", err, stderr.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("fish completion emitted errors: %s", stderr.String())
			}
			var suggestions []string
			for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
				candidate, _, _ := strings.Cut(line, "\t")
				suggestions = append(suggestions, candidate)
			}
			if tt.want != "" && !slices.Contains(suggestions, tt.want) {
				t.Errorf("completion for %q = %q, want %q", tt.line, suggestions, tt.want)
			}
			for _, candidate := range tt.absent {
				if slices.Contains(suggestions, candidate) {
					t.Errorf("completion for %q includes hidden command %q", tt.line, candidate)
				}
			}
			if tt.noCalls {
				if _, err := os.Stat(trace); !os.IsNotExist(err) {
					t.Errorf("completion after -- invoked vfox: %v", err)
				}
			}
		})
	}
}

func installCompletionTestCLI(t *testing.T) string {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0755); err != nil {
		t.Fatal(err)
	}
	// The wrapper only dispatches the test executable; it never evaluates the
	// text being completed as shell code.
	wrapper := "#!/bin/sh\nprintf '%s\\n' called >> \"$VFOX_COMPLETION_TEST_TRACE\"\nexec \"$VFOX_COMPLETION_TEST_EXECUTABLE\" -test.run='^TestShellCompletionProcess$' -- \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "vfox"), []byte(wrapper), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	t.Setenv("VFOX_HOME", filepath.Join(dir, "vfox"))
	t.Setenv("VFOX_COMPLETION_TEST_PROCESS", "1")
	t.Setenv("VFOX_COMPLETION_TEST_EXECUTABLE", executable)
	t.Setenv("VFOX_COMPLETION_TEST_TRACE", filepath.Join(dir, "calls"))
	return dir
}

func TestBashZshCompletionShellFormat(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX completion adapters are tested with native Bash and Zsh on Unix")
	}
	for _, shell := range []struct {
		name    string
		parent  string
		program string
		command string
		flag    string
	}{
		{
			name: "bash", parent: "/bin/zsh", command: "install", flag: "--global",
			program: `source "$1"
shift
_init_completion() { words=("${COMP_WORDS[@]}"); cword=$COMP_CWORD; cur=${COMP_WORDS[COMP_CWORD]}; }
COMP_WORDS=(vfox "$@")
COMP_CWORD=$((${#COMP_WORDS[@]} - 1))
__vfox_bash_autocomplete
printf '%s\n' "${COMPREPLY[@]}"`,
		},
		{
			name: "zsh", parent: "/bin/bash", command: "install:Install a version of the target SDK", flag: "--global:Used with the global environment",
			program: `compdef() { :; }
_describe() { printf '%s\n' "${opts[@]}"; }
_files() { :; }
source "$1"
shift
words=(vfox "$@")
_vfox`,
		},
	} {
		t.Run(shell.name, func(t *testing.T) {
			binary, err := exec.LookPath(shell.name)
			if err != nil {
				t.Skipf("%s is not installed", shell.name)
			}
			script, err := filepath.Abs(filepath.Join("..", "completions", shell.name+"_autocomplete"))
			if err != nil {
				t.Fatal(err)
			}
			dir := installCompletionTestCLI(t)
			t.Setenv("SHELL", shell.parent)
			for _, tt := range []struct {
				args []string
				want string
			}{
				{args: []string{""}, want: shell.command},
				{args: []string{"use", "--g"}, want: shell.flag},
			} {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				args := append([]string{"-f", "-c", shell.program, "completion-test", script}, tt.args...)
				command := exec.CommandContext(ctx, binary, args...)
				command.Dir = dir
				output, err := command.CombinedOutput()
				if err != nil {
					t.Fatalf("%s completion: %v\n%s", shell.name, err, output)
				}
				if !slices.Contains(strings.Split(strings.TrimSpace(string(output)), "\n"), tt.want) {
					t.Errorf("%s completion under SHELL=%s = %q, want %q", shell.name, shell.parent, output, tt.want)
				}
			}
		})
	}
}
