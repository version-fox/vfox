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
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

func TestExecWindowsBatchProbeProcess(t *testing.T) {
	if os.Getenv("EXEC_TEST_BATCH_PROBE") != "1" {
		return
	}
	sep := slices.Index(os.Args, "--")
	if sep < 0 {
		os.Exit(2)
	}
	outFile := os.Getenv("EXEC_TEST_BATCH_PROBE_OUT")
	if outFile != "" {
		data, err := json.Marshal(os.Args[sep+1:])
		if err != nil {
			os.Exit(3)
		}
		if err := os.WriteFile(outFile, data, 0o600); err != nil {
			os.Exit(4)
		}
	}
	os.Exit(0)
}

// When a bash/zsh hook is active, env.Paths.String() emits a POSIX-style PATH
// (":" separated, "/c/..." drive letters) meant for the shell. exec passes that
// PATH to a native Windows process, whose lookPathInEnvWithExt splits on ";",
// so the POSIX PATH becomes a single unusable entry and no SDK bin dir is searched.
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
			got, err := lookPathInEnvWithExt("vfox-probe.exe", pathValue, os.Getenv("PATHEXT"))
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

func TestExecuteCommandRefusesUnsafeBatchArgs(t *testing.T) {
	binDir := t.TempDir()
	markerDir := t.TempDir()
	ranMarker := filepath.Join(markerDir, "ran.txt")
	injectMarker := filepath.Join(markerDir, "injected.txt")

	batchPath := filepath.Join(binDir, "etcd.cmd")
	batchBody := "@echo off\r\necho ran > \"" + ranMarker + "\"\r\n"
	if err := os.WriteFile(batchPath, []byte(batchBody), 0o755); err != nil {
		t.Fatal(err)
	}

	envMap := map[string]string{
		"PATH":    binDir,
		"PATHEXT": ".COM;.EXE;.BAT;.CMD",
	}

	for _, ch := range []string{"%", "!", `"`, "\r", "\n", "\x00"} {
		arg := "safe" + ch + "echo pwned > \"" + injectMarker + "\""
		err := executeCommand("etcd", []string{arg}, envMap)
		if err == nil {
			t.Fatalf("char %q: expected error refusing unsafe batch character, got nil", ch)
		}
		if !strings.Contains(err.Error(), "refusing to pass unsafe cmd.exe character") {
			t.Fatalf("char %q: error = %q, want refusal message", ch, err.Error())
		}
		if _, statErr := os.Stat(ranMarker); !os.IsNotExist(statErr) {
			t.Fatalf("char %q: batch file ran despite refusal (stat err=%v)", ch, statErr)
		}
		if _, statErr := os.Stat(injectMarker); !os.IsNotExist(statErr) {
			t.Fatalf("char %q: second injected command ran (stat err=%v)", ch, statErr)
		}
	}

	// argv[0] paths with characters that quoting cannot protect (% / !) must also be refused.
	for _, dirName := range []string{"bad!dir", "100%dir"} {
		badDir := filepath.Join(t.TempDir(), dirName)
		if err := os.MkdirAll(badDir, 0o755); err != nil {
			t.Fatal(err)
		}
		badBatch := filepath.Join(badDir, "tool.cmd")
		if err := os.WriteFile(badBatch, []byte(batchBody), 0o755); err != nil {
			t.Fatal(err)
		}
		err := executeCommand("tool", []string{"--version"}, map[string]string{
			"PATH":    badDir,
			"PATHEXT": ".COM;.EXE;.BAT;.CMD",
		})
		if err == nil || !strings.Contains(err.Error(), "path contains unsafe cmd.exe characters") {
			t.Fatalf("dir %q: error = %v, want argv[0] refusal message", dirName, err)
		}
		if _, statErr := os.Stat(ranMarker); !os.IsNotExist(statErr) {
			t.Fatalf("dir %q: batch file ran despite argv[0] refusal", dirName)
		}
	}
}

func TestExecuteCommandQuotesBatchArgsAndAmpersandPath(t *testing.T) {
	testExe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	sdkDir := filepath.Join(t.TempDir(), "R&D with space", "bin")
	if err := os.MkdirAll(sdkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	npmCmd := filepath.Join(sdkDir, "npm.CMD")
	script := "@echo off\r\n\"" + testExe + "\" -test.run=^TestExecWindowsBatchProbeProcess$ -- %*\r\n"
	if err := os.WriteFile(npmCmd, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(t.TempDir(), "args.json")
	injectMarker := filepath.Join(t.TempDir(), "pwned.txt")
	t.Setenv("EXEC_TEST_BATCH_PROBE", "1")
	t.Setenv("EXEC_TEST_BATCH_PROBE_OUT", outFile)

	wantArgs := []string{
		"install",
		"express@^4.18.0",
		"x&echo pwned > " + injectMarker,
		"a|calc",
		"a<in",
		"a>out",
		"(grp)",
		"",
		"with space",
		`c:\dir\`,
	}

	envMap := map[string]string{
		"PATH":    sdkDir,
		"PATHEXT": ".COM;.EXE;.BAT;.CMD",
	}
	if err := executeCommand("npm", wantArgs, envMap); err != nil {
		t.Fatalf("executeCommand(npm, %v) failed: %v", wantArgs, err)
	}
	if _, statErr := os.Stat(injectMarker); !os.IsNotExist(statErr) {
		t.Fatalf("metacharacter triggered command injection: %q exists", injectMarker)
	}
	raw, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read probe output: %v", err)
	}
	var gotArgs []string
	if err := json.Unmarshal(raw, &gotArgs); err != nil {
		t.Fatalf("unmarshal probe output %q: %v", string(raw), err)
	}
	if !slices.Equal(gotArgs, wantArgs) {
		t.Fatalf("got args = %#v, want %#v", gotArgs, wantArgs)
	}

	// Relative batch path containing '&' via PATH=.
	if err := os.Remove(outFile); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sdkDir)
	relWant := []string{"express@^4.18.0", "a&b"}
	if err := executeCommand(`.\npm.CMD`, relWant, map[string]string{"PATH": sdkDir}); err != nil {
		t.Fatalf("executeCommand(.\\npm.CMD) failed: %v", err)
	}
	raw, err = os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read relative probe output: %v", err)
	}
	gotArgs = nil
	if err := json.Unmarshal(raw, &gotArgs); err != nil {
		t.Fatalf("unmarshal relative probe output %q: %v", string(raw), err)
	}
	if !slices.Equal(gotArgs, relWant) {
		t.Fatalf("relative got args = %#v, want %#v", gotArgs, relWant)
	}
}
