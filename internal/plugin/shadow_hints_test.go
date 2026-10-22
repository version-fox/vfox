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

package plugin

import (
	"reflect"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
)

func newTestPlugin(t *testing.T, source string) *Wrapper {
	t.Helper()
	dir := t.TempDir()
	writePluginFile(t, dir, "main.lua", minimalPlugin+source)
	p, err := CreatePlugin(dir, &env.RuntimeEnvContext{UserConfig: config.DefaultConfig})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(p.Close)
	return p
}

func TestShadowHintsMetadata(t *testing.T) {
	p := newTestPlugin(t, `
PLUGIN.shadowHints = {
  {match = "pnpm", message = "pnpm installed its own node.", url = "https://example.com/pnpm.md"},
  {match = "brew", message = "Homebrew's node comes first."},
}`)
	want := []ShadowHint{
		{Match: "pnpm", Message: "pnpm installed its own node.", URL: "https://example.com/pnpm.md"},
		{Match: "brew", Message: "Homebrew's node comes first."},
	}
	if !reflect.DeepEqual(p.ShadowHints, want) {
		t.Errorf("ShadowHints = %+v, want %+v", p.ShadowHints, want)
	}
	// Plugins written before the field existed have none, and old vfox ignores
	// the field on plugins that already have it: nothing else may depend on it.
	if plain := newTestPlugin(t, ""); len(plain.ShadowHints) != 0 {
		t.Errorf("ShadowHints = %+v, want none", plain.ShadowHints)
	}
}
