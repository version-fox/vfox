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
	"runtime"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/plugin"
)

// stubShellAndRegistry keeps a test from opening a shell and from writing the
// Windows registry of whoever runs it. It returns how many shells were opened.
func stubShellAndRegistry(t *testing.T) *int {
	t.Helper()
	opened := 0
	oldOpen, oldApply, oldRemove := openShell, env.ApplyEnvsToRegistry, env.RemoveEnvsFromRegistry
	openShell = func(int) error { opened++; return nil }
	env.ApplyEnvsToRegistry = func(*env.Envs) error { return nil }
	env.RemoveEnvsFromRegistry = func(*env.Envs) error { return nil }
	t.Cleanup(func() {
		openShell, env.ApplyEnvsToRegistry, env.RemoveEnvsFromRegistry = oldOpen, oldApply, oldRemove
	})
	return &opened
}

// UseGlobal selects the version with or without the hook, and never hands off
// to a new shell.
func TestUseGlobalOpensNoShell(t *testing.T) {
	for _, hook := range []string{"", "bash"} {
		t.Run("hook="+hook, func(t *testing.T) {
			s := installedSDK(t)
			opened := stubShellAndRegistry(t)
			t.Setenv(env.HookFlag, hook)
			capturePterm(t)

			if err := s.UseGlobal("1.0.0"); err != nil {
				t.Fatalf("UseGlobal(1.0.0): %v", err)
			}
			if got := s.globalVersion(); got != "1.0.0" {
				t.Errorf("global version = %q, want 1.0.0", got)
			}
			if *opened != 0 {
				t.Errorf("opened %d shells, want none", *opened)
			}
		})
	}
}

// use -g in a Windows shell without the hook still hands off to a new shell, so
// that the user gets the new version right away.
func TestUseGlobalScopeWithoutTheHookOpensAShellOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only Windows falls back to the global scope without the hook")
	}
	s := installedSDK(t)
	opened := stubShellAndRegistry(t)
	t.Setenv(env.HookFlag, "")
	capturePterm(t)

	if err := s.UseWithConfig("1.0.0", env.Global, false); err != nil {
		t.Fatalf("UseWithConfig(1.0.0, global): %v", err)
	}
	if *opened != 1 {
		t.Errorf("opened %d shells, want 1", *opened)
	}
}

// An error from the plugin's EnvKeys hook stops use, and the global config keeps
// the version it had.
func TestUseReturnsTheEnvKeysError(t *testing.T) {
	boom := false
	s := installedSDK(t, func(p *testPlugin) {
		p.envKeys = func(ctx *plugin.EnvKeysHookCtx) ([]*plugin.EnvKeysHookResultItem, error) {
			if boom {
				return nil, errors.New("EnvKeys exploded")
			}
			return []*plugin.EnvKeysHookResultItem{{Key: "PATH", Value: filepath.Join(ctx.Path, "bin")}}, nil
		}
	})
	opened := stubShellAndRegistry(t)
	t.Setenv(env.HookFlag, "bash")
	capturePterm(t)
	mustInstall(t, s, "2.0.0")
	if err := s.UseWithConfig("1.0.0", env.Global, false); err != nil {
		t.Fatalf("UseWithConfig(1.0.0, global): %v", err)
	}

	boom = true
	err := s.UseWithConfig("2.0.0", env.Global, false)
	if err == nil || !strings.Contains(err.Error(), "EnvKeys exploded") {
		t.Fatalf("UseWithConfig(2.0.0, global) = %v, want the EnvKeys error", err)
	}
	if got := s.globalVersion(); got != "1.0.0" {
		t.Errorf("global version = %q, want 1.0.0 unchanged", got)
	}
	if *opened != 0 {
		t.Errorf("opened %d shells, want none", *opened)
	}
}

// A version that is not installed is refused, and the global config keeps the
// version it had.
func TestUseGlobalRefusesAVersionThatIsNotInstalled(t *testing.T) {
	s := installedSDK(t)
	stubShellAndRegistry(t)
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
	stubShellAndRegistry(t)
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
