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
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/sdk"
)

// Note: In these tests, Manager instances are created without explicitly
// initializing the mu field. This is intentional and valid in Go, as the
// zero-value of sync.RWMutex is ready to use.

// TestLookupSdk_ConcurrentAccess tests that concurrent calls to LookupSdk
// do not cause race conditions or panics
func TestLookupSdk_ConcurrentAccess(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	userHome := filepath.Join(tmpDir, "user_home")
	vfoxHome := filepath.Join(tmpDir, "vfox_home")
	currentDir := tmpDir

	// Create necessary directories
	os.MkdirAll(userHome, 0755)
	os.MkdirAll(vfoxHome, 0755)

	// Create PathMeta
	meta := pathmeta.NewPathMeta(userHome, vfoxHome, currentDir, 12345)

	// Create plugins directory
	pluginsDir := meta.Shared.Plugins
	os.MkdirAll(pluginsDir, 0755)

	// Create Manager
	manager := &Manager{
		RuntimeEnvContext: &env.RuntimeEnvContext{
			UserConfig:        &config.Config{},
			CurrentWorkingDir: currentDir,
			PathMeta:          meta,
			RuntimeVersion:    "test",
		},
		openSdks: make(map[string]sdk.Sdk),
	}

	// Test concurrent access with non-existent SDK
	// This should not cause race conditions or panics
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			// Try to lookup a non-existent SDK
			_, err := manager.LookupSdk("nonexistent")
			// We expect an error since the SDK doesn't exist
			if err == nil {
				t.Errorf("Expected error for non-existent SDK, got nil")
			}
		}(i)
	}

	wg.Wait()
}

// TestLookupSdk_ConcurrentCacheHits tests that concurrent cache hits
// do not cause race conditions
func TestLookupSdk_ConcurrentCacheHits(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	userHome := filepath.Join(tmpDir, "user_home")
	vfoxHome := filepath.Join(tmpDir, "vfox_home")
	currentDir := tmpDir

	// Create necessary directories
	os.MkdirAll(userHome, 0755)
	os.MkdirAll(vfoxHome, 0755)

	// Create PathMeta
	meta := pathmeta.NewPathMeta(userHome, vfoxHome, currentDir, 12345)

	// Create Manager
	manager := &Manager{
		RuntimeEnvContext: &env.RuntimeEnvContext{
			UserConfig:        &config.Config{},
			CurrentWorkingDir: currentDir,
			PathMeta:          meta,
			RuntimeVersion:    "test",
		},
		openSdks: make(map[string]sdk.Sdk),
	}

	// Pre-populate the cache with a nil SDK (for testing purposes)
	// In real usage, this would be a valid SDK object
	// Use proper locking to avoid race conditions in the test itself
	manager.mu.Lock()
	manager.openSdks["test"] = nil
	manager.mu.Unlock()

	// Test concurrent cache reads
	const numGoroutines = 20
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			// Try to lookup the cached SDK
			_, err := manager.LookupSdk("test")
			if err != nil {
				// Error is expected since we stored nil
				// But we should not panic or race
			}
		}(i)
	}

	wg.Wait()
}

// TestLoadAllSdk_NoRaceCondition verifies that LoadAllSdk
// can safely populate the cache
func TestLoadAllSdk_NoRaceCondition(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	userHome := filepath.Join(tmpDir, "user_home")
	vfoxHome := filepath.Join(tmpDir, "vfox_home")
	currentDir := tmpDir

	// Create necessary directories
	os.MkdirAll(userHome, 0755)
	os.MkdirAll(vfoxHome, 0755)

	// Create PathMeta
	meta := pathmeta.NewPathMeta(userHome, vfoxHome, currentDir, 12345)

	// Create plugins directory
	pluginsDir := meta.Shared.Plugins
	os.MkdirAll(pluginsDir, 0755)

	// Create Manager
	manager := &Manager{
		RuntimeEnvContext: &env.RuntimeEnvContext{
			UserConfig:        &config.Config{},
			CurrentWorkingDir: currentDir,
			PathMeta:          meta,
			RuntimeVersion:    "test",
		},
		openSdks: make(map[string]sdk.Sdk),
	}

	// Call LoadAllSdk (will return empty list but shouldn't panic)
	sdks, err := manager.LoadAllSdk()
	if err != nil {
		t.Fatalf("LoadAllSdk failed: %v", err)
	}

	// Should return empty list for empty plugins directory
	if len(sdks) != 0 {
		t.Errorf("Expected 0 SDKs, got %d", len(sdks))
	}
}

// TestClose_NoRaceCondition verifies that Close
// can safely iterate over the cache
func TestClose_NoRaceCondition(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	userHome := filepath.Join(tmpDir, "user_home")
	vfoxHome := filepath.Join(tmpDir, "vfox_home")
	currentDir := tmpDir

	// Create necessary directories
	os.MkdirAll(userHome, 0755)
	os.MkdirAll(vfoxHome, 0755)

	// Create PathMeta
	meta := pathmeta.NewPathMeta(userHome, vfoxHome, currentDir, 12345)

	// Create Manager
	manager := &Manager{
		RuntimeEnvContext: &env.RuntimeEnvContext{
			UserConfig:        &config.Config{},
			CurrentWorkingDir: currentDir,
			PathMeta:          meta,
			RuntimeVersion:    "test",
		},
		openSdks: make(map[string]sdk.Sdk),
	}

	// Call Close (should not panic even with empty cache)
	manager.Close()
}

// TestLookupSdk_SimulateActivateScenario simulates the exact scenario from activate.go
// where multiple goroutines lookup different SDKs concurrently
func TestLookupSdk_SimulateActivateScenario(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	userHome := filepath.Join(tmpDir, "user_home")
	vfoxHome := filepath.Join(tmpDir, "vfox_home")
	currentDir := tmpDir

	// Create necessary directories
	os.MkdirAll(userHome, 0755)
	os.MkdirAll(vfoxHome, 0755)

	// Create PathMeta
	meta := pathmeta.NewPathMeta(userHome, vfoxHome, currentDir, 12345)

	// Create plugins directory
	pluginsDir := meta.Shared.Plugins
	os.MkdirAll(pluginsDir, 0755)

	// Create Manager
	manager := &Manager{
		RuntimeEnvContext: &env.RuntimeEnvContext{
			UserConfig:        &config.Config{},
			CurrentWorkingDir: currentDir,
			PathMeta:          meta,
			RuntimeVersion:    "test",
		},
		openSdks: make(map[string]sdk.Sdk),
	}

	// Simulate the activate.go scenario with multiple SDKs being looked up concurrently
	// This is what was causing the concurrent map writes error
	sdkNames := []string{"nodejs", "python", "java", "go", "ruby", "rust"}

	var wg sync.WaitGroup
	wg.Add(len(sdkNames))

	for _, sdkName := range sdkNames {
		sdkName := sdkName // Capture loop variable
		go func() {
			defer wg.Done()
			// Each goroutine tries to lookup a different SDK
			// This simulates the errgroup in activate.go
			_, err := manager.LookupSdk(sdkName)
			// We expect errors since these SDKs don't exist
			// But we should not panic or race
			if err == nil {
				t.Errorf("Expected error for non-existent SDK %s, got nil", sdkName)
			}
		}()
	}

	wg.Wait()
}

func TestDownloadPluginRejectsUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("<html>refused</html>"))
	}))
	defer srv.Close()

	userTemp := t.TempDir()
	pluginsDir := t.TempDir()
	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta: &pathmeta.PathMeta{
			User:   pathmeta.UserPaths{Temp: userTemp},
			Shared: pathmeta.SharedPaths{Plugins: pluginsDir},
		},
	}}

	if _, err := m.downloadPlugin(srv.URL + "/plugin.zip"); err == nil || !strings.Contains(err.Error(), "unexpected HTTP status 403 Forbidden") {
		t.Fatalf("downloadPlugin() error = %v, want unexpected HTTP status 403 Forbidden", err)
	}

	for name, dir := range map[string]string{"plugins": pluginsDir, "temp": userTemp} {
		entries, readErr := os.ReadDir(dir)
		if readErr != nil && !os.IsNotExist(readErr) {
			t.Fatalf("read %s dir: %v", name, readErr)
		}
		if len(entries) != 0 {
			t.Fatalf("downloadPlugin() left %d file(s) in %s dir for an HTTP 403 response, want none", len(entries), name)
		}
	}
}

func TestDownloadPluginKeepsNotFoundMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta: &pathmeta.PathMeta{
			User:   pathmeta.UserPaths{Temp: t.TempDir()},
			Shared: pathmeta.SharedPaths{Plugins: t.TempDir()},
		},
	}}

	if _, err := m.downloadPlugin(srv.URL + "/plugin.zip"); err == nil || !strings.Contains(err.Error(), "plugin not found") {
		t.Fatalf("downloadPlugin() error = %v, want plugin not found", err)
	}
}

func TestDownloadPluginNotFoundRedactsPassword(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta: &pathmeta.PathMeta{
			User:   pathmeta.UserPaths{Temp: t.TempDir()},
			Shared: pathmeta.SharedPaths{Plugins: t.TempDir()},
		},
	}}

	u, err := url.Parse(srv.URL + "/plugin.zip")
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	u.User = url.UserPassword("user", "s3cret")

	_, err = m.downloadPlugin(u.String())
	if err == nil || !strings.Contains(err.Error(), "plugin not found") {
		t.Fatalf("downloadPlugin() error = %v, want plugin not found", err)
	}
	if strings.Contains(err.Error(), "s3cret") {
		t.Fatalf("downloadPlugin() error = %v, want the URL password redacted", err)
	}
	if !strings.Contains(err.Error(), "user:xxxxx@") {
		t.Fatalf("downloadPlugin() error = %v, want the redacted userinfo in the URL", err)
	}
}

func TestFetchPluginManifestRedactsConnectionError(t *testing.T) {
	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
	}}

	_, err := m.fetchPluginManifest("http://user:s3cret@127.0.0.1:1/manifest.json?token=abc123")
	if err == nil {
		t.Fatal("fetchPluginManifest() error = nil, want a connection error")
	}
	msg := err.Error()
	if strings.Contains(msg, "s3cret") || strings.Contains(msg, "abc123") {
		t.Fatalf("fetchPluginManifest() error = %q, want the password and the query token redacted", msg)
	}
	if !strings.Contains(msg, "user:xxxxx@") || !strings.Contains(msg, "token=xxxxx") {
		t.Fatalf("fetchPluginManifest() error = %q, want the redacted userinfo and query", msg)
	}
}

func TestDownloadPluginRedactsRequestError(t *testing.T) {
	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta: &pathmeta.PathMeta{
			User:   pathmeta.UserPaths{Temp: t.TempDir()},
			Shared: pathmeta.SharedPaths{Plugins: t.TempDir()},
		},
	}}

	_, err := m.downloadPlugin("http://user:Zq9%ss@127.0.0.1/plugin.zip")
	if err == nil {
		t.Fatal("downloadPlugin() error = nil, want a parse error")
	}
	if msg := err.Error(); strings.Contains(msg, "Zq9") || strings.Contains(msg, "%ss") {
		t.Fatalf("downloadPlugin() error = %q, want the rejected URL credentials hidden", msg)
	}
}

func TestDownloadPluginRedactsConnectionError(t *testing.T) {
	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta: &pathmeta.PathMeta{
			User:   pathmeta.UserPaths{Temp: t.TempDir()},
			Shared: pathmeta.SharedPaths{Plugins: t.TempDir()},
		},
	}}

	_, err := m.downloadPlugin("http://user:s3cret@127.0.0.1:1/plugin.zip?token=abc123")
	if err == nil {
		t.Fatal("downloadPlugin() error = nil, want a connection error")
	}
	msg := err.Error()
	if strings.Contains(msg, "s3cret") || strings.Contains(msg, "abc123") {
		t.Fatalf("downloadPlugin() error = %q, want the password and the query token redacted", msg)
	}
}

func TestAvailableRedactsConnectionError(t *testing.T) {
	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{
			Proxy:    config.EmptyProxy,
			Registry: &config.Registry{Address: "http://user:s3cret@127.0.0.1:1/registry?token=abc123"},
		},
	}}

	_, err := m.Available()
	if err == nil {
		t.Fatal("Available() error = nil, want a connection error")
	}
	msg := err.Error()
	if strings.Contains(msg, "s3cret") || strings.Contains(msg, "abc123") {
		t.Fatalf("Available() error = %q, want the registry credentials redacted", msg)
	}
}

func TestInstallPluginToTempRedactsUnsupportedExtension(t *testing.T) {
	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
	}}

	_, err := m.installPluginToTemp("http://user:s3cret@example.com/plugin.txt?token=abc123")
	if err == nil {
		t.Fatal("installPluginToTemp() error = nil, want an unsupported type error")
	}
	msg := err.Error()
	if strings.Contains(msg, "s3cret") || strings.Contains(msg, "abc123") {
		t.Fatalf("installPluginToTemp() error = %q, want the password and the query token redacted", msg)
	}
	if !strings.Contains(msg, "token=xxxxx") {
		t.Fatalf("installPluginToTemp() error = %q, want the masked query in the extension", msg)
	}
}

func TestDownloadPluginNamesFileFromURLPath(t *testing.T) {
	const body = "plugin bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta: &pathmeta.PathMeta{
			User:   pathmeta.UserPaths{Temp: t.TempDir()},
			Shared: pathmeta.SharedPaths{Plugins: t.TempDir()},
		},
	}}

	// The saved name has to end with the extension the whole URL was accepted
	// by: the base of the URL path does not, so the name falls back to
	// "plugin" plus the extension.
	path, err := m.downloadPlugin(srv.URL + "/dl?token=abc123&file=plugin.zip")
	if err != nil {
		t.Fatalf("downloadPlugin() error = %v", err)
	}
	if base := filepath.Base(path); base != "plugin.zip" {
		t.Fatalf("downloadPlugin() file name = %q, want plugin.zip", base)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != body {
		t.Fatalf("downloaded content = %q, err = %v", got, err)
	}

	// A URL whose path has no file name falls back to "plugin" plus the extension.
	path, err = m.downloadPlugin(srv.URL + "/?file=plugin.zip")
	if err != nil {
		t.Fatalf("downloadPlugin() error = %v", err)
	}
	if base := filepath.Base(path); base != "plugin.zip" {
		t.Fatalf("downloadPlugin() file name = %q, want plugin.zip", base)
	}
}

func TestDownloadPluginKeepsBackslashPathsInDownloadTempDir(t *testing.T) {
	const body = "plugin bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	userTemp := t.TempDir()
	plugins := t.TempDir()
	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta: &pathmeta.PathMeta{
			User:   pathmeta.UserPaths{Temp: userTemp},
			Shared: pathmeta.SharedPaths{Plugins: plugins},
		},
	}}

	// A backslash in the URL path, literal or as %5C, must resolve to the
	// trailing path segment so the download stays inside the private directory
	// with a plain filename across Windows and POSIX.
	for _, p := range []string{"/..\\evil.zip", "/..%5Cevil.zip", "/sub/..%5C..%5Cevil.zip"} {
		saved, err := m.downloadPlugin(srv.URL + p)
		if err != nil {
			t.Fatalf("downloadPlugin(%q) error = %v", p, err)
		}
		if filepath.Dir(filepath.Dir(saved)) != userTemp || filepath.Base(saved) != "evil.zip" {
			t.Fatalf("downloadPlugin() saved %q, want a file inside a direct subdir of %q", saved, userTemp)
		}
		_ = os.RemoveAll(filepath.Dir(saved))
	}
}

func TestDownloadPluginSanitizesInvalidFilenameCharacters(t *testing.T) {
	const body = "plugin bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	userTemp := t.TempDir()
	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta: &pathmeta.PathMeta{
			User:   pathmeta.UserPaths{Temp: userTemp},
			Shared: pathmeta.SharedPaths{Plugins: t.TempDir()},
		},
	}}

	for _, p := range []string{
		"/a%3Fb.zip",
		"/a%22b.zip",
		"/a%3Ab.zip",
		"/a%3Cb.zip",
		"/a%3Eb.zip",
		"/a%7Cb.zip",
		"/a%2Ab.zip",
		"/a%0Ab.zip",
		"/a%7Fb.zip",
	} {
		saved, err := m.downloadPlugin(srv.URL + p)
		if err != nil {
			t.Fatalf("downloadPlugin(%q) error = %v", p, err)
		}
		if filepath.Dir(filepath.Dir(saved)) != userTemp || filepath.Base(saved) != "plugin.zip" {
			t.Fatalf("downloadPlugin(%q) saved %q, want plugin.zip inside %q", p, saved, userTemp)
		}
		_ = os.RemoveAll(filepath.Dir(saved))
	}
}

func TestAddFromMirroredManifest(t *testing.T) {
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
		if r.URL.Path != "/plugins/demo.lua" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, updateTestPlugin, "1.0.0", "")
	}))
	defer mirror.Close()

	thirdParty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		thirdPartyPaths = append(thirdPartyPaths, r.URL.Path)
		mu.Unlock()
		switch r.URL.Path {
		case "/plugins/other.lua":
			fmt.Fprintf(w, fixturePlugin, "other", "")
		case "/plugins/explicit.lua":
			fmt.Fprintf(w, fixturePlugin, "explicit", "")
		default:
			http.NotFound(w, r)
		}
	}))
	defer thirdParty.Close()

	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		registryPaths = append(registryPaths, r.URL.Path)
		mu.Unlock()
		switch r.URL.Path {
		case "/demo.json":
			fmt.Fprint(w, `{"name":"demo","downloadUrl":"https://github.com/plugins/demo.lua"}`)
		case "/other.json":
			fmt.Fprintf(w, `{"name":"other","downloadUrl":%q}`, thirdParty.URL+"/plugins/other.lua")
		default:
			http.NotFound(w, r)
		}
	}))
	defer registry.Close()

	f := newManagerFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{
		Proxy:    config.EmptyProxy,
		Registry: &config.Registry{Address: registry.URL},
		GitHub:   &config.GitHub{BaseURL: mirror.URL},
	}

	if err := f.Add("demo", "", ""); err != nil {
		t.Fatalf("Add(demo) error = %v", err)
	}
	if err := f.Add("other", "", ""); err != nil {
		t.Fatalf("Add(other) error = %v", err)
	}
	if err := f.Add("explicit", thirdParty.URL+"/plugins/explicit.lua", ""); err != nil {
		t.Fatalf("Add(explicit) error = %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if got := strings.Join(registryPaths, ","); got != "/demo.json,/other.json" {
		t.Fatalf("registry requests = %q, want /demo.json,/other.json", got)
	}
	if got := strings.Join(mirrorPaths, ","); got != "/plugins/demo.lua" {
		t.Fatalf("mirror requests = %q, want /plugins/demo.lua", got)
	}
	if got := strings.Join(thirdPartyPaths, ","); got != "/plugins/other.lua,/plugins/explicit.lua" {
		t.Fatalf("third-party requests = %q, want /plugins/other.lua,/plugins/explicit.lua", got)
	}
}

func TestEnsureWritableDir(t *testing.T) {
	t.Run("creates missing parents and removes probe file", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "missing", "plugin")
		if err := ensureWritableDir(target); err != nil {
			t.Fatalf("ensureWritableDir(%q) error = %v", target, err)
		}
		entries, err := os.ReadDir(target)
		if err != nil {
			t.Fatalf("ReadDir(%q) error = %v", target, err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected %q to be empty after probe cleanup, got %d entries", target, len(entries))
		}
	})

	t.Run("fails when a regular file blocks directory creation", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "not-a-dir")
		if err := os.WriteFile(blocker, []byte("file"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := ensureWritableDir(filepath.Join(blocker, "plugin")); err == nil {
			t.Fatal("expected error when parent is a regular file")
		}
	})

	t.Run("fails when existing directory is read-only", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("POSIX chmod 0555 does not block owner writes on Windows or root")
		}
		dir := filepath.Join(t.TempDir(), "plugin")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(dir, 0555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0755) })

		if err := ensureWritableDir(dir); err == nil {
			t.Fatal("expected error when directory is 0555")
		}
	})
}

func TestPreparePluginDirErrorWithoutVfoxHome(t *testing.T) {
	t.Setenv(env.HomeFromEnv, "")
	pluginsDir := filepath.Join(t.TempDir(), "plugin")
	probeErr := &fs.PathError{
		Op:   "open",
		Path: filepath.Join(pluginsDir, ".vfox-write-check-123"),
		Err:  fs.ErrPermission,
	}

	err := preparePluginDirError(pluginsDir, probeErr)
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("expected errors.Is(err, fs.ErrPermission), got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "prepare plugin directory "+pluginsDir+" is not writable") {
		t.Fatalf("unexpected error format: %q", msg)
	}
	if strings.Contains(msg, env.HomeFromEnv) {
		t.Fatalf("expected %s to be omitted when unset, got %q", env.HomeFromEnv, msg)
	}
	if strings.Contains(msg, ".vfox-write-check-") {
		t.Fatalf("expected probe filename to be unwrapped, got %q", msg)
	}

	blocker := filepath.Join(t.TempDir(), "file-blocker")
	dirErr := preparePluginDirError(filepath.Join(blocker, "plugin"), &fs.PathError{
		Op:   "mkdir",
		Path: filepath.Join(blocker, "plugin"),
		Err:  errors.New("not a directory"),
	})
	if strings.Contains(dirErr.Error(), "is not writable") || !strings.Contains(dirErr.Error(), "failed: not a directory") {
		t.Fatalf("unexpected non-permission error wording: %q", dirErr.Error())
	}
}

func TestDownloadPluginSavesToPrivateTemp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("small-payload"))
	}))
	defer srv.Close()

	root := t.TempDir()
	userTemp := filepath.Join(root, "tmp")
	pluginsDir := filepath.Join(root, "plugin")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		t.Fatal(err)
	}
	leftover := filepath.Join(pluginsDir, "demo.zip")
	if err := os.WriteFile(leftover, []byte(strings.Repeat("LATE", 1024)), 0644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta: &pathmeta.PathMeta{
			User:   pathmeta.UserPaths{Temp: userTemp},
			Shared: pathmeta.SharedPaths{Plugins: pluginsDir},
		},
	}}

	gotPath, err := m.downloadPlugin(srv.URL + "/demo.zip")
	if err != nil {
		t.Fatalf("downloadPlugin() error = %v", err)
	}
	defer os.RemoveAll(filepath.Dir(gotPath))

	if !strings.HasPrefix(gotPath, userTemp+string(filepath.Separator)) {
		t.Fatalf("downloaded archive %q should live under User.Temp %q", gotPath, userTemp)
	}
	content, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "small-payload" {
		t.Fatalf("downloaded content = %q, want small-payload", string(content))
	}
	// Shared plugins/ must not be written by downloadPlugin and leftover in plugin/ stays untouched.
	leftoverBytes, err := os.ReadFile(leftover)
	if err != nil || len(leftoverBytes) != 4096 {
		t.Fatalf("expected leftover file in pluginsDir to stay untouched (err=%v, len=%d)", err, len(leftoverBytes))
	}
}

func TestPluginFileNameRejectsWindowsReservedNames(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{url: "https://example.com/nul.zip", want: "plugin.zip"},
		{url: "https://example.com/CON.zip", want: "plugin.zip"},
		{url: "https://example.com/prn.zip", want: "plugin.zip"},
		{url: "https://example.com/aux.lua", want: "plugin.lua"},
		{url: "https://example.com/com0.zip", want: "plugin.zip"},
		{url: "https://example.com/com1.zip", want: "plugin.zip"},
		{url: "https://example.com/COM9.tar.xz", want: "plugin.xz"},
		{url: "https://example.com/lpt0.zip", want: "plugin.zip"},
		{url: "https://example.com/lpt9.tar.gz", want: "plugin.gz"},
		{url: "https://example.com/nul", want: "plugin"},
		{url: "https://example.com/console.zip", want: "console.zip"},
		{url: "https://example.com/my-nul.zip", want: "my-nul.zip"},
		{url: "https://example.com/demo.zip", want: "demo.zip"},
	}
	for _, tc := range tests {
		if got := pluginFileName(tc.url); got != tc.want {
			t.Errorf("pluginFileName(%q) = %q, want %q", tc.url, got, tc.want)
		}
	}
}

func TestCleanTmpSweepsStaleDownloadDirs(t *testing.T) {
	userTemp := t.TempDir()
	staleDir := filepath.Join(userTemp, "vfox-dl-stale")
	freshDir := filepath.Join(userTemp, "vfox-dl-fresh")
	for _, dir := range []string{staleDir, freshDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "partial.zip"), []byte("partial"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	staleTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(staleDir, staleTime, staleTime); err != nil {
		t.Fatal(err)
	}

	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		PathMeta: &pathmeta.PathMeta{
			User: pathmeta.UserPaths{Temp: userTemp},
		},
	}}
	m.CleanTmp()

	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Fatalf("expected stale download dir %q to be removed, stat err = %v", staleDir, err)
	}
	if info, err := os.Stat(freshDir); err != nil || !info.IsDir() {
		t.Fatalf("expected fresh download dir %q to be kept (info=%v, err=%v)", freshDir, info, err)
	}
}
