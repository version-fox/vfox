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
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/util"
)

// `vfox exec cmake@4.5` must run the installed 4.5.0, as `vfox use cmake@4.5`
// selects it, instead of trying to install a version called 4.5.
func TestExecResolvesAPartialVersionToTheInstalledOne(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "4.5.0", "alphatool")
	f.installOnDisk(t, "alpha", "4.6.1", "alphatool")
	manager := f.manager(t)

	for _, tt := range []struct {
		typed, want sdk.Version
		why         string
	}{
		{"4.5.0", "4.5.0", "an exact version is taken as is"},
		{"4.5", "4.5.0", "a partial version stands for the installed version it extends"},
		{"4", "4.6.1", "the newest installed version wins, as with use"},
		{"4.7", "4.7", "nothing installed extends it, so it is left to be installed"},
		{"4.5.0-rc1", "4.5.0-rc1", "a longer name is not a prefix of the installed one"},
	} {
		got, err := resolveExecSDKVersion(manager, execSDKSpec{Name: "alpha", Version: tt.typed})
		if err != nil || got != tt.want {
			t.Errorf("exec alpha@%s resolved to (%q, %v), want %q: %s", tt.typed, got, err, tt.want, tt.why)
		}
	}
}

const execOutputPlugin = `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}, {version="2.0.0"}} end
function PLUGIN:PreInstall(ctx)
  print("lua-preinstall-print")
  io.write("lua-preinstall-iowrite\n")
  io.stdout:write("lua-preinstall-stdout\n")
  io.stderr:write("lua-preinstall-stderr\n")
  return {version=ctx.version}
end
function PLUGIN:PostInstall(ctx)
  print("lua-postinstall-print")
  io.write("lua-postinstall-iowrite\n")
  io.stdout:write("lua-postinstall-stdout\n")
end
function PLUGIN:EnvKeys(ctx)
  print("lua-envkeys-print")
  return {{key="PATH", value=ctx.path .. "/bin"}}
end
`

func writeExecTestTool(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, commandFile(name))
	content := "#!/bin/sh\nprintf 'cmd-stdout\\n'\nprintf 'cmd-stderr\\n' >&2\n"
	if runtime.GOOS == "windows" {
		content = "@echo off\r\necho cmd-stdout\r\necho cmd-stderr>&2\r\n"
	}
	must(t, os.WriteFile(path, []byte(content), 0755))
}

func assertExecStreamsSeparated(t *testing.T, res cliResult) {
	t.Helper()
	if got := strings.TrimSpace(res.stdout); got != "cmd-stdout" {
		t.Errorf("exec stdout = %q, want %q (stderr:\n%s)", res.stdout, "cmd-stdout", res.stderr)
	}
	mustNotContain(t, "exec stdout", res.stdout,
		"not found, installing",
		"Preinstalling",
		"success!",
		"lua-preinstall",
		"lua-postinstall",
		"lua-envkeys",
		"cmd-stderr",
	)
	mustContain(t, "exec stderr", res.stderr,
		"SDK alpha@1.0.0 not found, installing...",
		"Preinstalling alpha@1.0.0...",
		"Install ",
		"success!",
		"lua-preinstall-print",
		"lua-preinstall-iowrite",
		"lua-preinstall-stdout",
		"lua-preinstall-stderr",
		"lua-postinstall-print",
		"lua-postinstall-iowrite",
		"lua-postinstall-stdout",
		"lua-envkeys-print",
		"cmd-stderr",
	)
}

func TestExecAutoInstallSeparatesStdoutAndStderr(t *testing.T) {
	t.Run("process", func(t *testing.T) {
		f := newIsolatedCLIFixture(t, false)
		f.writePlugin(t, "alpha", execOutputPlugin)
		writeExecTestTool(t, f.system, "alphatool")

		res := f.runProcess(t, "exec", "alpha@1.0.0", "--", "alphatool").mustSucceed(t, "exec auto-install")
		assertExecStreamsSeparated(t, res)
	})

	t.Run("in-process restores streams", func(t *testing.T) {
		f := newIsolatedCLIFixture(t, false)
		f.writePlugin(t, "alpha", execOutputPlugin)
		writeExecTestTool(t, f.system, "alphatool")

		res := f.runHere(t, "exec", "alpha@1.0.0", "--", "alphatool").mustSucceed(t, "exec auto-install")
		assertExecStreamsSeparated(t, res)

		inst := f.runHere(t, "install", "alpha@2.0.0").mustSucceed(t, "subsequent install")
		mustContain(t, "install stdout after exec", inst.stdout, "Preinstalling alpha@2.0.0...", "success!")
		mustNotContain(t, "install stderr after exec", inst.stderr, "Preinstalling alpha@2.0.0...", "success!")
	})

	t.Run("error restores streams", func(t *testing.T) {
		f := newIsolatedCLIFixture(t, false)
		f.writePlugin(t, "alpha", execOutputPlugin)
		writeExecTestTool(t, f.system, "alphatool")

		f.runHere(t, "exec", "missing@1.0.0", "--", "alphatool").mustFail(t, "exec missing sdk")

		inst := f.runHere(t, "install", "alpha@2.0.0").mustSucceed(t, "install after failed exec")
		mustContain(t, "install stdout after failed exec", inst.stdout, "Preinstalling alpha@2.0.0...", "success!")
		mustNotContain(t, "install stderr after failed exec", inst.stderr, "Preinstalling alpha@2.0.0...", "success!")
	})
}

func writeStubClipboardTool(t *testing.T, binDir, markerPath string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		content := "@echo off\r\necho called>\"" + markerPath + "\"\r\n"
		must(t, os.WriteFile(filepath.Join(binDir, "clip.cmd"), []byte(content), 0755))
		return
	}
	script := "#!/bin/sh\necho called > \"" + markerPath + "\"\n"
	for _, name := range []string{"xclip", "xsel", "pbcopy"} {
		must(t, os.WriteFile(filepath.Join(binDir, name), []byte(script), 0755))
	}
}

func TestExecAutoInstallNeverWritesClipboard(t *testing.T) {
	t.Run("forced TTY cross-platform", func(t *testing.T) {
		origIsTTY := util.IsTTY
		util.IsTTY = func() bool { return true }
		t.Cleanup(func() { util.IsTTY = origIsTTY })

		f := newIsolatedCLIFixture(t, false)
		f.writePlugin(t, "alpha", execOutputPlugin)
		writeExecTestTool(t, f.system, "alphatool")
		marker := filepath.Join(f.root, "clipboard.called")
		writeStubClipboardTool(t, f.system, marker)

		res := f.runHere(t, "exec", "alpha@1.0.0", "--", "alphatool").mustSucceed(t, "exec auto-install under forced TTY")
		assertExecStreamsSeparated(t, res)
		mustNotContain(t, "exec clipboard message", res.output(), "Copied to clipboard")
		if exists(marker) {
			t.Fatalf("exec auto-install invoked clipboard utility unexpectedly (marker %q created)", marker)
		}
	})
}

func runExecSubprocess(t *testing.T, f *cliFixture, stdin string, stderrFile *os.File, args ...string) cliResult {
	t.Helper()
	exe, err := os.Executable()
	must(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := f.command(ctx, exe, append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
	cmd.Env = append(cmd.Env, "VFOX_TEST_CLI=1")
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	if stderrFile == nil {
		return runCommand(t, cmd)
	}
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = stderrFile
	res := cliResult{}
	if runErr := cmd.Run(); runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			res.code = exitErr.ExitCode()
		} else {
			t.Fatalf("%v: %v", cmd.Args, runErr)
		}
	}
	res.stdout = stdout.String()
	return res
}

func TestExecAutoInstallStdioCloseRefusalAndFileErrors(t *testing.T) {
	writeIORoundtripTool := func(t *testing.T, dir, name string) {
		t.Helper()
		path := filepath.Join(dir, commandFile(name))
		content := "#!/bin/sh\nIFS= read -r line\nprintf 'read: [%s]\\n' \"$line\"\nprintf 'cmd-stdout\\n'\nprintf 'cmd-stderr\\n' >&2\n"
		if runtime.GOOS == "windows" {
			content = "@echo off\r\nset /p line=\r\necho read: [%line%]\r\necho cmd-stdout\r\necho cmd-stderr>&2\r\n"
		}
		must(t, os.WriteFile(path, []byte(content), 0755))
	}

	for _, tc := range []struct {
		name      string
		closeStmt string
	}{
		{name: "io_stdin_close", closeStmt: `local ok, err = io.stdin:close(); assert(ok == nil and err == "cannot close standard input")`},
		{name: "io_stdout_close", closeStmt: `local ok, err = io.stdout:close(); assert(ok == nil and err == "cannot close standard output")`},
		{name: "io_stderr_close", closeStmt: `local ok, err = io.stderr:close(); assert(ok == nil and err == "cannot close standard error")`},
		{name: "io_close", closeStmt: `local ok, err = io.close(); assert(ok == nil and err == "cannot close standard output")`},
		{name: "io_output_close", closeStmt: `local ok, err = io.output():close(); assert(ok == nil and err == "cannot close standard output")`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newIsolatedCLIFixture(t, false)
			pluginSrc := `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx)
  ` + tc.closeStmt + `
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`
			f.writePlugin(t, "alpha", pluginSrc)
			writeIORoundtripTool(t, f.system, "iotool")

			res := runExecSubprocess(t, f, "hello\n", nil, "exec", "alpha@1.0.0", "--", "iotool").mustSucceed(t, tc.name)
			mustContain(t, tc.name+" stdout", res.stdout, "read: [hello]", "cmd-stdout")
			mustContain(t, tc.name+" stderr", res.stderr, "cmd-stderr")
			mustNotContain(t, tc.name+" stdout", res.stdout, "cmd-stderr", "Preinstalling")
		})
	}

	t.Run("ordinary closed file error during auto-install matches non-exec output", func(t *testing.T) {
		probeTarget := filepath.Join(t.TempDir(), "probe.txt")
		escaped := strings.ReplaceAll(probeTarget, `\`, `\\`)
		pluginSrc := `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx)
  local f = assert(io.open("` + escaped + `", "w"))
  f:close()
  f:write("x")
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`
		fExec := newIsolatedCLIFixture(t, false)
		fExec.writePlugin(t, "alpha", pluginSrc)
		writeExecTestTool(t, fExec.system, "alphatool")
		execRes := fExec.runProcess(t, "exec", "alpha@1.0.0", "--", "alphatool").mustFail(t, "exec failing PreInstall")

		fInst := newIsolatedCLIFixture(t, false)
		fInst.writePlugin(t, "alpha", pluginSrc)
		instRes := fInst.runProcess(t, "install", "alpha@1.0.0").mustFail(t, "install failing PreInstall")

		mustContain(t, "exec stderr closed-file error", execRes.stderr, "bad argument #1 to write", "file is closed")
		if strings.Contains(execRes.stderr, "(anonymous)") {
			t.Fatalf("exec stderr rewrote method name to anonymous:\n%s", execRes.stderr)
		}
		if count := strings.Count(execRes.stderr, "stack traceback:"); count != 1 {
			t.Fatalf("expected 1 stack traceback, got %d:\n%s", count, execRes.stderr)
		}
		if !strings.Contains(instRes.output(), "bad argument #1 to write") {
			t.Fatalf("install output baseline mismatch:\n%s", instRes.output())
		}
	})

	t.Run("write error on failing stderr returns nil msg 1 and hook succeeds", func(t *testing.T) {
		errSinkPath := filepath.Join(t.TempDir(), "ro-sink")
		must(t, os.WriteFile(errSinkPath, nil, 0644))
		errSink, err := os.Open(errSinkPath)
		if err != nil {
			t.Fatal(err)
		}
		defer errSink.Close()

		f := newIsolatedCLIFixture(t, false)
		pluginSrc := `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx)
  local ok1, msg1, code1 = io.write("a")
  assert(ok1 == nil and type(msg1) == "string" and code1 == 1)
  local ok2, msg2, code2 = io.stderr:write("b")
  assert(ok2 == nil and type(msg2) == "string" and code2 == 1)
  print("ignored-print")
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`
		f.writePlugin(t, "alpha", pluginSrc)
		toolPath := filepath.Join(f.system, commandFile("oktool"))
		toolSrc := "#!/bin/sh\nprintf 'cmd-stdout\\n'\n"
		if runtime.GOOS == "windows" {
			toolSrc = "@echo off\r\necho cmd-stdout\r\n"
		}
		must(t, os.WriteFile(toolPath, []byte(toolSrc), 0755))

		res := runExecSubprocess(t, f, "", errSink, "exec", "alpha@1.0.0", "--", "oktool").mustSucceed(t, "exec with unwritable stderr")
		if got := strings.TrimSpace(res.stdout); got != "cmd-stdout" {
			t.Fatalf("exec stdout = %q, want %q", res.stdout, "cmd-stdout")
		}
	})
}
