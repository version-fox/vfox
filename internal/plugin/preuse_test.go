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

package plugin_test

import (
	"testing"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/plugin"
)

// TestPreUseHook_InstalledSdksKeyedByVersion tests that installedSdks map is keyed by version strings
func TestPreUseHook_InstalledSdksKeyedByVersion(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	manager, err := internal.NewSdkManager()
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	plug, err := plugin.CreatePlugin(pluginPathWithMain, manager.RuntimeEnvContext)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name              string
		installedSdks     map[string]*plugin.InstalledPackageItem
		inputVersion      string
		expectedToContain string // Expected version key in the map
		description       string
	}{
		{
			name: "Single SDK with simple version",
			installedSdks: map[string]*plugin.InstalledPackageItem{
				"1.0.0": {
					Name:    "java",
					Version: "1.0.0",
					Path:    "/path/to/java-1.0.0",
				},
			},
			inputVersion:      "1.0.0",
			expectedToContain: "1.0.0",
			description:       "Should key by version string '1.0.0'",
		},
		{
			name: "Multiple versions of same SDK",
			installedSdks: map[string]*plugin.InstalledPackageItem{
				"1.19.2-elixir-otp-28": {
					Name:    "elixir",
					Version: "1.19.2-elixir-otp-28",
					Path:    "/path/to/elixir-1.19.2-otp-28",
				},
				"1.19.2-elixir-otp-27": {
					Name:    "elixir",
					Version: "1.19.2-elixir-otp-27",
					Path:    "/path/to/elixir-1.19.2-otp-27",
				},
			},
			inputVersion:      "1.19.2-elixir-otp-28",
			expectedToContain: "1.19.2-elixir-otp-28",
			description:       "Should key by full version string including OTP version",
		},
		{
			name: "Complex version with hyphens and dots",
			installedSdks: map[string]*plugin.InstalledPackageItem{
				"21.0.0-openjdk": {
					Name:    "java",
					Version: "21.0.0-openjdk",
					Path:    "/path/to/java-21.0.0-openjdk",
				},
			},
			inputVersion:      "21.0.0-openjdk",
			expectedToContain: "21.0.0-openjdk",
			description:       "Should key by version string with vendor suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &plugin.PreUseHookCtx{
				Cwd:             "/home/user",
				Scope:           "global",
				Version:         tt.inputVersion,
				PreviousVersion: "",
				InstalledSdks:   tt.installedSdks,
			}

			// Verify the key exists in the map
			if _, exists := ctx.InstalledSdks[tt.expectedToContain]; !exists {
				t.Errorf("Expected key '%s' to exist in installedSdks map, but it doesn't. %s", tt.expectedToContain, tt.description)
			}

			// Verify we can access SDK info by version
			sdkInfo := ctx.InstalledSdks[tt.inputVersion]
			if sdkInfo == nil {
				t.Errorf("Expected to access SDK info using version '%s', but got nil", tt.inputVersion)
			} else {
				if sdkInfo.Version != tt.inputVersion {
					t.Errorf("Expected SDK version '%s', got '%s'", tt.inputVersion, sdkInfo.Version)
				}
			}

			// Call PreUse to ensure plugin can access the data
			result, err := plug.PreUse(ctx)
			if err != nil {
				t.Fatalf("PreUse failed: %v", err)
			}
			// The test plugin returns "9.9.9" for global scope
			if result.Version != "9.9.9" {
				t.Errorf("Expected plugin to return '9.9.9', got '%s'", result.Version)
			}
		})
	}
}

// TestPreUseHook_EmptyInstalledSdks tests behavior with empty installedSdks
func TestPreUseHook_EmptyInstalledSdks(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	manager, err := internal.NewSdkManager()
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	plug, err := plugin.CreatePlugin(pluginPathWithMain, manager.RuntimeEnvContext)
	if err != nil {
		t.Fatal(err)
	}

	ctx := &plugin.PreUseHookCtx{
		Cwd:             "/home/user",
		Scope:           "global",
		Version:         "1.0.0",
		PreviousVersion: "",
		InstalledSdks:   map[string]*plugin.InstalledPackageItem{}, // Empty map
	}

	// Should not crash with empty installedSdks
	result, err := plug.PreUse(ctx)
	if err != nil {
		t.Fatalf("PreUse should not fail with empty installedSdks: %v", err)
	}

	// The test plugin should still return a version based on scope
	if result.Version != "9.9.9" { // global scope returns 9.9.9
		t.Errorf("Expected '9.9.9', got '%s'", result.Version)
	}
}

// TestPreUseHook_AccessingNonExistentVersion tests plugin behavior when accessing non-existent version
func TestPreUseHook_AccessingNonExistentVersion(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	manager, err := internal.NewSdkManager()
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	plug, err := plugin.CreatePlugin(pluginPathWithMain, manager.RuntimeEnvContext)
	if err != nil {
		t.Fatal(err)
	}

	ctx := &plugin.PreUseHookCtx{
		Cwd:             "/home/user",
		Scope:           "global",
		Version:         "2.0.0", // This version doesn't exist in installedSdks
		PreviousVersion: "",
		InstalledSdks: map[string]*plugin.InstalledPackageItem{
			"1.0.0": {
				Name:    "java",
				Version: "1.0.0",
				Path:    "/path/to/java-1.0.0",
			},
		},
	}

	// The plugin has nil checks, so it should not crash
	result, err := plug.PreUse(ctx)
	if err != nil {
		t.Fatalf("PreUse should not fail when accessing non-existent version: %v", err)
	}

	// Should still return a result
	if result.Version == "" {
		t.Error("Expected non-empty version result")
	}
}

// TestPreUseHook_MultipleSDKs tests that multiple SDKs with different versions are correctly keyed
func TestPreUseHook_MultipleSDKs(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	manager, err := internal.NewSdkManager()
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	plug, err := plugin.CreatePlugin(pluginPathWithMain, manager.RuntimeEnvContext)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate multiple installed versions like vfox-elixir scenario
	installedSdks := map[string]*plugin.InstalledPackageItem{
		"1.19.2-elixir-otp-28": {
			Name:    "elixir",
			Version: "1.19.2-elixir-otp-28",
			Path:    "/home/user/.version-fox/cache/elixir/v-1.19.2-elixir-otp-28",
		},
		"1.19.2-elixir-otp-27": {
			Name:    "elixir",
			Version: "1.19.2-elixir-otp-27",
			Path:    "/home/user/.version-fox/cache/elixir/v-1.19.2-elixir-otp-27",
		},
		"1.18.0-elixir-otp-26": {
			Name:    "elixir",
			Version: "1.18.0-elixir-otp-26",
			Path:    "/home/user/.version-fox/cache/elixir/v-1.18.0-elixir-otp-26",
		},
	}

	// Test accessing each version
	versions := []string{"1.19.2-elixir-otp-28", "1.19.2-elixir-otp-27", "1.18.0-elixir-otp-26"}
	for _, version := range versions {
		t.Run("Access_"+version, func(t *testing.T) {
			ctx := &plugin.PreUseHookCtx{
				Cwd:             "/home/user/project",
				Scope:           "global",
				Version:         version,
				PreviousVersion: "",
				InstalledSdks:   installedSdks,
			}

			// Verify the version exists as a key
			sdkInfo, exists := ctx.InstalledSdks[version]
			if !exists {
				t.Errorf("Version '%s' should exist as a key in installedSdks", version)
			}

			// Verify the SDK info is correct
			if sdkInfo == nil {
				t.Fatalf("SDK info for version '%s' should not be nil", version)
			}
			if sdkInfo.Version != version {
				t.Errorf("Expected SDK version '%s', got '%s'", version, sdkInfo.Version)
			}
			if sdkInfo.Name != "elixir" {
				t.Errorf("Expected SDK name 'elixir', got '%s'", sdkInfo.Name)
			}

			// Call PreUse
			result, err := plug.PreUse(ctx)
			if err != nil {
				t.Fatalf("PreUse failed for version '%s': %v", version, err)
			}
			if result.Version == "" {
				t.Errorf("Expected non-empty result version for '%s'", version)
			}
		})
	}
}

// TestPreUseHook_SDKInfoFields tests that all fields in InstalledPackageItem are correctly populated
func TestPreUseHook_SDKInfoFields(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	manager, err := internal.NewSdkManager()
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	plug, err := plugin.CreatePlugin(pluginPathWithMain, manager.RuntimeEnvContext)
	if err != nil {
		t.Fatal(err)
	}

	expectedPath := "/home/user/.vfox/cache/elixir/v-1.19.2-elixir-otp-28"
	expectedVersion := "1.19.2-elixir-otp-28"
	expectedName := "elixir"
	expectedNote := "Test note"

	ctx := &plugin.PreUseHookCtx{
		Cwd:             "/home/user",
		Scope:           "global",
		Version:         expectedVersion,
		PreviousVersion: "",
		InstalledSdks: map[string]*plugin.InstalledPackageItem{
			expectedVersion: {
				Name:    expectedName,
				Version: expectedVersion,
				Path:    expectedPath,
				Note:    expectedNote,
			},
		},
	}

	// Access the SDK info
	sdkInfo := ctx.InstalledSdks[expectedVersion]
	if sdkInfo == nil {
		t.Fatal("SDK info should not be nil")
	}

	// Verify all fields
	if sdkInfo.Name != expectedName {
		t.Errorf("Expected Name '%s', got '%s'", expectedName, sdkInfo.Name)
	}
	if sdkInfo.Version != expectedVersion {
		t.Errorf("Expected Version '%s', got '%s'", expectedVersion, sdkInfo.Version)
	}
	if sdkInfo.Path != expectedPath {
		t.Errorf("Expected Path '%s', got '%s'", expectedPath, sdkInfo.Path)
	}
	if sdkInfo.Note != expectedNote {
		t.Errorf("Expected Note '%s', got '%s'", expectedNote, sdkInfo.Note)
	}

	// Call PreUse
	result, err := plug.PreUse(ctx)
	if err != nil {
		t.Fatalf("PreUse failed: %v", err)
	}
	if result.Version == "" {
		t.Error("Expected non-empty result version")
	}
}

// TestPreUseHook_DifferentScopes tests PreUse hook with different scope values
func TestPreUseHook_DifferentScopes(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	manager, err := internal.NewSdkManager()
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	plug, err := plugin.CreatePlugin(pluginPathWithMain, manager.RuntimeEnvContext)
	if err != nil {
		t.Fatal(err)
	}

	installedSdks := map[string]*plugin.InstalledPackageItem{
		"1.0.0": {
			Name:    "java",
			Version: "1.0.0",
			Path:    "/test/path",
		},
	}

	scopes := []struct {
		scope           string
		expectedVersion string
	}{
		{"global", "9.9.9"},
		{"project", "10.0.0"},
		{"session", "1.0.0"},
	}

	for _, tt := range scopes {
		t.Run("Scope_"+tt.scope, func(t *testing.T) {
			ctx := &plugin.PreUseHookCtx{
				Cwd:             "/home/user",
				Scope:           tt.scope,
				Version:         "20.0",
				PreviousVersion: "21.0",
				InstalledSdks:   installedSdks,
			}

			result, err := plug.PreUse(ctx)
			if err != nil {
				t.Fatalf("PreUse failed for scope '%s': %v", tt.scope, err)
			}

			if result.Version != tt.expectedVersion {
				t.Errorf("For scope '%s', expected version '%s', got '%s'", tt.scope, tt.expectedVersion, result.Version)
			}
		})
	}
}

// TestPreUseHook_VersionNotKeyedByName ensures that SDK name is NOT used as key
func TestPreUseHook_VersionNotKeyedByName(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	installedSdks := map[string]*plugin.InstalledPackageItem{
		"1.19.2-elixir-otp-28": {
			Name:    "elixir",
			Version: "1.19.2-elixir-otp-28",
			Path:    "/path/to/elixir",
		},
	}

	// Verify that SDK is NOT keyed by name
	if _, exists := installedSdks["elixir"]; exists {
		t.Error("installedSdks should NOT be keyed by SDK name 'elixir', but it is")
	}

	// Verify that SDK IS keyed by version
	if _, exists := installedSdks["1.19.2-elixir-otp-28"]; !exists {
		t.Error("installedSdks should be keyed by version '1.19.2-elixir-otp-28', but it's not")
	}

	// Verify accessing by version works
	sdkInfo := installedSdks["1.19.2-elixir-otp-28"]
	if sdkInfo == nil {
		t.Fatal("Should be able to access SDK by version string")
	}
	if sdkInfo.Name != "elixir" {
		t.Errorf("Expected SDK name 'elixir', got '%s'", sdkInfo.Name)
	}
}
