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
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/sdk"
	"github.com/version-fox/vfox/internal/shared/logger"
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

		res := f.runProcess(t, "env", "-s", "bash").mustSucceed(t, "env -s bash with missing nested VFOX_HOME")
		if res.stderr != "" {
			t.Fatalf("env -s bash wrote unexpected stderr on missing VFOX_HOME: %q", res.stderr)
		}
		if _, err := os.Stat(typoRoot); !os.IsNotExist(err) {
			t.Fatalf("expected env read path not to create %q, stat err = %v", typoRoot, err)
		}

		debugOut, debugErr := runInProcessCaptured(t, f, logger.DebugLevel, "env", "-s", "bash")
		if debugErr != "" {
			t.Fatalf("env -s bash wrote unexpected stderr at debug level: %q", debugErr)
		}
		mustNotContain(t, "debug output for missing VFOX_HOME", debugOut, "create shared directory")
		if _, err := os.Stat(typoRoot); !os.IsNotExist(err) {
			t.Fatalf("expected debug env read path not to create %q, stat err = %v", typoRoot, err)
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
		if blockedEnv.stderr != "" {
			t.Fatalf("env wrote unexpected stderr when GlobalSdkDir was blocked: %q", blockedEnv.stderr)
		}
	})

	t.Run("env and activate keep stdout and stderr clean when session temp dir cannot be created", func(t *testing.T) {
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
		if blockedEnv.stderr != "" {
			t.Fatalf("env wrote unexpected stderr on blocked session temp dir: %q", blockedEnv.stderr)
		}
		mustNotContain(t, "env stdout", blockedEnv.stdout, "create temp dir failed:")

		blockedAct := fBlocked.runProcess(t, "activate", "bash").mustSucceed(t, "activate bash with blocked session temp dir")
		if blockedAct.stderr != "" {
			t.Fatalf("activate wrote unexpected stderr on blocked session temp dir: %q", blockedAct.stderr)
		}
		mustNotContain(t, "activate stdout", blockedAct.stdout, "create temp dir failed:")

		debugEnvOut, debugEnvErr := runInProcessCaptured(t, fBlocked, logger.DebugLevel, "env", "-s", "bash")
		if debugEnvErr != "" {
			t.Fatalf("env debug run wrote unexpected stderr: %q", debugEnvErr)
		}
		mustContain(t, "env debug output", debugEnvOut, "create temp dir failed:")

		debugActOut, debugActErr := runInProcessCaptured(t, fBlocked, logger.DebugLevel, "activate", "bash")
		if debugActErr != "" {
			t.Fatalf("activate debug run wrote unexpected stderr: %q", debugActErr)
		}
		mustContain(t, "activate debug output", debugActOut, "create temp dir failed:")

		// Activated shell case: __VFOX_CURTMPPATH points inside the blocked tmp dir.
		fActivatedBlocked := newCLIFixtureWithoutRegistryGuard(t, false)
		must(t, os.WriteFile(filepath.Join(fActivatedBlocked.userHome, "tmp"), []byte("file"), 0644))
		fActivatedBlocked.env["__VFOX_CURTMPPATH"] = filepath.Join(fActivatedBlocked.userHome, "tmp", "activated-session")
		actDebugEnvOut, actDebugEnvErr := runInProcessCaptured(t, fActivatedBlocked, logger.DebugLevel, "env", "-s", "bash")
		if actDebugEnvErr != "" {
			t.Fatalf("activated env debug run wrote unexpected stderr: %q", actDebugEnvErr)
		}
		if count := strings.Count(actDebugEnvOut, "create temp dir failed:"); count != 1 {
			t.Fatalf("expected create temp dir failed once in activated debug env, got %d:\n%s", count, actDebugEnvOut)
		}
	})

	t.Run("doctor and use report blocked temp directory while env stays clean", func(t *testing.T) {
		f := newCLIFixtureWithoutRegistryGuard(t, true)
		f.runProcess(t, "install", "alpha@1.0.0").mustSucceed(t, "install alpha@1.0.0")
		tmpPath := filepath.Join(f.userHome, "tmp")
		must(t, os.RemoveAll(tmpPath))
		must(t, os.WriteFile(tmpPath, []byte("file"), 0644))
		f.env["__VFOX_CURTMPPATH"] = filepath.Join(tmpPath, "session-1")

		docRes := f.runProcess(t, "doctor", "--json").mustFail(t, "doctor --json with blocked tmp")
		mustContain(t, "doctor --json output", docRes.stdout, sdk.FindingPathUnusable, "user temp directory")

		useRes := f.runProcess(t, "use", "-s", "alpha@1.0.0").mustFail(t, "use -s with blocked tmp")
		mustContain(t, "use output", useRes.output(), "Cannot use user temp directory")
	})

	t.Run("activate keeps stdout and stderr clean on normal runs and logs executable diagnostic at debug level", func(t *testing.T) {
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		origExe := pathmeta.Executable
		pathmeta.Executable = func() (string, error) {
			return "", errors.New("readlink /proc/self/exe: no such file or directory")
		}
		t.Cleanup(func() { pathmeta.Executable = origExe })

		normalOut, normalErr := runInProcessCaptured(t, f, logger.InfoLevel, "activate", "bash")
		if normalErr != "" {
			t.Fatalf("activate bash wrote unexpected stderr on executable lookup failure: %q", normalErr)
		}
		mustNotContain(t, "activate stdout", normalOut, "get executable path failed:")

		debugOut, debugErr := runInProcessCaptured(t, f, logger.DebugLevel, "activate", "bash")
		if debugErr != "" {
			t.Fatalf("activate bash debug run wrote unexpected stderr: %q", debugErr)
		}
		mustContain(t, "activate debug output", debugOut, "get executable path failed:")
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

	t.Run("doctor and use in activated shell succeed without path.unusable on read-only VFOX_HOME", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("POSIX chmod 0555 does not block owner writes on Windows or root")
		}

		// Case 1: read-only shared root with cache/ existing and plugin/ missing.
		fNoPlugin := newCLIFixtureWithoutRegistryGuard(t, false)
		roNoPlugin := filepath.Join(fNoPlugin.root, "ro-no-plugin")
		must(t, os.MkdirAll(filepath.Join(roNoPlugin, "cache"), 0755))
		must(t, os.Chmod(roNoPlugin, 0555))
		t.Cleanup(func() { _ = os.Chmod(roNoPlugin, 0755) })
		fNoPlugin.env["VFOX_HOME"] = roNoPlugin
		fNoPlugin.env["__VFOX_SHELL"] = "bash"

		doc := fNoPlugin.runProcess(t, "doctor").mustSucceed(t, "doctor on read-only VFOX_HOME without plugin/")
		mustContain(t, "doctor output", doc.output(), "No problems found.")
		mustNotContain(t, "doctor output", doc.output(), sdk.FindingPathUnusable, "Cannot use shared directory")

		docJSON := fNoPlugin.runProcess(t, "doctor", "--json").mustSucceed(t, "doctor --json on read-only VFOX_HOME without plugin/")
		mustNotContain(t, "doctor --json output", docJSON.stdout, sdk.FindingPathUnusable, "shared directory")

		useMissing := fNoPlugin.runProcess(t, "use", "-s", "alpha@1.0.0")
		mustNotContain(t, "use output without plugin/", useMissing.output(), sdk.FindingPathUnusable, "Cannot use shared directory")

		// Case 2: read-only shared root with plugin/ existing, cache/ missing, and custom storage.sdkPath.
		fNoCache := newCLIFixtureWithoutRegistryGuard(t, false)
		fNoCache.runProcess(t, "install", "alpha@1.0.0").mustSucceed(t, "install alpha@1.0.0")
		roNoCache := filepath.Join(fNoCache.root, "ro-no-cache")
		must(t, os.MkdirAll(filepath.Join(roNoCache, "plugin", "alpha"), 0755))
		pluginSrc, err := os.ReadFile(filepath.Join(fNoCache.shared, "plugin", "alpha", "main.lua"))
		must(t, err)
		must(t, os.WriteFile(filepath.Join(roNoCache, "plugin", "alpha", "main.lua"), pluginSrc, 0644))
		sdkStore := filepath.Join(fNoCache.shared, "cache")
		cfgYAML := "storage:\n  sdkPath: '" + strings.ReplaceAll(sdkStore, "'", "''") + "'\n"
		must(t, os.WriteFile(filepath.Join(fNoCache.userHome, "config.yaml"), []byte(cfgYAML), 0644))
		must(t, os.Chmod(roNoCache, 0555))
		t.Cleanup(func() { _ = os.Chmod(roNoCache, 0755) })
		fNoCache.env["VFOX_HOME"] = roNoCache
		fNoCache.env["__VFOX_SHELL"] = "bash"

		docWithPlugin := fNoCache.runProcess(t, "doctor").mustSucceed(t, "doctor on read-only VFOX_HOME without cache/")
		mustContain(t, "doctor output with plugin/", docWithPlugin.output(), "No problems found.")
		mustNotContain(t, "doctor output with plugin/", docWithPlugin.output(), sdk.FindingPathUnusable, "Cannot use shared directory")

		docWithPluginJSON := fNoCache.runProcess(t, "doctor", "--json").mustSucceed(t, "doctor --json on read-only VFOX_HOME without cache/")
		mustNotContain(t, "doctor --json output with plugin/", docWithPluginJSON.stdout, sdk.FindingPathUnusable, "shared directory")

		useOK := fNoCache.runProcess(t, "use", "-s", "alpha@1.0.0").mustSucceed(t, "use -s on read-only VFOX_HOME without cache/")
		mustNotContain(t, "use output without cache/", useOK.output(), sdk.FindingPathUnusable, "Cannot use shared directory")
	})

	t.Run("env keeps stdout and stderr clean when cwd was deleted", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("Linux reliably fails getcwd on an unlinked cwd; darwin caches F_GETPATH and Windows locks open cwd handles")
		}
		fClean := newCLIFixtureWithoutRegistryGuard(t, false)
		cleanEnv := fClean.runProcess(t, "env", "-s", "bash").mustSucceed(t, "baseline env -s bash")

		fDeleted := newCLIFixtureWithoutRegistryGuard(t, false)
		goneDir := filepath.Join(fDeleted.root, "deleted-work")
		must(t, os.MkdirAll(goneDir, 0755))
		fDeleted.env["VFOX_TEST_DELETED_CWD"] = goneDir
		deletedEnv := fDeleted.runProcess(t, "env", "-s", "bash").mustSucceed(t, "env -s bash with deleted cwd")

		want := strings.ReplaceAll(cleanEnv.stdout, shellPath(fClean.root), shellPath(fDeleted.root))
		want = strings.ReplaceAll(want, fClean.root, fDeleted.root)
		if deletedEnv.stdout != want {
			t.Fatalf("env stdout changed when cwd was deleted:\ngot:\n%s\nwant:\n%s", deletedEnv.stdout, want)
		}
		if deletedEnv.stderr != "" {
			t.Fatalf("env wrote unexpected stderr when cwd was deleted: %q", deletedEnv.stderr)
		}
		mustNotContain(t, "env stdout", deletedEnv.stdout, "get current working directory failed:")
	})

	t.Run("env keeps stdout and stderr clean on getwd failure and logs diagnostic at debug level", func(t *testing.T) {
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		baseline := f.runProcess(t, "env", "-s", "bash").mustSucceed(t, "baseline env -s bash")

		origGetwd := internal.Getwd
		internal.Getwd = func() (string, error) {
			return "", errors.New("getwd: no such file or directory")
		}
		t.Cleanup(func() { internal.Getwd = origGetwd })

		normalOut, normalErr := runInProcessCaptured(t, f, logger.InfoLevel, "env", "-s", "bash")
		if normalErr != "" {
			t.Fatalf("env -s bash wrote unexpected stderr on getwd failure: %q", normalErr)
		}
		if normalOut != baseline.stdout {
			t.Fatalf("env stdout changed on getwd failure:\ngot:\n%s\nwant:\n%s", normalOut, baseline.stdout)
		}
		mustNotContain(t, "env stdout", normalOut, "get current working directory failed:")

		debugOut, debugErr := runInProcessCaptured(t, f, logger.DebugLevel, "env", "-s", "bash")
		if debugErr != "" {
			t.Fatalf("env -s bash debug run wrote unexpected stderr on getwd failure: %q", debugErr)
		}
		mustContain(t, "env debug output", debugOut, "get current working directory failed: getwd: no such file or directory\n")
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

func runInProcessCaptured(t *testing.T, f *cliFixture, level logger.LoggerLevel, args ...string) (string, string) {
	t.Helper()
	for k, v := range f.environ() {
		if v == "-" {
			t.Setenv(k, "")
		} else {
			t.Setenv(k, v)
		}
	}
	t.Chdir(f.work)
	logger.SetLevel(level)
	t.Cleanup(func() { logger.SetLevel(logger.InfoLevel) })

	tempDir := t.TempDir()
	outFile, err := os.CreateTemp(tempDir, "stdout")
	must(t, err)
	errFile, err := os.CreateTemp(tempDir, "stderr")
	must(t, err)

	origStdout, origStderr := os.Stdout, os.Stderr
	t.Cleanup(func() {
		os.Stdout, os.Stderr = origStdout, origStderr
		_ = outFile.Close()
		_ = errFile.Close()
	})
	os.Stdout, os.Stderr = outFile, errFile

	runErr := newTestApp().Run(context.Background(), append([]string{"vfox"}, args...))
	os.Stdout, os.Stderr = origStdout, origStderr
	must(t, outFile.Close())
	must(t, errFile.Close())

	outBytes, err := os.ReadFile(outFile.Name())
	must(t, err)
	errBytes, err := os.ReadFile(errFile.Name())
	must(t, err)

	if runErr != nil {
		t.Fatalf("vfox %v (level=%v) returned unexpected error: %v", args, level, runErr)
	}
	return string(outBytes), string(errBytes)
}
