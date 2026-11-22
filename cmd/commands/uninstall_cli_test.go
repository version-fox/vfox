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

package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

// keepRegistry keeps the commands the test runs in this process from writing
// the Windows registry of whoever runs it.
func keepRegistry(t *testing.T) {
	t.Helper()
	oldApply, oldRemove := env.ApplyEnvsToRegistry, env.RemoveEnvsFromRegistry
	env.ApplyEnvsToRegistry = func(*env.Envs) error { return nil }
	env.RemoveEnvsFromRegistry = func(*env.Envs) error { return nil }
	t.Cleanup(func() { env.ApplyEnvsToRegistry, env.RemoveEnvsFromRegistry = oldApply, oldRemove })
}

// Uninstalling a version switches the global scope away from it only when the
// global scope asks for it, not when only the project does.
func TestUninstallLeavesTheGlobalVersionOfOtherScopes(t *testing.T) {
	f := newCLIFixture(t, false)
	for _, v := range []string{"1.0.0", "2.0.0", "3.0.0"} {
		f.installOnDisk(t, "alpha", v)
	}
	globalConfig := filepath.Join(f.userHome, ".vfox.toml")
	must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nalpha = \"3.0.0\"\n"), 0644))

	t.Run("a version only the project asks for", func(t *testing.T) {
		must(t, os.WriteFile(globalConfig, []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))
		out := f.run(t, "uninstall", "alpha@3.0.0").mustSucceed(t, "uninstall alpha@3.0.0").output()
		mustNotContain(t, "uninstall", out, "Auto switch")
		mustContain(t, "global config", readFile(t, globalConfig), `alpha = "1.0.0"`)
	})

	t.Run("the global version", func(t *testing.T) {
		keepRegistry(t)
		must(t, os.WriteFile(globalConfig, []byte("[tools]\nalpha = \"2.0.0\"\n"), 0644))
		out := f.run(t, "uninstall", "alpha@2.0.0").mustSucceed(t, "uninstall alpha@2.0.0").output()
		mustContain(t, "uninstall", out, "Auto switch")
		mustContain(t, "global config", readFile(t, globalConfig), `alpha = "1.0.0"`)
	})
}

// Uninstalling the global version in a shell that was not activated still moves
// the global version to one that is left, and succeeds: the global scope does not
// need the hook.
func TestUninstallSwitchesTheGlobalVersionWithoutAHook(t *testing.T) {
	f := newCLIFixture(t, false)
	for _, v := range []string{"1.0.0", "2.0.0"} {
		f.installOnDisk(t, "alpha", v)
	}
	globalConfig := filepath.Join(f.userHome, ".vfox.toml")
	must(t, os.WriteFile(globalConfig, []byte("[tools]\nalpha = \"2.0.0\"\n"), 0644))
	f.env["__VFOX_SHELL"] = "-"
	keepRegistry(t)

	out := f.run(t, "uninstall", "alpha@2.0.0").mustSucceed(t, "uninstall alpha@2.0.0").output()
	mustContain(t, "uninstall", out, "Auto switch", "alpha@1.0.0")
	mustNotContain(t, "uninstall", out, "hook support")
	mustContain(t, "global config", readFile(t, globalConfig), `alpha = "1.0.0"`)
}

// The version that takes over is the newest one left by version number, not by
// text: 10.0.0 is newer than 9.0.0. vfox list shows them in the same order.
func TestUninstallSwitchesToTheNewestVersionLeft(t *testing.T) {
	f := newCLIFixture(t, false)
	for _, v := range []string{"9.0.0", "10.0.0", "11.0.0"} {
		f.installOnDisk(t, "alpha", v)
	}
	globalConfig := filepath.Join(f.userHome, ".vfox.toml")
	must(t, os.WriteFile(globalConfig, []byte("[tools]\nalpha = \"11.0.0\"\n"), 0644))
	keepRegistry(t)

	list := f.run(t, "list", "alpha").mustSucceed(t, "list alpha").output()
	if i11, i10, i9 := strings.Index(list, "11.0.0"), strings.Index(list, "10.0.0"), strings.Index(list, "9.0.0"); !(i11 >= 0 && i11 < i10 && i10 < i9) {
		t.Errorf("list alpha shows the versions out of order, want 11.0.0, 10.0.0, 9.0.0:\n%s", list)
	}
	f.run(t, "uninstall", "alpha@11.0.0").mustSucceed(t, "uninstall alpha@11.0.0")
	mustContain(t, "global config", readFile(t, globalConfig), `alpha = "10.0.0"`)
}

// A version whose directory cannot be removed is still installed, so it, or the
// version that took over, stays the global version: never none.
func TestUninstallKeepsAVersionInUseWhenTheRemovalFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a read-only directory can still be emptied on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can remove a read-only directory")
	}
	for _, tc := range []struct {
		name, want string
		installed  []string
	}{
		{"another version is left", "1.0.0", []string{"1.0.0", "2.0.0"}},
		{"no other version is left", "2.0.0", []string{"2.0.0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCLIFixture(t, false)
			for _, v := range tc.installed {
				f.installOnDisk(t, "alpha", v)
			}
			globalConfig := filepath.Join(f.userHome, ".vfox.toml")
			must(t, os.WriteFile(globalConfig, []byte("[tools]\nalpha = \"2.0.0\"\n"), 0644))
			keepRegistry(t)
			locked := filepath.Join(f.shared, "cache", "alpha", "v-2.0.0")
			must(t, os.Chmod(locked, 0555))
			t.Cleanup(func() { _ = os.Chmod(locked, 0755) })

			f.run(t, "uninstall", "alpha@2.0.0").mustFail(t, "uninstall alpha@2.0.0")
			mustContain(t, "global config", readFile(t, globalConfig), `alpha = "`+tc.want+`"`)
		})
	}
}

// betaPlugin fails to give the environment of version 1.0.0.
const betaPlugin = `PLUGIN = {name="beta", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx)
  if ctx.main.version == "1.0.0" then error("EnvKeys exploded") end
  return {{key="PATH", value=ctx.path .. "/bin"}}
end
`

// When the version that would take over cannot be used, nothing is removed and
// the global version stays as it was.
func TestUninstallKeepsTheGlobalVersionWhenTheSwitchFails(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "beta", betaPlugin)
	for _, v := range []string{"1.0.0", "2.0.0"} {
		f.installOnDisk(t, "beta", v)
	}
	globalConfig := filepath.Join(f.userHome, ".vfox.toml")
	must(t, os.WriteFile(globalConfig, []byte("[tools]\nbeta = \"2.0.0\"\n"), 0644))
	keepRegistry(t)

	out := f.run(t, "uninstall", "beta@2.0.0").mustFail(t, "uninstall beta@2.0.0").output()
	mustContain(t, "uninstall", out, "EnvKeys exploded")
	mustContain(t, "global config", readFile(t, globalConfig), `beta = "2.0.0"`)
	if _, err := os.Stat(filepath.Join(f.shared, "cache", "beta", "v-2.0.0")); err != nil {
		t.Errorf("beta@2.0.0 was removed: %v", err)
	}
}
