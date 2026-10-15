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

// PostUse is optional: existing plugins without it must keep loading.
func TestPostUseHook_Optional(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	manager, err := internal.NewSdkManager()
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	for _, path := range []string{pluginPathWithMain, pluginPathWithMetadata} {
		plug, err := plugin.CreatePlugin(path, manager.RuntimeEnvContext)
		if err != nil {
			t.Fatalf("CreatePlugin(%s) failed: %v", path, err)
		}
		if !plug.HasFunction("PostUse") {
			t.Errorf("expected PostUse function in fixture %s", path)
		}
	}
}

// TestPostUseHook_InvokesLua verifies ctx fields reach Lua and shim dir is honored.
func TestPostUseHook_InvokesLua(t *testing.T) {
	teardownSuite := setupSuite(t)
	defer teardownSuite(t)

	manager, err := internal.NewSdkManager()
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()

	shimDir := t.TempDir()

	for _, path := range []string{pluginPathWithMain, pluginPathWithMetadata} {
		plug, err := plugin.CreatePlugin(path, manager.RuntimeEnvContext)
		if err != nil {
			t.Fatalf("CreatePlugin(%s) failed: %v", path, err)
		}
		ctx := &plugin.PostUseHookCtx{
			Cwd:             "/home/user/project",
			Scope:           "global",
			Version:         "1.0.0",
			PreviousVersion: "0.9.0",
			Main: &plugin.InstalledPackageItem{
				Name:    "java",
				Version: "1.0.0",
				Path:    "/path/to/java-1.0.0",
			},
			SdkInfo: map[string]*plugin.InstalledPackageItem{
				"java": {
					Name:    "java",
					Version: "1.0.0",
					Path:    "/path/to/java-1.0.0",
				},
			},
			ShimDir: shimDir,
		}
		if err := plug.PostUse(ctx); err != nil {
			t.Errorf("PostUse(%s) failed: %v", path, err)
		}
	}
}

// TestPostUseHook_MissingShimDirFails ensures Lua fixtures validate the generic contract.
func TestPostUseHook_MissingShimDirFails(t *testing.T) {
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
	ctx := &plugin.PostUseHookCtx{
		Cwd:     "/home/user",
		Scope:   "global",
		Version: "1.0.0",
		Main: &plugin.InstalledPackageItem{
			Name:    "java",
			Version: "1.0.0",
			Path:    "/path",
		},
		SdkInfo: map[string]*plugin.InstalledPackageItem{},
		ShimDir: "",
	}
	if err := plug.PostUse(ctx); err == nil {
		t.Error("expected PostUse to fail with empty shimDir, got nil")
	}
}
