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
	"testing"
)

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
		must(t, os.WriteFile(globalConfig, []byte("[tools]\nalpha = \"2.0.0\"\n"), 0644))
		out := f.run(t, "uninstall", "alpha@2.0.0").mustSucceed(t, "uninstall alpha@2.0.0").output()
		mustContain(t, "uninstall", out, "Auto switch")
		mustContain(t, "global config", readFile(t, globalConfig), `alpha = "1.0.0"`)
	})
}

// Uninstalling the version the global scope uses must succeed even when the
// shell has no vfox hook (#809): the version is removed, a remaining version
// takes its place in the global config, and the command exits 0. Only a
// hook-related warning may remain.
func TestUninstallCurrentGlobalVersionWithoutShellHook(t *testing.T) {
	f := newCLIFixture(t, false)
	f.env["__VFOX_SHELL"] = "-" // this shell was never activated
	for _, v := range []string{"1.0.0", "2.0.0"} {
		f.installOnDisk(t, "alpha", v)
	}
	globalConfig := filepath.Join(f.userHome, ".vfox.toml")
	must(t, os.WriteFile(globalConfig, []byte("[tools]\nalpha = \"2.0.0\"\n"), 0644))

	out := f.run(t, "uninstall", "alpha@2.0.0").mustSucceed(t, "uninstall the current version without a shell hook").output()
	mustContain(t, "uninstall output", out, "Uninstalled alpha@2.0.0 successfully!", "Auto switch")
	if _, err := os.Stat(filepath.Join(f.shared, "cache", "alpha", "v-2.0.0")); !os.IsNotExist(err) {
		t.Errorf("alpha@2.0.0 is still installed: %v", err)
	}
	mustContain(t, "global config", readFile(t, globalConfig), `alpha = "1.0.0"`)
	current := f.run(t, "current", "alpha").mustSucceed(t, "current alpha").output()
	mustContain(t, "current output", current, "1.0.0")
}

// Uninstalling the only installed version without a shell hook must also
// succeed and leave nothing behind (#809).
func TestUninstallOnlyVersionWithoutShellHook(t *testing.T) {
	f := newCLIFixture(t, false)
	f.env["__VFOX_SHELL"] = "-" // this shell was never activated
	f.installOnDisk(t, "alpha", "1.0.0")
	globalConfig := filepath.Join(f.userHome, ".vfox.toml")
	must(t, os.WriteFile(globalConfig, []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))

	out := f.run(t, "uninstall", "alpha@1.0.0").mustSucceed(t, "uninstall the only version without a shell hook").output()
	mustContain(t, "uninstall output", out, "Uninstalled alpha@1.0.0 successfully!")
	mustNotContain(t, "uninstall output", out, "Auto switch")
	if _, err := os.Stat(filepath.Join(f.shared, "cache", "alpha")); !os.IsNotExist(err) {
		t.Errorf("the alpha install directory is still there: %v", err)
	}
	mustNotContain(t, "global config", readFile(t, globalConfig), "alpha")
	current := f.run(t, "current", "alpha").mustFail(t, "current alpha").output()
	mustContain(t, "current output", current, "no current version")
}
