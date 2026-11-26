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

package sdk

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
)

func TestAvailableCacheIncludesEnvReadByHook(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "elixir")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	countFile := filepath.Join(root, "calls")
	source := `PLUGIN = {name = "elixir", version = "0.0.1"}
function PLUGIN:Available(ctx)
  local f = assert(io.open(os.getenv("VFOX_COUNT_FILE"), "a"))
  f:write("x")
  f:close()
  local mirror = os.getenv("VFOX_ELIXIR_MIRROR") or "release"
  return {{version = mirror}}
end
function PLUGIN:PreInstall(ctx) return {version = ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key = "PATH", value = ctx.path}} end
`
	if err := os.WriteFile(filepath.Join(pluginDir, "main.lua"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	sdk, err := NewSdk(&env.RuntimeEnvContext{
		UserConfig: config.DefaultConfig.Clone(),
		PathMeta:   &pathmeta.PathMeta{Shared: pathmeta.SharedPaths{Installs: filepath.Join(root, "installs")}},
	}, pluginDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sdk.Close)

	t.Setenv("VFOX_COUNT_FILE", countFile)
	t.Setenv("VFOX_ELIXIR_MIRROR", "release")
	first, err := sdk.Available(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || string(first[0].Version) != "release" {
		t.Fatalf("first = %+v", first)
	}

	t.Setenv("VFOX_ELIXIR_MIRROR", "hex")
	second, err := sdk.Available(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != 1 || string(second[0].Version) != "hex" {
		t.Fatalf("cached the release list for hex: %+v", second)
	}

	third, err := sdk.Available(nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(third[0].Version) != "hex" {
		t.Fatalf("third = %s", third[0].Version)
	}
	calls, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(calls) != "xx" {
		t.Fatalf("hook calls = %q, want two (the third search is cached)", calls)
	}
}
