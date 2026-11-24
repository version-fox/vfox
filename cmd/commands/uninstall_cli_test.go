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
	mustNotContain(t, "uninstall", out, "hook support", "Now using")
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

// skipUnlessPermissionsHold skips a test that takes away write permission to make
// a step fail, where that does not stop the step.
func skipUnlessPermissionsHold(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("a read-only directory can still be changed on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can change a read-only directory")
	}
}

// lock takes away write permission on path and returns a function that gives it back.
func lock(t *testing.T, path string) (unlock func()) {
	t.Helper()
	info, err := os.Stat(path)
	must(t, err)
	must(t, os.Chmod(path, info.Mode().Perm()&^0222))
	unlock = func() { _ = os.Chmod(path, info.Mode().Perm()) }
	t.Cleanup(unlock)
	return unlock
}

// A version that cannot be removed at all stays installed, and the config and the
// links are what they were. Once the cause is gone, the same command works.
func TestUninstallChangesNothingWhenTheRemovalCannotStart(t *testing.T) {
	skipUnlessPermissionsHold(t)
	for _, tc := range []struct {
		name      string
		installed []string
		after     string
	}{
		{"another version is left", []string{"1.0.0", "2.0.0"}, `alpha = "1.0.0"`},
		{"no other version is left", []string{"2.0.0"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCLIFixture(t, false)
			for _, v := range tc.installed {
				f.installOnDisk(t, "alpha", v)
			}
			installs := filepath.Join(f.shared, "cache", "alpha")
			must(t, os.MkdirAll(filepath.Join(installs, "v-2.0.0", "extra-2.0.0"), 0755))
			globalConfig := filepath.Join(f.userHome, ".vfox.toml")
			keepRegistry(t)
			f.run(t, "use", "-g", "alpha@2.0.0").mustSucceed(t, "use -g alpha@2.0.0")
			unlock := lock(t, installs)

			out := f.run(t, "uninstall", "alpha@2.0.0").mustFail(t, "uninstall alpha@2.0.0").output()
			mustContain(t, "uninstall", out, "alpha@2.0.0", "still installed")
			mustNotContain(t, "uninstall", out, "Unset", "Auto switch", "Now using", "Uninstalled")
			mustContain(t, "global config", readFile(t, globalConfig), `alpha = "2.0.0"`)
			for _, name := range []string{"alpha", "extra"} {
				target, err := os.Readlink(filepath.Join(f.userHome, "sdks", name))
				must(t, err)
				mustContain(t, "link of "+name, target, "v-2.0.0")
			}
			f.run(t, "list", "alpha").mustSucceed(t, "list alpha")

			unlock()
			f.run(t, "uninstall", "alpha@2.0.0").mustSucceed(t, "uninstall alpha@2.0.0 again")
			if tc.after == "" {
				mustNotContain(t, "global config", readFile(t, globalConfig), "alpha")
			} else {
				mustContain(t, "global config", readFile(t, globalConfig), tc.after)
			}
			if _, err := os.Lstat(filepath.Join(f.userHome, "sdks", "extra")); !os.IsNotExist(err) {
				t.Errorf("the link of the addition is still there: %v", err)
			}
		})
	}
}

// When files of the version cannot be deleted, the version is gone all the same:
// the config and the links no longer name it, the command says what is left and
// fails, and a retry deletes the rest.
func TestUninstallThatLeavesFilesIsFinishedByARetry(t *testing.T) {
	skipUnlessPermissionsHold(t)
	for _, tc := range []struct {
		name      string
		installed []string
		says      string
	}{
		{"another version is left", []string{"1.0.0", "2.0.0"}, "Auto switch"},
		{"no other version is left", []string{"2.0.0"}, "Unset"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCLIFixture(t, false)
			for _, v := range tc.installed {
				f.installOnDisk(t, "alpha", v)
			}
			installs := filepath.Join(f.shared, "cache", "alpha")
			inside := filepath.Join(installs, "v-2.0.0", "extra-2.0.0", "locked")
			must(t, os.MkdirAll(inside, 0755))
			must(t, os.WriteFile(filepath.Join(inside, "file"), nil, 0644))
			globalConfig := filepath.Join(f.userHome, ".vfox.toml")
			keepRegistry(t)
			f.run(t, "use", "-g", "alpha@2.0.0").mustSucceed(t, "use -g alpha@2.0.0")
			f.run(t, "alias", "set", "alpha", "mine", "2.0.0").mustSucceed(t, "alias set")
			must(t, os.Chmod(inside, 0555))
			left := filepath.Join(installs, ".v-2.0.0.removing")
			unlock := func() { _ = os.Chmod(filepath.Join(left, "extra-2.0.0", "locked"), 0755) }
			t.Cleanup(unlock)

			out := f.run(t, "uninstall", "alpha@2.0.0").mustFail(t, "uninstall alpha@2.0.0").output()
			mustContain(t, "uninstall", out, "no longer installed", tc.says, "could not delete all files of alpha@2.0.0", left, "Removed the alias(es) mine")
			mustNotContain(t, "uninstall", out, "Uninstalled", "Now using")
			mustNotContain(t, "global config", readFile(t, globalConfig), "2.0.0")
			if _, err := os.Lstat(filepath.Join(f.userHome, "sdks", "extra")); !os.IsNotExist(err) {
				t.Errorf("the link of the addition is still there: %v", err)
			}
			if target, err := os.Readlink(filepath.Join(f.userHome, "sdks", "alpha")); err == nil && strings.Contains(target, "2.0.0") {
				t.Errorf("the link of alpha still points at %s", target)
			}

			out = f.run(t, "uninstall", "alpha@2.0.0").mustFail(t, "uninstall alpha@2.0.0 again").output()
			mustContain(t, "uninstall again", out, "could not delete all files of alpha@2.0.0")

			unlock()
			f.run(t, "uninstall", "alpha@2.0.0").mustSucceed(t, "the last uninstall alpha@2.0.0")
			if _, err := os.Lstat(left); !os.IsNotExist(err) {
				t.Errorf("%s is still there: %v", left, err)
			}
			out = f.run(t, "uninstall", "alpha@2.0.0").mustFail(t, "one more uninstall alpha@2.0.0").output()
			mustContain(t, "one more uninstall", out, "not installed")
		})
	}
}

// Files left of v1.0.0 do not change what v1.0.0 stands for while 1.0.0 is
// installed. Once it is not, the name as typed finds them.
func TestLeftoverFilesDoNotChangeALeadingV(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0")
	keepRegistry(t)
	installs := filepath.Join(f.shared, "cache", "alpha")
	left := filepath.Join(installs, ".v-v1.0.0.removing")
	must(t, os.MkdirAll(left, 0755))

	f.run(t, "use", "-g", "alpha@v1.0.0").mustSucceed(t, "use -g alpha@v1.0.0")
	mustContain(t, "global config", readFile(t, filepath.Join(f.userHome, ".vfox.toml")), `alpha = "1.0.0"`)

	out := f.run(t, "uninstall", "alpha@v1.0.0").mustSucceed(t, "uninstall alpha@v1.0.0").output()
	mustContain(t, "uninstall", out, "Uninstalled alpha@1.0.0")
	if _, err := os.Lstat(filepath.Join(installs, "v-1.0.0")); !os.IsNotExist(err) {
		t.Errorf("alpha@1.0.0 is still there: %v", err)
	}
	must(t, os.MkdirAll(left, 0755)) // the empty SDK directory went with the last version

	out = f.run(t, "uninstall", "alpha@v1.0.0").mustSucceed(t, "uninstall alpha@v1.0.0 again").output()
	mustContain(t, "uninstall again", out, "Deleted the files that were left of alpha@v1.0.0")
	if _, err := os.Lstat(left); !os.IsNotExist(err) {
		t.Errorf("%s is still there: %v", left, err)
	}
}

// Files that an earlier uninstall left go with the next one of the same version,
// whether another version is installed or not. Those of another version stay.
func TestUninstallDeletesTheFilesOfAnEarlierOne(t *testing.T) {
	for _, tc := range []struct {
		name      string
		installed []string
	}{
		{"another version is left", []string{"1.0.0", "2.0.0"}},
		{"no other version is left", []string{"2.0.0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCLIFixture(t, false)
			for _, v := range tc.installed {
				f.installOnDisk(t, "alpha", v)
			}
			keepRegistry(t)
			installs := filepath.Join(f.shared, "cache", "alpha")
			for _, name := range []string{".v-2.0.0.removing", ".v-2.0.0.removing.1", ".v-2.0.0-rc1.removing"} {
				must(t, os.MkdirAll(filepath.Join(installs, name), 0755))
				must(t, os.WriteFile(filepath.Join(installs, name, "file"), nil, 0644))
			}

			out := f.run(t, "uninstall", "alpha@2.0.0").mustSucceed(t, "uninstall alpha@2.0.0").output()
			mustContain(t, "uninstall", out, "Uninstalled alpha@2.0.0 successfully!")
			for _, name := range []string{"v-2.0.0", ".v-2.0.0.removing", ".v-2.0.0.removing.1", ".v-2.0.0.removing.2"} {
				if _, err := os.Lstat(filepath.Join(installs, name)); !os.IsNotExist(err) {
					t.Errorf("%s is still there: %v", name, err)
				}
			}
			if len(tc.installed) > 1 {
				if _, err := os.Lstat(filepath.Join(installs, ".v-2.0.0-rc1.removing", "file")); err != nil {
					t.Errorf("the files of another version are gone: %v", err)
				}
			}
			out = f.run(t, "uninstall", "alpha@2.0.0").mustFail(t, "uninstall alpha@2.0.0 again").output()
			mustContain(t, "uninstall again", out, "not installed")
		})
	}
}

// Files that an earlier uninstall left and that still cannot be deleted are
// reported, and the version is gone all the same.
func TestUninstallReportsTheFilesOfAnEarlierOne(t *testing.T) {
	skipUnlessPermissionsHold(t)
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0")
	f.installOnDisk(t, "alpha", "2.0.0")
	installs := filepath.Join(f.shared, "cache", "alpha")
	left := filepath.Join(installs, ".v-2.0.0.removing")
	must(t, os.MkdirAll(left, 0755))
	must(t, os.WriteFile(filepath.Join(left, "file"), nil, 0644))
	must(t, os.Chmod(left, 0555))
	t.Cleanup(func() { _ = os.Chmod(left, 0755) })

	out := f.run(t, "uninstall", "alpha@2.0.0").mustFail(t, "uninstall alpha@2.0.0").output()
	mustContain(t, "uninstall", out, "alpha@2.0.0 is no longer installed", "could not delete all files of alpha@2.0.0", left)
	mustNotContain(t, "uninstall", out, "successfully")
	for _, name := range []string{"v-2.0.0", ".v-2.0.0.removing.1"} {
		if _, err := os.Lstat(filepath.Join(installs, name)); !os.IsNotExist(err) {
			t.Errorf("%s is still there: %v", name, err)
		}
	}

	must(t, os.Chmod(left, 0755))
	out = f.run(t, "uninstall", "alpha@2.0.0").mustSucceed(t, "uninstall alpha@2.0.0 again").output()
	mustContain(t, "uninstall again", out, "Deleted the files that were left of alpha@2.0.0")
	if _, err := os.Lstat(left); !os.IsNotExist(err) {
		t.Errorf("%s is still there: %v", left, err)
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
	mustContain(t, "uninstall", out, "EnvKeys exploded", "beta@1.0.0")
	mustNotContain(t, "uninstall", out, "Auto switch")
	mustContain(t, "global config", readFile(t, globalConfig), `beta = "2.0.0"`)
	if _, err := os.Stat(filepath.Join(f.shared, "cache", "beta", "v-2.0.0")); err != nil {
		t.Errorf("beta@2.0.0 was removed: %v", err)
	}
}

// A leading "v" is not part of the order: v22.0.0 is newer than v20.1.0, which is
// newer than 18.0.0. The names stay as they were installed.
func TestUninstallOrdersVersionsWithALeadingV(t *testing.T) {
	f := newCLIFixture(t, false)
	for _, v := range []string{"18.0.0", "v20.1.0", "v22.0.0"} {
		f.installOnDisk(t, "alpha", v)
	}
	globalConfig := filepath.Join(f.userHome, ".vfox.toml")
	must(t, os.WriteFile(globalConfig, []byte("[tools]\nalpha = \"v22.0.0\"\n"), 0644))
	keepRegistry(t)

	list := f.run(t, "list", "alpha").mustSucceed(t, "list alpha").output()
	if i22, i20, i18 := strings.Index(list, "v22.0.0"), strings.Index(list, "v20.1.0"), strings.Index(list, "18.0.0"); !(i22 >= 0 && i22 < i20 && i20 < i18) {
		t.Errorf("list alpha shows the versions out of order, want v22.0.0, v20.1.0, 18.0.0:\n%s", list)
	}
	f.run(t, "uninstall", "alpha@v22.0.0").mustSucceed(t, "uninstall alpha@v22.0.0")
	mustContain(t, "global config", readFile(t, globalConfig), `alpha = "v20.1.0"`)
}

// A pre-release is older than its release, so the release takes over.
func TestUninstallPrefersAReleaseToItsPreRelease(t *testing.T) {
	f := newCLIFixture(t, false)
	for _, v := range []string{"1.0.0", "2.0.0", "2.0.0-rc1", "3.0.0"} {
		f.installOnDisk(t, "alpha", v)
	}
	globalConfig := filepath.Join(f.userHome, ".vfox.toml")
	must(t, os.WriteFile(globalConfig, []byte("[tools]\nalpha = \"3.0.0\"\n"), 0644))
	keepRegistry(t)

	f.run(t, "uninstall", "alpha@3.0.0").mustSucceed(t, "uninstall alpha@3.0.0")
	mustContain(t, "global config", readFile(t, globalConfig), `alpha = "2.0.0"`)
}

// Uninstalling a global version that is not installed fails without changing
// the global version.
func TestUninstallOfAVersionNotInstalledKeepsTheGlobalVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0")
	globalConfig := filepath.Join(f.userHome, ".vfox.toml")
	must(t, os.WriteFile(globalConfig, []byte("[tools]\nalpha = \"2.0.0\"\n"), 0644))
	keepRegistry(t)

	out := f.run(t, "uninstall", "alpha@2.0.0").mustFail(t, "uninstall alpha@2.0.0").output()
	mustContain(t, "uninstall", out, "not installed")
	mustNotContain(t, "uninstall", out, "Auto switch")
	mustContain(t, "global config", readFile(t, globalConfig), `alpha = "2.0.0"`)
}

// deltaPlugin refuses to uninstall anything.
const deltaPlugin = `PLUGIN = {name="delta", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
function PLUGIN:PreUninstall(ctx) error("PreUninstall refused") end
`

// When the plugin's PreUninstall hook fails, nothing is removed and the global
// version stays as it was.
func TestUninstallKeepsTheGlobalVersionWhenPreUninstallFails(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "delta", deltaPlugin)
	for _, v := range []string{"1.0.0", "2.0.0"} {
		f.installOnDisk(t, "delta", v)
	}
	globalConfig := filepath.Join(f.userHome, ".vfox.toml")
	must(t, os.WriteFile(globalConfig, []byte("[tools]\ndelta = \"2.0.0\"\n"), 0644))
	keepRegistry(t)

	out := f.run(t, "uninstall", "delta@2.0.0").mustFail(t, "uninstall delta@2.0.0").output()
	mustContain(t, "uninstall", out, "PreUninstall refused")
	mustNotContain(t, "uninstall", out, "Auto switch")
	mustContain(t, "global config", readFile(t, globalConfig), `delta = "2.0.0"`)
}

// zetaPlugin fails to give the environment of version 2.0.0.
const zetaPlugin = `PLUGIN = {name="zeta", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx)
  if ctx.main.version == "2.0.0" then error("EnvKeys exploded") end
  return {{key="PATH", value=ctx.path .. "/bin"}}
end
`

// When the newest version left cannot be used, the next one takes over, and the
// command says which one it skipped and why. One it did not get to is not named.
func TestUninstallTriesTheNextVersionWhenTheSwitchFails(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "zeta", zetaPlugin)
	for _, v := range []string{"0.9.0", "1.0.0", "2.0.0", "3.0.0"} {
		f.installOnDisk(t, "zeta", v)
	}
	globalConfig := filepath.Join(f.userHome, ".vfox.toml")
	must(t, os.WriteFile(globalConfig, []byte("[tools]\nzeta = \"3.0.0\"\n"), 0644))
	keepRegistry(t)

	out := f.run(t, "uninstall", "zeta@3.0.0").mustSucceed(t, "uninstall zeta@3.0.0").output()
	mustContain(t, "uninstall", out, "Auto switch", "zeta@1.0.0", "Could not switch to zeta@2.0.0", "EnvKeys exploded")
	mustNotContain(t, "uninstall", out, "zeta@0.9.0")
	mustContain(t, "global config", readFile(t, globalConfig), `zeta = "1.0.0"`)
	if _, err := os.Stat(filepath.Join(f.shared, "cache", "zeta", "v-3.0.0")); !os.IsNotExist(err) {
		t.Errorf("zeta@3.0.0 is still there: %v", err)
	}
}

// The global links of additions that only the removed version had are removed
// with it. The link of the main runtime stays: it points at the version that
// took over.
func TestUninstallRemovesTheLinksOfAdditionsTheNextVersionLacks(t *testing.T) {
	f := newCLIFixture(t, false)
	for _, v := range []string{"1.0.0", "2.0.0"} {
		f.installOnDisk(t, "alpha", v)
	}
	must(t, os.MkdirAll(filepath.Join(f.shared, "cache", "alpha", "v-1.0.0", "extra-1.0.0"), 0755))
	keepRegistry(t)
	f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use -g alpha@1.0.0")
	extra := filepath.Join(f.userHome, "sdks", "extra")
	if _, err := os.Stat(extra); err != nil {
		t.Fatalf("use -g did not link the addition: %v", err)
	}

	f.run(t, "uninstall", "alpha@1.0.0").mustSucceed(t, "uninstall alpha@1.0.0")
	if _, err := os.Lstat(extra); !os.IsNotExist(err) {
		t.Errorf("the link of the addition is still there: %v", err)
	}
	if got := env.IsDirSymlink(filepath.Join(f.userHome, "sdks", "alpha")); !got {
		t.Error("the link of alpha is gone")
	}
	if _, err := os.Stat(filepath.Join(f.userHome, "sdks", "alpha")); err != nil {
		t.Errorf("the link of alpha is broken: %v", err)
	}
}

// When the links cannot be moved to the version that would take over, the config
// keeps the version the links still point at.
func TestUninstallKeepsTheConfigWhenTheLinksCannotBeMoved(t *testing.T) {
	skipUnlessPermissionsHold(t)
	f := newCLIFixture(t, false)
	for _, v := range []string{"1.0.0", "2.0.0"} {
		f.installOnDisk(t, "alpha", v)
	}
	f.run(t, "use", "-g", "alpha@2.0.0").mustSucceed(t, "use -g alpha@2.0.0")
	sdks := filepath.Join(f.userHome, "sdks")
	unlock := lock(t, sdks)

	out := f.run(t, "uninstall", "alpha@2.0.0").mustFail(t, "uninstall alpha@2.0.0").output()
	mustContain(t, "uninstall", out, "nothing was removed")
	mustContain(t, "global config", readFile(t, filepath.Join(f.userHome, ".vfox.toml")), `alpha = "2.0.0"`)
	target, err := os.Readlink(filepath.Join(sdks, "alpha"))
	must(t, err)
	mustContain(t, "link of alpha", target, "v-2.0.0")

	unlock()
	f.run(t, "uninstall", "alpha@2.0.0").mustSucceed(t, "uninstall alpha@2.0.0 again")
	mustContain(t, "global config", readFile(t, filepath.Join(f.userHome, ".vfox.toml")), `alpha = "1.0.0"`)
}

// Uninstalling the only version unsets it once it is gone.
func TestUninstallUnsetsTheOnlyVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "2.0.0")
	keepRegistry(t)
	f.run(t, "use", "-g", "alpha@2.0.0").mustSucceed(t, "use -g alpha@2.0.0")

	out := f.run(t, "uninstall", "alpha@2.0.0").mustSucceed(t, "uninstall alpha@2.0.0").output()
	mustContain(t, "uninstall", out, "Uninstalled", "Unset")
	mustNotContain(t, "global config", readFile(t, filepath.Join(f.userHome, ".vfox.toml")), "alpha")
	if env.IsDirSymlink(filepath.Join(f.userHome, "sdks", "alpha")) {
		t.Error("the global link of alpha is still there")
	}
}

// Uninstalling a version that is not global removes the global links that still
// point into it, and leaves those of the global version alone.
func TestUninstallRemovesTheGlobalLinksOfAVersionThatIsNotGlobal(t *testing.T) {
	f := newCLIFixture(t, false)
	for _, v := range []string{"1.0.0", "2.0.0"} {
		f.installOnDisk(t, "alpha", v)
	}
	must(t, os.MkdirAll(filepath.Join(f.shared, "cache", "alpha", "v-1.0.0", "extra-1.0.0"), 0755))
	keepRegistry(t)
	f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use -g alpha@1.0.0")
	f.run(t, "use", "-g", "alpha@2.0.0").mustSucceed(t, "use -g alpha@2.0.0")

	f.run(t, "uninstall", "alpha@1.0.0").mustSucceed(t, "uninstall alpha@1.0.0")
	if _, err := os.Lstat(filepath.Join(f.userHome, "sdks", "extra")); !os.IsNotExist(err) {
		t.Errorf("the link of the addition is still there: %v", err)
	}
	if _, err := os.Stat(filepath.Join(f.userHome, "sdks", "alpha")); err != nil {
		t.Errorf("the link of alpha is broken: %v", err)
	}
	mustContain(t, "global config", readFile(t, filepath.Join(f.userHome, ".vfox.toml")), `alpha = "2.0.0"`)
}

// A link that cannot be removed does not fail the uninstall, but it is reported.
func TestUninstallWarnsWhenALinkCannotBeRemoved(t *testing.T) {
	skipUnlessPermissionsHold(t)
	f := newCLIFixture(t, false)
	for _, v := range []string{"1.0.0", "2.0.0"} {
		f.installOnDisk(t, "alpha", v)
	}
	must(t, os.MkdirAll(filepath.Join(f.shared, "cache", "alpha", "v-1.0.0", "extra-1.0.0"), 0755))
	f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use -g alpha@1.0.0")
	f.run(t, "use", "-g", "alpha@2.0.0").mustSucceed(t, "use -g alpha@2.0.0")
	sdks := filepath.Join(f.userHome, "sdks")
	must(t, os.Chmod(sdks, 0555))
	t.Cleanup(func() { _ = os.Chmod(sdks, 0755) })

	out := f.run(t, "uninstall", "alpha@1.0.0").mustSucceed(t, "uninstall alpha@1.0.0").output()
	mustContain(t, "uninstall", out, "Uninstalled", "could not be removed", "extra")
}

// When the global setting cannot be changed, nothing is removed, and a retry
// works once it can.
func TestUninstallChangesNothingWhenTheConfigCannotBeWritten(t *testing.T) {
	skipUnlessPermissionsHold(t)
	for _, tc := range []struct {
		name      string
		installed []string
	}{
		{"another version is left", []string{"1.0.0", "2.0.0"}},
		{"no other version is left", []string{"2.0.0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newCLIFixture(t, false)
			for _, v := range tc.installed {
				f.installOnDisk(t, "alpha", v)
			}
			keepRegistry(t)
			f.run(t, "use", "-g", "alpha@2.0.0").mustSucceed(t, "use -g alpha@2.0.0")
			f.run(t, "alias", "set", "alpha", "mine", "2.0.0").mustSucceed(t, "alias set")
			globalConfig := filepath.Join(f.userHome, ".vfox.toml")
			unlock := lock(t, globalConfig)

			out := f.run(t, "uninstall", "alpha@2.0.0").mustFail(t, "uninstall alpha@2.0.0").output()
			mustContain(t, "uninstall", out, "nothing was removed")
			mustNotContain(t, "uninstall", out, "Removed the alias(es)", "Auto switch", "Unset")
			mustContain(t, "global config", readFile(t, globalConfig), `alpha = "2.0.0"`)
			if _, err := os.Stat(filepath.Join(f.userHome, "sdks", "alpha")); err != nil {
				t.Errorf("the link of alpha is broken: %v", err)
			}

			unlock()
			out = f.run(t, "uninstall", "alpha@2.0.0").mustSucceed(t, "uninstall alpha@2.0.0 again").output()
			mustContain(t, "uninstall again", out, "Uninstalled", "Removed the alias(es) mine")
			mustNotContain(t, "global config", readFile(t, globalConfig), "2.0.0")
		})
	}
}
