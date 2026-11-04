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
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/config"
)

// updateTestPlugin is a minimal SDK plugin whose version and manifestUrl are
// chosen by the test.
const updateTestPlugin = `PLUGIN = {name="demo", version="%s", description="test", author="test"%s}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`

// A plugin installed from a manifestUrl outside the registry is updated from
// that manifest: the registry answers 404, the manifest offers a newer
// version, and Update installs it.
func TestUpdateInstallsManifestUrlUpdate(t *testing.T) {
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
		case "/manifest/demo.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"name":"demo","version":"9.9.9","downloadUrl":"http://%s/files/demo.zip"}`, r.Host)
		case "/files/demo.zip":
			_, _ = w.Write(zipBuf.Bytes())
		default:
			// the registry path answers 404
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	f := newManagerFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{
		Proxy:    config.EmptyProxy,
		Registry: &config.Registry{Address: srv.URL + "/registry"},
	}

	pluginDir := filepath.Join(f.meta.Shared.Plugins, "demo")
	if err = os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	oldPlugin := fmt.Sprintf(updateTestPlugin, "0.0.1", fmt.Sprintf(`, manifestUrl="%s/manifest/demo.json"`, srv.URL))
	if err = os.WriteFile(filepath.Join(pluginDir, "main.lua"), []byte(oldPlugin), 0644); err != nil {
		t.Fatal(err)
	}

	if err = f.Update("demo"); err != nil {
		t.Fatalf("Update() error = %v, want the manifestUrl update installed", err)
	}
	content, err := os.ReadFile(filepath.Join(pluginDir, "main.lua"))
	if err != nil {
		t.Fatalf("read updated plugin: %v", err)
	}
	if !strings.Contains(string(content), `version="9.9.9"`) {
		t.Fatalf("updated plugin = %q, want version 9.9.9 in the plugin directory", content)
	}
}
