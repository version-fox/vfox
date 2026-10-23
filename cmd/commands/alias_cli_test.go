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
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/sdk"
)

func TestAliasCommandEndToEnd(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "17.0.9-tem", "alphatool")
	f.installOnDisk(t, "alpha", "21.0.1-graal", "alphatool")

	out := f.run(t, "alias", "list").mustSucceed(t, "alias list").output()
	mustContain(t, "alias list", out, "No aliases defined", "vfox alias set")

	out = f.run(t, "alias", "set", "alpha", "j17", "17.0.9-tem").mustSucceed(t, "alias set").output()
	mustContain(t, "alias set", out, "alpha@j17", "alpha@17.0.9-tem")
	f.run(t, "alias", "set", "alpha", "j21", "v21.0.1-graal").mustSucceed(t, "alias set with the leading v that `vfox current` prints")

	t.Run("they are listed", func(t *testing.T) {
		out := f.run(t, "alias", "list").mustSucceed(t, "alias list").stdout
		if want := "alpha@j17 -> alpha@17.0.9-tem\nalpha@j21 -> alpha@21.0.1-graal\n"; out != want {
			t.Errorf("alias list = %q, want %q", out, want)
		}
		out = f.run(t, "alias", "ls", "plain").mustSucceed(t, "alias ls plain").stdout
		mustContain(t, "alias ls plain", out, "No aliases defined")
	})

	t.Run("list shows them next to the version", func(t *testing.T) {
		out := f.run(t, "list", "alpha").mustSucceed(t, "list").output()
		mustContain(t, "list", out, "17.0.9-tem [alias: j17]", "21.0.1-graal [alias: j21]")
	})

	t.Run("the commands that take a version understand them", func(t *testing.T) {
		info := parseInfo(t, f.run(t, "info", "--json", "alpha@j17").mustSucceed(t, "info alpha@j17"))
		if !info.Installed || info.Version != "17.0.9-tem" {
			t.Errorf("info = %+v, want the version the alias names", info)
		}
		out := f.run(t, "use", "-g", "alpha@j21").mustSucceed(t, "use alpha@j21").output()
		mustContain(t, "use", out, "Now using", "alpha@21.0.1-graal")
		// What lands in the version file is the real version, never the alias.
		config := readFile(t, filepath.Join(f.userHome, ".vfox.toml"))
		mustContain(t, "global config", config, `alpha = "21.0.1-graal"`)
		mustNotContain(t, "global config", config, "j21")
	})

	t.Run("they are written to the user config", func(t *testing.T) {
		mustContain(t, "config.yaml", readFile(t, filepath.Join(f.userHome, "config.yaml")), "aliases:", "j17: 17.0.9-tem", "j21: 21.0.1-graal")
	})

	t.Run("uninstalling a version takes its aliases along", func(t *testing.T) {
		out := f.run(t, "uninstall", "alpha@j17").mustSucceed(t, "uninstall alpha@j17").output()
		mustContain(t, "uninstall", out, "Uninstalled", "alpha@17.0.9-tem", "Removed the alias(es) j17")
		if exists(filepath.Join(f.shared, "cache", "alpha", "v-17.0.9-tem")) {
			t.Error("the version was not uninstalled")
		}
		out = f.run(t, "alias", "list").mustSucceed(t, "alias list").stdout
		mustNotContain(t, "alias list", out, "j17")
		mustContain(t, "alias list", out, "alpha@j21")
	})

	t.Run("unset", func(t *testing.T) {
		f.run(t, "alias", "unset", "alpha", "j21").mustSucceed(t, "alias unset")
		mustContain(t, "alias list", f.run(t, "alias", "list").mustSucceed(t, "alias list").output(), "No aliases defined")
		f.run(t, "alias", "unset", "alpha", "j21").mustFail(t, "unset of an alias that is gone")
	})
}

func TestAliasCommandRefusesWhatWouldBeConfusing(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	f.installOnDisk(t, "alpha", "2.0.0", "alphatool")

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"the name of an installed version", []string{"alias", "set", "alpha", "2.0.0", "1.0.0"}, "installed version"},
		{"a version that is not installed", []string{"alias", "set", "alpha", "old", "0.9.0"}, "not installed"},
		{"latest", []string{"alias", "set", "alpha", "latest", "1.0.0"}, "not a usable alias"},
		{"a name with a slash", []string{"alias", "set", "alpha", "a/b", "1.0.0"}, "not a usable alias"},
		{"an SDK without a plugin", []string{"alias", "set", "nosuch", "x", "1.0.0"}, "not supported"},
		{"wrong number of arguments", []string{"alias", "set", "alpha", "x"}, "usage: vfox alias set"},
		{"unset with the wrong arguments", []string{"alias", "unset", "alpha"}, "usage: vfox alias unset"},
		{"unset of an alias that does not exist", []string{"alias", "unset", "alpha", "nope"}, "has no alias nope"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := f.run(t, tc.args...).mustFail(t, strings.Join(tc.args, " "))
			mustContain(t, "error", r.output(), tc.want)
		})
	}
	if _, err := os.Stat(filepath.Join(f.userHome, "config.yaml")); err == nil {
		mustNotContain(t, "config.yaml", readFile(t, filepath.Join(f.userHome, "config.yaml")), "old", "latest")
	}
}

// An installed version beats an alias of the same name, so an alias cannot make
// a version unreachable.
func TestInstalledVersionBeatsAnAliasOfTheSameName(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	f.installOnDisk(t, "alpha", "2.0.0", "alphatool")
	f.run(t, "alias", "set", "alpha", "stable", "1.0.0").mustSucceed(t, "alias set")
	// The alias exists first; then a version appears under that very name.
	f.installOnDisk(t, "alpha", "stable", "alphatool")

	info := parseInfo(t, f.run(t, "info", "--json", "alpha@stable").mustSucceed(t, "info"))
	if info.Version != "stable" {
		t.Errorf("alpha@stable resolved to %q: the installed version of that name must win", info.Version)
	}
}

func TestAliasesFromTheSharedConfigAreUsedButNotEditedHere(t *testing.T) {
	f := newCLIFixture(t, true)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	f.installOnDisk(t, "alpha", "2.0.0", "alphatool")
	must(t, os.WriteFile(filepath.Join(f.shared, "config.yaml"), []byte("aliases:\n  alpha:\n    team: 2.0.0\n"), 0644))
	f.writeUserConfig(t, "aliases:\n  alpha:\n    mine: 1.0.0\n")

	out := f.run(t, "alias", "list").mustSucceed(t, "alias list").stdout
	mustContain(t, "alias list", out, "alpha@mine -> alpha@1.0.0", "alpha@team -> alpha@2.0.0")
	if info := parseInfo(t, f.run(t, "info", "--json", "alpha@team").mustSucceed(t, "info")); info.Version != "2.0.0" {
		t.Errorf("a shared alias resolved to %q", info.Version)
	}

	r := f.run(t, "alias", "unset", "alpha", "team").mustFail(t, "unset of a shared alias")
	mustContain(t, "error", r.output(), "shared config")

	// Changing the user's aliases does not copy the shared ones into the user's file.
	f.run(t, "alias", "set", "alpha", "other", "2.0.0").mustSucceed(t, "alias set")
	user := readFile(t, filepath.Join(f.userHome, "config.yaml"))
	mustContain(t, "user config", user, "mine: 1.0.0", "other: 2.0.0")
	mustNotContain(t, "user config", user, "team")
}

func TestDoctorReportsAnAliasForAVersionThatIsGone(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	f.writeUserConfig(t, "aliases:\n  alpha:\n    old: 0.5.0\n")

	r := f.run(t, "doctor").mustSucceed(t, "doctor")
	mustContain(t, "doctor", r.stdout, "warn   alias.dangling  alpha@0.5.0", "The alias old names alpha@0.5.0", "fix: vfox alias unset alpha old")
	f.run(t, "alias", "unset", "alpha", "old").mustSucceed(t, "alias unset")
	mustNotContain(t, "doctor", f.run(t, "doctor").mustSucceed(t, "doctor").stdout, "alias.dangling")
}

func TestListShowsTheAliasesOfAVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	f.run(t, "alias", "set", "alpha", "stable", "1.0.0").mustSucceed(t, "alias set")
	f.run(t, "alias", "set", "alpha", "j1", "1.0.0").mustSucceed(t, "alias set")

	mustContain(t, "list alpha", f.run(t, "list", "alpha").mustSucceed(t, "list alpha").output(), "1.0.0 [alias: j1, stable]")
	mustContain(t, "list", f.run(t, "list").mustSucceed(t, "list").output(), "1.0.0 [alias: j1, stable]")
}

func TestExecResolvesAliases(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	f.run(t, "alias", "set", "alpha", "stable", "1.0.0").mustSucceed(t, "alias set")
	manager := f.manager(t)

	for _, tt := range []struct {
		name, sdk string
		version   sdk.Version
		want      sdk.Version
	}{
		{"an alias", "alpha", "stable", "1.0.0"},
		{"a version", "alpha", "1.0.0", "1.0.0"},
		{"something that is neither", "alpha", "unknown", "unknown"},
		{"an SDK that is not there", "nosuch", "stable", "stable"},
	} {
		got, err := resolveExecSDKVersion(manager, execSDKSpec{Name: tt.sdk, Version: tt.version})
		if err != nil || got != tt.want {
			t.Errorf("%s: resolveExecSDKVersion(%s@%s) = (%q, %v), want %q", tt.name, tt.sdk, tt.version, got, err, tt.want)
		}
	}
}

// Changing an alias changes the aliases and nothing else: a section the user's
// file does not declare keeps following the shared config, so the settings an
// administrator put there must still apply afterwards.
func TestAliasChangesLeaveTheRestOfTheUserConfigAlone(t *testing.T) {
	f := newCLIFixture(t, true)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	// Settings that differ from the defaults, so that losing them shows.
	must(t, os.WriteFile(filepath.Join(f.shared, "config.yaml"), []byte("path:\n  mode: legacy\ndiagnostics:\n  onUse: false\n"), 0644))
	f.writeUserConfig(t, "aliases: {}\n")

	f.run(t, "alias", "set", "alpha", "stable", "1.0.0").mustSucceed(t, "alias set")
	f.run(t, "alias", "set", "alpha", "other", "1.0.0").mustSucceed(t, "alias set")
	f.run(t, "alias", "unset", "alpha", "other").mustSucceed(t, "alias unset")

	user := readFile(t, filepath.Join(f.userHome, "config.yaml"))
	mustContain(t, "user config", user, "stable: 1.0.0")
	mustNotContain(t, "user config", user, "path:", "diagnostics:", "plugin:", "proxy:", "storage:", "registry:", "cache:", "gitignore:", "other")

	// The next process, which loads both files.
	settings := f.manager(t).RuntimeEnvContext.UserConfig
	if mode := settings.Path.EffectiveMode(); mode != "legacy" {
		t.Errorf("path.mode = %q after an alias change, want the shared legacy", mode)
	}
	if settings.Diagnostics.OnUse {
		t.Error("diagnostics.onUse was switched back on by an alias change")
	}

	// Taking the aliases of an uninstalled version along does not copy anything either.
	f.run(t, "uninstall", "alpha@stable").mustSucceed(t, "uninstall")
	user = readFile(t, filepath.Join(f.userHome, "config.yaml"))
	mustNotContain(t, "user config", user, "path:", "diagnostics:", "stable")
}
