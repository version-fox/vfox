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
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/pathmeta"
)

func TestReadOnlyAndBlockedVfoxHomeReadPaths(t *testing.T) {
	t.Run("activate produces identical output across runs", func(t *testing.T) {
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		first := f.runProcess(t, "activate", "bash").mustSucceed(t, "first activate bash")
		second := f.runProcess(t, "activate", "bash").mustSucceed(t, "second activate bash")
		if first.stdout != second.stdout {
			t.Fatalf("activate output differed across runs:\nfirst:\n%s\nsecond:\n%s", first.stdout, second.stdout)
		}
	})

	t.Run("env does not create missing nested VFOX_HOME directories", func(t *testing.T) {
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		typoRoot := filepath.Join(f.root, "typo")
		f.env["VFOX_HOME"] = filepath.Join(typoRoot, "a", "b")

		f.runProcess(t, "env", "-s", "bash").mustSucceed(t, "env -s bash with missing nested VFOX_HOME")
		if _, err := os.Stat(typoRoot); !os.IsNotExist(err) {
			t.Fatalf("expected env read path not to create %q, stat err = %v", typoRoot, err)
		}
	})

	t.Run("env stdout stays clean shell code when GlobalSdkDir cannot be created", func(t *testing.T) {
		fClean := newCLIFixtureWithoutRegistryGuard(t, false)
		cleanEnv := fClean.runProcess(t, "env", "-s", "bash").mustSucceed(t, "baseline env -s bash")

		fBlocked := newCLIFixtureWithoutRegistryGuard(t, false)
		must(t, os.WriteFile(filepath.Join(fBlocked.userHome, "sdks"), []byte("file"), 0644))
		blockedEnv := fBlocked.runProcess(t, "env", "-s", "bash").mustSucceed(t, "env -s bash with blocked GlobalSdkDir")

		want := strings.ReplaceAll(cleanEnv.stdout, shellPath(fClean.root), shellPath(fBlocked.root))
		want = strings.ReplaceAll(want, fClean.root, fBlocked.root)
		if blockedEnv.stdout != want {
			t.Fatalf("env stdout changed when GlobalSdkDir could not be created:\ngot:\n%s\nwant:\n%s", blockedEnv.stdout, want)
		}
	})

	t.Run("env and activate write temp-dir diagnostic to stderr while keeping stdout clean", func(t *testing.T) {
		fClean := newCLIFixtureWithoutRegistryGuard(t, false)
		cleanEnv := fClean.runProcess(t, "env", "-s", "bash").mustSucceed(t, "baseline env -s bash")

		fBlocked := newCLIFixtureWithoutRegistryGuard(t, false)
		fBlocked.env["__VFOX_CURTMPPATH"] = "-"
		must(t, os.WriteFile(filepath.Join(fBlocked.userHome, "tmp"), []byte("file"), 0644))
		blockedEnv := fBlocked.runProcess(t, "env", "-s", "bash").mustSucceed(t, "env -s bash with blocked session temp dir")

		want := strings.ReplaceAll(cleanEnv.stdout, shellPath(fClean.root), shellPath(fBlocked.root))
		want = strings.ReplaceAll(want, fClean.root, fBlocked.root)
		if blockedEnv.stdout != want {
			t.Fatalf("env stdout changed when session temp dir could not be created:\ngot:\n%s\nwant:\n%s", blockedEnv.stdout, want)
		}
		mustContain(t, "env stderr", blockedEnv.stderr, "create temp dir failed:")
		mustNotContain(t, "env stdout", blockedEnv.stdout, "create temp dir failed:")

		blockedAct := fBlocked.runProcess(t, "activate", "bash").mustSucceed(t, "activate bash with blocked session temp dir")
		mustContain(t, "activate stderr", blockedAct.stderr, "create temp dir failed:")
		mustNotContain(t, "activate stdout", blockedAct.stdout, "create temp dir failed:")
	})

	t.Run("activate writes executable lookup diagnostic to stderr and not stdout", func(t *testing.T) {
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		for k, v := range f.environ() {
			if v == "-" {
				t.Setenv(k, "")
			} else {
				t.Setenv(k, v)
			}
		}
		origExe := pathmeta.Executable
		pathmeta.Executable = func() (string, error) {
			return "", errors.New("readlink /proc/self/exe: no such file or directory")
		}
		t.Cleanup(func() { pathmeta.Executable = origExe })

		origStdout, origStderr := os.Stdout, os.Stderr
		rOut, wOut, err := os.Pipe()
		must(t, err)
		rErr, wErr, err := os.Pipe()
		must(t, err)
		os.Stdout, os.Stderr = wOut, wErr

		runErr := newTestApp().Run(context.Background(), []string{"vfox", "activate", "bash"})
		_ = wOut.Close()
		_ = wErr.Close()
		os.Stdout, os.Stderr = origStdout, origStderr

		var out, errBuf bytes.Buffer
		_, _ = io.Copy(&out, rOut)
		_, _ = io.Copy(&errBuf, rErr)
		_ = rOut.Close()
		_ = rErr.Close()

		if runErr != nil {
			t.Fatalf("activate bash returned unexpected error: %v", runErr)
		}
		mustContain(t, "activate stderr", errBuf.String(), "get executable path failed:")
		mustNotContain(t, "activate stdout", out.String(), "get executable path failed:")
	})

	t.Run("read-only VFOX_HOME without cache or plugin keeps env and activate identical", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("POSIX chmod 0555 does not block owner writes on Windows or root")
		}
		fBase := newCLIFixtureWithoutRegistryGuard(t, false)
		roEmpty := filepath.Join(fBase.root, "ro-vfox-empty")
		must(t, os.MkdirAll(roEmpty, 0755))
		fBase.env["VFOX_HOME"] = roEmpty
		wantEnv := fBase.runProcess(t, "env", "-s", "bash").mustSucceed(t, "writable empty env")
		wantAct := fBase.runProcess(t, "activate", "bash").mustSucceed(t, "writable empty activate")

		fRO := newCLIFixtureWithoutRegistryGuard(t, false)
		roHome := filepath.Join(fRO.root, "ro-vfox")
		must(t, os.MkdirAll(roHome, 0555))
		t.Cleanup(func() { _ = os.Chmod(roHome, 0755) })
		fRO.env["VFOX_HOME"] = roHome

		gotEnv := fRO.runProcess(t, "env", "-s", "bash").mustSucceed(t, "read-only VFOX_HOME env")
		gotAct := fRO.runProcess(t, "activate", "bash").mustSucceed(t, "read-only VFOX_HOME activate")

		norm := func(s string) string {
			s = strings.ReplaceAll(s, shellPath(roEmpty), shellPath(roHome))
			s = strings.ReplaceAll(s, roEmpty, roHome)
			s = strings.ReplaceAll(s, shellPath(fBase.root), shellPath(fRO.root))
			return strings.ReplaceAll(s, fBase.root, fRO.root)
		}
		if expectedEnv := norm(wantEnv.stdout); gotEnv.stdout != expectedEnv {
			t.Fatalf("env output mismatch on read-only VFOX_HOME:\ngot:\n%s\nwant:\n%s", gotEnv.stdout, expectedEnv)
		}
		if expectedAct := norm(wantAct.stdout); gotAct.stdout != expectedAct {
			t.Fatalf("activate output mismatch on read-only VFOX_HOME:\ngot:\n%s\nwant:\n%s", gotAct.stdout, expectedAct)
		}
	})

	t.Run("env writes deleted cwd diagnostic to stderr while keeping stdout clean shell code", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows holds an open handle on a process working directory")
		}
		fClean := newCLIFixtureWithoutRegistryGuard(t, false)
		cleanEnv := fClean.runProcess(t, "env", "-s", "bash").mustSucceed(t, "baseline env -s bash")

		fDeleted := newCLIFixtureWithoutRegistryGuard(t, false)
		goneDir := filepath.Join(fDeleted.root, "deleted-work")
		must(t, os.MkdirAll(goneDir, 0755))

		exe, err := os.Executable()
		must(t, err)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cmd := fDeleted.command(
			ctx,
			"/bin/sh",
			"-c",
			`cd "$1" && rmdir "$1" && exec "$2" -test.run=^TestCLIProcess$ -- env -s bash`,
			"sh",
			goneDir,
			exe,
		)
		cmd.Env = append(cmd.Env, "VFOX_TEST_CLI=1")
		deletedEnv := runCommand(t, cmd).mustSucceed(t, "env -s bash with deleted cwd")

		want := strings.ReplaceAll(cleanEnv.stdout, shellPath(fClean.root), shellPath(fDeleted.root))
		want = strings.ReplaceAll(want, fClean.root, fDeleted.root)
		if deletedEnv.stdout != want {
			t.Fatalf("env stdout changed when cwd was deleted:\ngot:\n%s\nwant:\n%s", deletedEnv.stdout, want)
		}
		mustContain(t, "env stderr", deletedEnv.stderr, "get current working directory failed:")
		mustNotContain(t, "env stdout", deletedEnv.stdout, "get current working directory failed:")
	})

	t.Run("read-only VFOX_HOME honors storage.sdkPath override", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("POSIX chmod 0555 does not block owner writes on Windows or root")
		}
		f := newCLIFixtureWithoutRegistryGuard(t, true)
		overrideDir := filepath.Join(f.root, "custom-sdks")
		sdkBin := filepath.Join(overrideDir, "alpha", "v-1.0.0", "alpha-1.0.0", "bin")
		must(t, os.MkdirAll(sdkBin, 0755))
		writeCommand(t, sdkBin, "alpha-tool")

		// Seed only plugin/ in shared VFOX_HOME (no cache/) and make VFOX_HOME read-only.
		must(t, os.RemoveAll(filepath.Join(f.shared, "cache")))
		must(t, os.Chmod(f.shared, 0555))
		t.Cleanup(func() { _ = os.Chmod(f.shared, 0755) })

		f.writeUserConfig(t, "storage:\n  sdkPath: "+overrideDir+"\n")
		must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))

		envRes := f.runProcess(t, "env", "-s", "bash").mustSucceed(t, "env with read-only VFOX_HOME and storage.sdkPath")
		if !strings.Contains(envRes.stdout, "alpha") {
			t.Fatalf("expected PATH export to include alpha from storage.sdkPath override, got:\n%s", envRes.stdout)
		}
		listRes := f.runHere(t, "list", "alpha").mustSucceed(t, "list alpha with storage.sdkPath")
		mustContain(t, "listed version under storage.sdkPath", listRes.output(), "1.0.0")
	})
}
