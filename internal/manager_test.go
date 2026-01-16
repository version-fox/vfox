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
	"os"
	"path/filepath"
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

