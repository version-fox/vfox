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

func TestInstallAndExecIgnoreUnwritableStdioStreams(t *testing.T) {
	unwritablePlugin := `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx)
  print("ignored-print-error")
  local ok, msg, errno = io.write("ignored-io-write")
  assert(ok == nil and type(msg) == "string" and type(errno) == "number", "expected io.write error tuple")
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`

	runWithReadOnlyStream := func(t *testing.T, redirectStderr bool, args ...string) string {
		t.Helper()
		f := newIsolatedCLIFixture(t, false)
		f.writePlugin(t, "alpha", unwritablePlugin)
		stdoutOnlyTool := filepath.Join(f.system, commandFile("stdouttool"))
		toolScript := "#!/bin/sh\nprintf 'ok\\n'\n"
		if runtime.GOOS == "windows" {
			toolScript = "@echo off\r\necho ok\r\n"
		}
		must(t, os.WriteFile(stdoutOnlyTool, []byte(toolScript), 0755))

		roPath := filepath.Join(f.root, "readonly-stream")
		must(t, os.WriteFile(roPath, []byte("ro"), 0644))
		roFile, err := os.Open(roPath)
		must(t, err)
		defer roFile.Close()

		exe, err := os.Executable()
		must(t, err)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		cmd := f.command(ctx, exe, append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
		cmd.Env = append(cmd.Env, "VFOX_TEST_CLI=1")
		var outBuf, errBuf bytes.Buffer
		if redirectStderr {
			cmd.Stdout = &outBuf
			cmd.Stderr = roFile
		} else {
			cmd.Stdout = roFile
			cmd.Stderr = &errBuf
		}
		if err := cmd.Run(); err != nil {
			t.Fatalf("vfox %v failed with unwritable stream: %v\nstdout:\n%s\nstderr:\n%s", args, err, outBuf.String(), errBuf.String())
		}
		return outBuf.String()
	}

	t.Run("install with unwritable stdout", func(t *testing.T) {
		_ = runWithReadOnlyStream(t, false, "install", "alpha@1.0.0")
	})

	t.Run("exec with unwritable stderr", func(t *testing.T) {
		gotStdout := runWithReadOnlyStream(t, true, "exec", "alpha@1.0.0", "--", "stdouttool")
		if got := strings.TrimSpace(gotStdout); got != "ok" {
			t.Fatalf("exec stdout = %q, want %q", gotStdout, "ok")
		}
	})
}

func TestExecPreservesStdinWhenPluginClosesStdin(t *testing.T) {
	f := newIsolatedCLIFixture(t, false)
	f.writePlugin(t, "alpha", `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx)
  local ok1, msg1 = io.stdin:close()
  assert(ok1 == nil and msg1 == "cannot close standard input", tostring(msg1))
  local ok2, msg2 = io.close(io.stdin)
  assert(ok2 == nil and msg2 == "cannot close standard input", tostring(msg2))
  local ok3, msg3 = io.input():close()
  assert(ok3 == nil and msg3 == "cannot close standard input", tostring(msg3))
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`)
	readTool := filepath.Join(f.system, commandFile("readtool"))
	toolScript := "#!/bin/sh\nIFS= read -r line\nprintf 'read: [%s]\\n' \"$line\"\n"
	if runtime.GOOS == "windows" {
		toolScript = "@echo off\r\nset \"line=\"\r\nset /p line=\r\necho read: [%line%]\r\n"
	}
	must(t, os.WriteFile(readTool, []byte(toolScript), 0755))

	exe, err := os.Executable()
	must(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := f.command(ctx, exe, "-test.run=^TestCLIProcess$", "--", "exec", "alpha@1.0.0", "--", "readtool")
	cmd.Env = append(cmd.Env, "VFOX_TEST_CLI=1")
	cmd.Stdin = strings.NewReader("hello-from-stdin\n")
	res := runCommand(t, cmd).mustSucceed(t, "exec alpha@1.0.0 -- readtool")
	if got := strings.TrimSpace(res.stdout); got != "read: [hello-from-stdin]" {
		t.Fatalf("exec stdout = %q, want %q\nstderr:\n%s", res.stdout, "read: [hello-from-stdin]", res.stderr)
	}
}
