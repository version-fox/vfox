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
	"archive/zip"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

func writeDemoPluginZip(t *testing.T, dir string) string {
	t.Helper()
	zipPath := filepath.Join(dir, "demo.zip")
	f, err := os.Create(zipPath)
	must(t, err)
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("main.lua")
	must(t, err)
	_, err = w.Write([]byte(`PLUGIN = {name="demo", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {} end
`))
	must(t, err)
	must(t, zw.Close())
	return zipPath
}

func TestAddSourceMissingVfoxHome(t *testing.T) {
	for _, tc := range []struct {
		name string
		rel  []string
	}{
		{name: "missing directory", rel: []string{"vfox-home"}},
		{name: "missing parent directories", rel: []string{"no-such-dir", "vfox-home"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newIsolatedCLIFixture(t, false)
			missingHome := filepath.Join(append([]string{f.root}, tc.rel...)...)
			f.env["VFOX_HOME"] = missingHome
			zipPath := writeDemoPluginZip(t, f.work)

			f.runProcess(t, "add", "--source", zipPath).mustSucceed(t, "add --source with missing VFOX_HOME")
			if !exists(filepath.Join(missingHome, "plugin", "demo", "main.lua")) {
				t.Fatalf("expected plugin to be installed under %q", missingHome)
			}
		})
	}

	t.Run("blocked VFOX_HOME fails before download and names VFOX_HOME", func(t *testing.T) {
		f := newIsolatedCLIFixture(t, false)
		blocker := filepath.Join(f.root, "not-a-dir")
		must(t, os.WriteFile(blocker, []byte("file"), 0644))
		f.env["VFOX_HOME"] = filepath.Join(blocker, "vfox-home")

		var hits atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			http.Error(w, "unexpected download", http.StatusInternalServerError)
		}))
		defer srv.Close()

		res := f.runHere(t, "add", "--source", srv.URL+"/demo.zip").mustFail(t, "add --source with blocked VFOX_HOME")
		mustContain(t, "add blocked VFOX_HOME error", res.output(), "VFOX_HOME")
		mustContain(t, "Manager.Add error path", res.output(), "prepare plugin directory")
		if hits.Load() != 0 {
			t.Fatalf("expected add to stop before downloading plugin, got %d requests", hits.Load())
		}
	})
}

func TestReadOnlyAndBlockedVfoxHomeReadPaths(t *testing.T) {
	t.Run("env stdout stays clean shell code when GlobalSdkDir cannot be created", func(t *testing.T) {
		fClean := newIsolatedCLIFixture(t, false)
		cleanEnv := fClean.runProcess(t, "env", "-s", "bash").mustSucceed(t, "baseline env -s bash")

		fBlocked := newIsolatedCLIFixture(t, false)
		must(t, os.WriteFile(filepath.Join(fBlocked.userHome, "sdks"), []byte("file"), 0644))
		blockedEnv := fBlocked.runProcess(t, "env", "-s", "bash").mustSucceed(t, "env -s bash with blocked GlobalSdkDir")

		want := strings.ReplaceAll(cleanEnv.stdout, shellPath(fClean.root), shellPath(fBlocked.root))
		want = strings.ReplaceAll(want, fClean.root, fBlocked.root)
		if blockedEnv.stdout != want {
			t.Fatalf("env stdout changed when GlobalSdkDir could not be created:\ngot:\n%s\nwant:\n%s", blockedEnv.stdout, want)
		}
	})

	t.Run("read-only VFOX_HOME without cache or plugin keeps env and activate identical", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("POSIX chmod 0555 does not block owner writes on Windows or root")
		}
		fBase := newIsolatedCLIFixture(t, false)
		roEmpty := filepath.Join(fBase.root, "ro-vfox-empty")
		must(t, os.MkdirAll(roEmpty, 0755))
		fBase.env["VFOX_HOME"] = roEmpty
		wantEnv := fBase.runProcess(t, "env", "-s", "bash").mustSucceed(t, "writable empty env")
		wantAct := fBase.runProcess(t, "activate", "bash").mustSucceed(t, "writable empty activate")

		fRO := newIsolatedCLIFixture(t, false)
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

	t.Run("read-only VFOX_HOME honors storage.sdkPath override", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("POSIX chmod 0555 does not block owner writes on Windows or root")
		}
		f := newIsolatedCLIFixture(t, true)
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
