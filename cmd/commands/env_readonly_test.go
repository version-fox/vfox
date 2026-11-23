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
	"sort"
	"strings"
	"testing"
)

func sortSemicolonStatements(s string) []string {
	raw := strings.Split(s, ";")
	parts := make([]string, 0, len(raw))
	for _, part := range raw {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	sort.Strings(parts)
	return parts
}

func TestReadOnlyAndBlockedVfoxHomeReadPaths(t *testing.T) {
	t.Run("activate comparison helper normalizes map-ordered exports", func(t *testing.T) {
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		first := f.runProcess(t, "activate", "bash").mustSucceed(t, "first activate bash")
		second := f.runProcess(t, "activate", "bash").mustSucceed(t, "second activate bash")
		if !slices.Equal(sortSemicolonStatements(first.stdout), sortSemicolonStatements(second.stdout)) {
			t.Fatalf("normalized activate output differed across runs:\nfirst:\n%s\nsecond:\n%s", first.stdout, second.stdout)
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

	t.Run("env stdout stays clean shell code when session temp dir cannot be created", func(t *testing.T) {
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
		if expectedAct := norm(wantAct.stdout); !slices.Equal(sortSemicolonStatements(gotAct.stdout), sortSemicolonStatements(expectedAct)) {
			t.Fatalf("activate output mismatch on read-only VFOX_HOME:\ngot:\n%s\nwant:\n%s", gotAct.stdout, expectedAct)
		}
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
