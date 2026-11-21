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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/plugin"
)

// A version that is not installed is refused, and the global config keeps the
// version it had.
func TestUseGlobalRefusesAVersionThatIsNotInstalled(t *testing.T) {
	s := installedSDK(t)
	t.Setenv(env.HookFlag, "") // UseGlobal does not need the hook
	capturePterm(t)

	if err := s.UseGlobal("1.0.0"); err != nil {
		t.Fatalf("UseGlobal(1.0.0): %v", err)
	}
	err := s.UseGlobal("2.0.0")
	var notExists *VersionNotExistsError
	if !errors.As(err, &notExists) {
		t.Fatalf("UseGlobal(2.0.0) = %v, want a VersionNotExistsError", err)
	}
	if got := s.globalVersion(); got != "1.0.0" {
		t.Errorf("global version = %q, want 1.0.0 unchanged", got)
	}
}

const failingPreUsePlugin = `PLUGIN = {name="delta", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
function PLUGIN:PreUse(ctx) error("no version for you") end
`

// An error from the plugin's PreUse hook is returned, and the global config is
// not written.
func TestUseGlobalReturnsThePreUseError(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	dir := filepath.Join(t.TempDir(), "delta")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.lua"), []byte(failingPreUsePlugin), 0644); err != nil {
		t.Fatal(err)
	}
	plug, err := plugin.CreatePlugin(dir, s.envContext)
	if err != nil {
		t.Fatalf("CreatePlugin: %v", err)
	}
	defer plug.Close()
	s.plugin = plug
	s.Name = "delta"
	t.Setenv(env.HookFlag, "")
	capturePterm(t)
	mustInstall(t, s, "1.0.0")

	err = s.UseGlobal("1.0.0")
	if err == nil || !strings.Contains(err.Error(), "no version for you") {
		t.Fatalf("UseGlobal(1.0.0) = %v, want the PreUse error", err)
	}
	if got := s.globalVersion(); got != "" {
		t.Errorf("global version = %q, want none", got)
	}
}
