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
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

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
	meta, err := pathmeta.NewPathMeta(userHome, vfoxHome, currentDir, 12345)
	if err != nil {
		t.Fatalf("Failed to create PathMeta: %v", err)
	}

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
	meta, err := pathmeta.NewPathMeta(userHome, vfoxHome, currentDir, 12345)
	if err != nil {
		t.Fatalf("Failed to create PathMeta: %v", err)
	}

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
	meta, err := pathmeta.NewPathMeta(userHome, vfoxHome, currentDir, 12345)
	if err != nil {
		t.Fatalf("Failed to create PathMeta: %v", err)
	}

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
	meta, err := pathmeta.NewPathMeta(userHome, vfoxHome, currentDir, 12345)
	if err != nil {
		t.Fatalf("Failed to create PathMeta: %v", err)
	}

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
	meta, err := pathmeta.NewPathMeta(userHome, vfoxHome, currentDir, 12345)
	if err != nil {
		t.Fatalf("Failed to create PathMeta: %v", err)
	}

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

	pluginsDir := t.TempDir()
	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta:   &pathmeta.PathMeta{Shared: pathmeta.SharedPaths{Plugins: pluginsDir}},
	}}

	if _, err := m.downloadPlugin(srv.URL + "/plugin.zip"); err == nil || !strings.Contains(err.Error(), "unexpected HTTP status 403 Forbidden") {
		t.Fatalf("downloadPlugin() error = %v, want unexpected HTTP status 403 Forbidden", err)
	}

	entries, readErr := os.ReadDir(pluginsDir)
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatalf("read plugins dir: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("downloadPlugin() saved %d file(s) for an HTTP 403 response, want none", len(entries))
	}
}

func TestDownloadPluginKeepsNotFoundMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta:   &pathmeta.PathMeta{Shared: pathmeta.SharedPaths{Plugins: t.TempDir()}},
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
		PathMeta:   &pathmeta.PathMeta{Shared: pathmeta.SharedPaths{Plugins: t.TempDir()}},
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
		PathMeta:   &pathmeta.PathMeta{Shared: pathmeta.SharedPaths{Plugins: t.TempDir()}},
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
		PathMeta:   &pathmeta.PathMeta{Shared: pathmeta.SharedPaths{Plugins: t.TempDir()}},
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
		PathMeta:   &pathmeta.PathMeta{Shared: pathmeta.SharedPaths{Plugins: t.TempDir()}},
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

func TestDownloadPluginKeepsBackslashPathsInPluginDir(t *testing.T) {
	const body = "plugin bytes"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	plugins := t.TempDir()
	m := &Manager{RuntimeEnvContext: &env.RuntimeEnvContext{
		UserConfig: &config.Config{Proxy: config.EmptyProxy},
		PathMeta:   &pathmeta.PathMeta{Shared: pathmeta.SharedPaths{Plugins: plugins}},
	}}

	// A backslash in the URL path, literal or as %5C, must not let the
	// download escape the plugin directory on Windows, where filepath.Join
	// splits at it.
	for _, p := range []string{"/..\\evil.zip", "/..%5Cevil.zip", "/sub/..%5C..%5Cevil.zip"} {
		saved, err := m.downloadPlugin(srv.URL + p)
		if err != nil {
			t.Fatalf("downloadPlugin(%q) error = %v", p, err)
		}
		if rel, err := filepath.Rel(plugins, saved); err != nil || rel != filepath.Base(saved) {
			t.Fatalf("downloadPlugin() saved %q, want a file directly in %q", saved, plugins)
		}
	}
}
