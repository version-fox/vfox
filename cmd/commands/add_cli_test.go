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
		name  string
		rel   []string
		alias string
	}{
		{name: "missing directory", rel: []string{"vfox-home"}},
		{name: "missing parent directories", rel: []string{"no-such-dir", "vfox-home"}},
		{name: "missing parent directories with alias", rel: []string{"no-such-dir", "vfox-home"}, alias: "mydemo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCLIFixtureWithoutRegistryGuard(t, false)
			missingHome := filepath.Join(append([]string{f.root}, tc.rel...)...)
			f.env["VFOX_HOME"] = missingHome
			zipPath := writeDemoPluginZip(t, f.work)

			args := []string{"add", "--source", zipPath}
			installedName := "demo"
			if tc.alias != "" {
				args = append(args, "--alias", tc.alias)
				installedName = tc.alias
			}
			f.runProcess(t, args...).mustSucceed(t, "add --source with missing VFOX_HOME")
			if !exists(filepath.Join(missingHome, "plugin", installedName, "main.lua")) {
				t.Fatalf("expected plugin to be installed under %q", missingHome)
			}
		})
	}

	t.Run("remote --source installs into missing nested VFOX_HOME", func(t *testing.T) {
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		missingHome := filepath.Join(f.root, "no-such-dir", "remote-vfox-home")
		f.env["VFOX_HOME"] = missingHome
		zipBytes, err := os.ReadFile(writeDemoPluginZip(t, f.work))
		must(t, err)

		var hits atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipBytes)
		}))
		defer srv.Close()

		f.runProcess(t, "add", "--source", srv.URL+"/demo.zip").mustSucceed(t, "add remote --source with missing VFOX_HOME")
		if hits.Load() != 1 {
			t.Fatalf("expected 1 remote download request, got %d", hits.Load())
		}
		if !exists(filepath.Join(missingHome, "plugin", "demo", "main.lua")) {
			t.Fatalf("expected remote plugin to be installed under %q", missingHome)
		}
	})

	t.Run("blocked VFOX_HOME fails before download and names VFOX_HOME", func(t *testing.T) {
		f := newCLIFixtureWithoutRegistryGuard(t, false)
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

	t.Run("non-writable VFOX_HOME plugin dir fails before download and names VFOX_HOME", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("POSIX chmod 0555 does not block owner writes on Windows or root")
		}
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		pluginsDir := filepath.Join(f.shared, "plugin")
		must(t, os.Chmod(pluginsDir, 0555))
		t.Cleanup(func() { _ = os.Chmod(pluginsDir, 0755) })

		var hits atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			http.Error(w, "unexpected download", http.StatusInternalServerError)
		}))
		defer srv.Close()

		res := f.runHere(t, "add", "--source", srv.URL+"/demo.zip").mustFail(t, "add --source with non-writable VFOX_HOME/plugin")
		mustContain(t, "add non-writable plugin dir error", res.output(), "VFOX_HOME")
		mustContain(t, "Manager.Add error path", res.output(), "prepare plugin directory")
		mustContain(t, "add non-writable plugin dir error", res.output(), "is not writable")
		mustNotContain(t, "add non-writable plugin dir error", res.output(), ".vfox-write-check-")
		if hits.Load() != 0 {
			t.Fatalf("expected add to stop before downloading plugin, got %d requests", hits.Load())
		}
	})

	t.Run("local source fails cleanly when plugin dir is read-only and plugin is not installed", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("POSIX chmod 0555 does not block owner writes on Windows or root")
		}
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		zipPath := writeDemoPluginZip(t, f.work)
		pluginsDir := filepath.Join(f.shared, "plugin")
		must(t, os.Chmod(pluginsDir, 0555))
		t.Cleanup(func() { _ = os.Chmod(pluginsDir, 0755) })

		res := f.runHere(t, "add", "--source", zipPath).mustFail(t, "add --source local zip with non-writable plugin/")
		mustContain(t, "local source unwritable error", res.output(), "prepare plugin directory")
		mustContain(t, "local source unwritable error", res.output(), "is not writable")
		mustNotContain(t, "local source unwritable error", res.output(), ".vfox-write-check-")
		if exists(filepath.Join(pluginsDir, "demo")) {
			t.Fatal("expected demo plugin not to be installed when plugin dir is read-only")
		}
	})

	t.Run("installed plugin reports already exists when plugin dir is read-only", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("POSIX chmod 0555 does not block owner writes on Windows or root")
		}
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		zipPath := writeDemoPluginZip(t, f.work)
		f.runProcess(t, "add", "--source", zipPath).mustSucceed(t, "seed demo plugin")

		pluginsDir := filepath.Join(f.shared, "plugin")
		must(t, os.Chmod(pluginsDir, 0555))
		t.Cleanup(func() { _ = os.Chmod(pluginsDir, 0755) })

		byName := f.runHere(t, "add", "demo").mustFail(t, "add demo when already installed")
		mustContain(t, "add existing plugin error", byName.output(), "plugin named demo already exists")
		mustNotContain(t, "add existing plugin error", byName.output(), "prepare plugin directory")

		bySource := f.runHere(t, "add", "--source", zipPath).mustFail(t, "add --source demo.zip when already installed")
		mustContain(t, "add --source existing plugin error", bySource.output(), "plugin named demo already exists")
		mustNotContain(t, "add --source existing plugin error", bySource.output(), "prepare plugin directory")
	})
}
