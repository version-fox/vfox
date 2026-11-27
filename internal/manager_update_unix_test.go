//go:build !windows

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

package internal

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/version-fox/vfox/internal/config"
)

func deviceOf(t *testing.T, path string) uint64 {
	t.Helper()
	var st syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil {
		t.Fatalf("stat device of %s: %v", path, err)
	}
	return uint64(st.Dev)
}

// TestUpdatePluginDirTakesPluginHomePermissions covers the rename path of the
// move, where the staging directory keeps its mode. When the staging directory
// and the plugin home sit on different filesystems, the move falls back to a
// copy whose top-level directory is created through os.MkdirAll, and the umask
// masks its mode back down: with umask 077 the updated plugin ended at 0700
// while the plugin home was 0755, locking out every other user sharing the
// VFOX_HOME (#833 review). This test stages on /dev/shm, a tmpfs separate from
// the plugin home, so the move really goes through the copy path.
func TestUpdatePluginDirKeepsPluginHomeModeAcrossFilesystems(t *testing.T) {
	const shm = "/dev/shm"
	if info, err := os.Stat(shm); err != nil || !info.IsDir() {
		t.Skip("/dev/shm is not available: cannot stage on a second filesystem")
	}

	f := newManagerFixture(t)
	if deviceOf(t, shm) == deviceOf(t, f.meta.Shared.Plugins) {
		t.Skip("/dev/shm is on the plugin home's filesystem: cannot force the copy path")
	}

	tempRoot := filepath.Join(shm, fmt.Sprintf("vfox-test-%d", os.Getpid()))
	if err := os.MkdirAll(tempRoot, 0755); err != nil {
		t.Fatalf("create staging root on /dev/shm: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempRoot) })
	f.meta.User.Temp = tempRoot

	// The mode is only lost under a restrictive umask; 077 reproduces the
	// original report. Restored before the test ends.
	oldUmask := syscall.Umask(0o077)
	t.Cleanup(func() { syscall.Umask(oldUmask) })

	newPlugin := fmt.Sprintf(updateTestPlugin, "9.9.9", "")
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	entry, err := zw.Create("main.lua")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err = entry.Write([]byte(newPlugin)); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err = zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/demo.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"name":"demo","version":"9.9.9","downloadUrl":"http://%s/files/demo.zip"}`, r.Host)
		case "/files/demo.zip":
			_, _ = w.Write(zipBuf.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	f.RuntimeEnvContext.UserConfig = &config.Config{
		Proxy:    config.EmptyProxy,
		Registry: &config.Registry{Address: srv.URL},
	}

	pluginDir := filepath.Join(f.meta.Shared.Plugins, "demo")
	if err = os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(pluginDir, "main.lua"), []byte(fmt.Sprintf(updateTestPlugin, "0.0.1", "")), 0644); err != nil {
		t.Fatal(err)
	}

	if err = f.Update("demo"); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	homeInfo, err := os.Stat(f.meta.Shared.Plugins)
	if err != nil {
		t.Fatalf("stat plugin home: %v", err)
	}
	info, err := os.Stat(pluginDir)
	if err != nil {
		t.Fatalf("stat updated plugin dir: %v", err)
	}
	if got, want := info.Mode().Perm(), homeInfo.Mode().Perm(); got != want {
		t.Fatalf("updated plugin dir mode = %o across filesystems, want plugin home mode %o", got, want)
	}
}
