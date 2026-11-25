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
	"sync"
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
	entry, err := zw.CreateHeader(&zip.FileHeader{Name: "main.lua", Method: zip.Store})
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err = entry.Write([]byte(newPlugin)); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err = zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/demo.zip" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(zipBuf.Bytes())
	}))
	defer mirror.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest/demo.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"name":"demo","version":"9.9.9","downloadUrl":"https://github.com/files/demo.zip"}`)
		default:
			// the registry path answers 404
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	f := newManagerFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{
		GitHub:   &config.GitHub{BaseURL: mirror.URL},
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

	// Damage a stored entry, retaining the original ZIP checksum.
	damaged := bytes.Replace(zipBuf.Bytes(), []byte(newPlugin), []byte(strings.ToUpper(newPlugin)), 1)
	corrupt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(damaged) }))
	defer corrupt.Close()
	if err := f.Add("", corrupt.URL+"/explicit.zip", ""); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("explicit archive error = %v, want checksum mismatch", err)
	}
}

func TestUpdateFromRegistryWithGitHubMirrorAndThirdPartyURL(t *testing.T) {
	makeZip := func(t *testing.T, luaSrc string) []byte {
		t.Helper()
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		entry, err := zw.Create("main.lua")
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err = entry.Write([]byte(luaSrc)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
		if err = zw.Close(); err != nil {
			t.Fatalf("close zip: %v", err)
		}
		return buf.Bytes()
	}

	demoZip := makeZip(t, fmt.Sprintf(updateTestPlugin, "2.0.0", ""))
	otherPlugin := strings.Replace(fmt.Sprintf(updateTestPlugin, "2.0.0", ""), `name="demo"`, `name="other"`, 1)
	otherZip := makeZip(t, otherPlugin)

	var (
		mu              sync.Mutex
		mirrorPaths     []string
		thirdPartyPaths []string
		registryPaths   []string
	)
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		mirrorPaths = append(mirrorPaths, r.URL.Path)
		mu.Unlock()
		if r.URL.Path != "/files/demo.zip" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(demoZip)
	}))
	defer mirror.Close()

	thirdParty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		thirdPartyPaths = append(thirdPartyPaths, r.URL.Path)
		mu.Unlock()
		if r.URL.Path != "/files/other.zip" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(otherZip)
	}))
	defer thirdParty.Close()

	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		registryPaths = append(registryPaths, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/demo.json":
			fmt.Fprint(w, `{"name":"demo","version":"2.0.0","downloadUrl":"https://github.com/files/demo.zip"}`)
		case "/other.json":
			fmt.Fprintf(w, `{"name":"other","version":"2.0.0","downloadUrl":%q}`, thirdParty.URL+"/files/other.zip")
		default:
			http.NotFound(w, r)
		}
	}))
	defer registry.Close()

	f := newManagerFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{
		GitHub:   &config.GitHub{BaseURL: mirror.URL},
		Proxy:    config.EmptyProxy,
		Registry: &config.Registry{Address: registry.URL},
	}

	for _, name := range []string{"demo", "other"} {
		pluginDir := filepath.Join(f.meta.Shared.Plugins, name)
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatal(err)
		}
		src := strings.Replace(fmt.Sprintf(updateTestPlugin, "0.0.1", ""), `name="demo"`, fmt.Sprintf(`name=%q`, name), 1)
		if err := os.WriteFile(filepath.Join(pluginDir, "main.lua"), []byte(src), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := f.Update("demo"); err != nil {
		t.Fatalf("Update(demo) error = %v", err)
	}
	if err := f.Update("other"); err != nil {
		t.Fatalf("Update(other) error = %v", err)
	}

	for _, name := range []string{"demo", "other"} {
		content, err := os.ReadFile(filepath.Join(f.meta.Shared.Plugins, name, "main.lua"))
		if err != nil {
			t.Fatalf("read updated %s plugin: %v", name, err)
		}
		if !strings.Contains(string(content), `version="2.0.0"`) {
			t.Fatalf("updated %s plugin = %q, want version 2.0.0", name, content)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if got := strings.Join(registryPaths, ","); got != "/demo.json,/other.json" {
		t.Fatalf("registry requests = %q, want /demo.json,/other.json", got)
	}
	if got := strings.Join(mirrorPaths, ","); got != "/files/demo.zip" {
		t.Fatalf("mirror requests = %q, want /files/demo.zip", got)
	}
	if got := strings.Join(thirdPartyPaths, ","); got != "/files/other.zip" {
		t.Fatalf("third-party requests = %q, want /files/other.zip", got)
	}
}

// The registry answers 404 and the plugin declares no manifestUrl: Update
// reports that the plugin does not support update.
func TestUpdatePluginWithoutManifestUrlNotSupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	f := newManagerFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{
		Proxy:    config.EmptyProxy,
		Registry: &config.Registry{Address: srv.URL + "/registry"},
	}

	pluginDir := filepath.Join(f.meta.Shared.Plugins, "demo")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "main.lua"), []byte(fmt.Sprintf(updateTestPlugin, "0.0.1", "")), 0644); err != nil {
		t.Fatal(err)
	}

	err := f.Update("demo")
	if err == nil || !strings.Contains(err.Error(), "demo plugin not support update") {
		t.Fatalf("Update() error = %v, want demo plugin not support update", err)
	}
}

// The registry answers 404 and the manifestUrl answers 404 too: Update returns
// the "manifest not found" error, as it did before the manifestUrl fallback.
func TestUpdateManifestUrlAlsoNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	f := newManagerFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{
		Proxy:    config.EmptyProxy,
		Registry: &config.Registry{Address: srv.URL + "/registry"},
	}

	pluginDir := filepath.Join(f.meta.Shared.Plugins, "demo")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	oldPlugin := fmt.Sprintf(updateTestPlugin, "0.0.1", fmt.Sprintf(`, manifestUrl="%s/manifest/demo.json"`, srv.URL))
	if err := os.WriteFile(filepath.Join(pluginDir, "main.lua"), []byte(oldPlugin), 0644); err != nil {
		t.Fatal(err)
	}

	err := f.Update("demo")
	if err == nil || !strings.Contains(err.Error(), "manifest not found") {
		t.Fatalf("Update() error = %v, want manifest not found", err)
	}
}

// The registry manifest offers the installed version: Update reports the
// plugin is already the latest and leaves it alone.
func TestUpdateAlreadyLatestFromRegistry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"name":"demo","version":"0.0.1","downloadUrl":"http://%s/files/demo.zip"}`, r.Host)
	}))
	defer srv.Close()

	f := newManagerFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{
		Proxy:    config.EmptyProxy,
		Registry: &config.Registry{Address: srv.URL},
	}

	pluginDir := filepath.Join(f.meta.Shared.Plugins, "demo")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "main.lua"), []byte(fmt.Sprintf(updateTestPlugin, "0.0.1", "")), 0644); err != nil {
		t.Fatal(err)
	}

	if err := f.Update("demo"); err != nil {
		t.Fatalf("Update() error = %v, want the plugin reported as up to date", err)
	}
	content, err := os.ReadFile(filepath.Join(pluginDir, "main.lua"))
	if err != nil {
		t.Fatalf("read plugin: %v", err)
	}
	if !strings.Contains(string(content), `version="0.0.1"`) {
		t.Fatalf("plugin = %q, want it unchanged", content)
	}
}

// installPluginToTemp accepts a URL by the extension at the end of the whole
// URL, so the saved file name must carry that extension even when the URL path
// does not: NewDecompressor picks the unpacker from the file name and a name
// without the extension panics on a nil decompressor.
func TestInstallPluginToTempNamesFileWithAcceptedExtension(t *testing.T) {
	pluginSrc := fmt.Sprintf(updateTestPlugin, "1.0.0", "")
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	entry, err := zw.Create("main.lua")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err = entry.Write([]byte(pluginSrc)); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err = zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(zipBuf.Bytes())
	}))
	defer srv.Close()

	f := newManagerFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{Proxy: config.EmptyProxy}

	for _, p := range []string{
		"/plugin.zip",
		"/dl/?file=plugin.zip",
		"/dl.php?file=plugin.zip",
		"/download.php?id=7&name=plugin.zip",
		"/dl?token=abc123&file=plugin.zip",
	} {
		t.Run(p, func(t *testing.T) {
			wrapper, err := f.installPluginToTemp(srv.URL + p)
			if err != nil {
				t.Fatalf("installPluginToTemp() error = %v", err)
			}
			wrapper.Close()
		})
	}
}
