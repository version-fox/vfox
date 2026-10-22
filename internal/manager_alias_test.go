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

package internal

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/sdk"
)

// aliasFixture has the SDK alpha with the versions 1.0.0 and 2.0.0 installed.
func aliasFixture(t *testing.T) (*managerFixture, sdk.Sdk) {
	t.Helper()
	f := newManagerFixture(t)
	s := f.addSDK(t, "alpha")
	f.install(t, "alpha", "1.0.0")
	f.install(t, "alpha", "2.0.0")
	return f, s
}

// userAliases reads the aliases from the user's config file, not from memory.
func userAliases(t *testing.T, f *managerFixture) map[string]map[string]string {
	t.Helper()
	c, err := config.NewConfigWithPath(f.meta.User.Config)
	if err != nil {
		t.Fatal(err)
	}
	return c.Aliases
}

func TestAliasesFor(t *testing.T) {
	f := newManagerFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{Aliases: map[string]map[string]string{
		"alpha": {"stable": "1.0.0", "lts": "2.0.0"},
		"beta":  {"x": "9"},
	}}

	got := f.AliasesFor("Alpha") // SDK names are not case sensitive
	if want := map[string]string{"stable": "1.0.0", "lts": "2.0.0"}; !reflect.DeepEqual(got, want) {
		t.Errorf("AliasesFor = %v, want %v", got, want)
	}
	got["stable"] = "changed"
	if f.AliasesFor("alpha")["stable"] != "1.0.0" {
		t.Error("the caller got the configuration itself, not a copy")
	}
	if got := f.AliasesFor("gamma"); got == nil || len(got) != 0 {
		t.Errorf("AliasesFor an SDK without aliases = %v, want an empty map", got)
	}

	f.RuntimeEnvContext.UserConfig = nil
	if got := f.AliasesFor("alpha"); got == nil || len(got) != 0 {
		t.Errorf("without a config = %v, want an empty map", got)
	}
	f.RuntimeEnvContext = nil
	if got := f.AliasesFor("alpha"); got == nil || len(got) != 0 {
		t.Errorf("without a context = %v, want an empty map", got)
	}
}

func TestResolveAlias(t *testing.T) {
	f, s := aliasFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{Aliases: map[string]map[string]string{
		"alpha": {"stable": "1.0.0", "2.0.0": "1.0.0"}, // the second one hides a version
	}}

	for _, tt := range []struct {
		name    string
		sdk     sdk.Sdk
		version sdk.Version
		want    sdk.Version
	}{
		{"an alias", s, "stable", "1.0.0"},
		{"a version that is not an alias", s, "2.0.0-rc1", "2.0.0-rc1"},
		{"an installed version wins over an alias of the same name", s, "2.0.0", "2.0.0"},
		{"nothing to resolve", s, "", ""},
		{"an SDK that is not known still has aliases", nil, "stable", "1.0.0"},
	} {
		if got := f.ResolveAlias("alpha", tt.sdk, tt.version); got != tt.want {
			t.Errorf("%s: ResolveAlias(%q) = %q, want %q", tt.name, tt.version, got, tt.want)
		}
	}
}

func TestSetAlias(t *testing.T) {
	f, _ := aliasFixture(t)

	if err := f.SetAlias("Alpha", "stable", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetAlias("alpha", "lts", "2.0.0"); err != nil {
		t.Fatal(err)
	}
	if err := f.SetAlias("alpha", "stable", "2.0.0"); err != nil { // the same name can be pointed elsewhere
		t.Fatal(err)
	}
	want := map[string]map[string]string{"alpha": {"stable": "2.0.0", "lts": "2.0.0"}}
	if got := userAliases(t, f); !reflect.DeepEqual(got, want) {
		t.Errorf("user config aliases = %v, want %v", got, want)
	}
	// The process that made the change sees it without reading the file again.
	if got := f.AliasesFor("alpha"); !reflect.DeepEqual(got, want["alpha"]) {
		t.Errorf("AliasesFor = %v, want %v", got, want["alpha"])
	}
}

func TestSetAliasRefusesWhatWouldBeConfusing(t *testing.T) {
	f, _ := aliasFixture(t)
	for _, tt := range []struct {
		name, sdk, alias, version, want string
	}{
		{"an SDK without a plugin", "nosuch", "x", "1.0.0", "not supported"},
		{"a name that cannot be a version name", "alpha", "a/b", "1.0.0", "not a usable alias"},
		{"the reserved name", "alpha", "latest", "1.0.0", "not a usable alias"},
		{"the name of an installed version", "alpha", "2.0.0", "1.0.0", "installed version"},
		{"a version that is not installed", "alpha", "old", "0.9.0", "not installed"},
	} {
		err := f.SetAlias(tt.sdk, tt.alias, sdk.Version(tt.version))
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want it to contain %q", tt.name, err, tt.want)
		}
	}
	if _, err := os.Stat(f.meta.User.Config); err == nil {
		t.Error("a refused alias must not write the config file")
	}
}

func TestUnsetAlias(t *testing.T) {
	f, _ := aliasFixture(t)
	for _, alias := range []string{"stable", "lts"} {
		if err := f.SetAlias("alpha", alias, "1.0.0"); err != nil {
			t.Fatal(err)
		}
	}

	if err := f.UnsetAlias("Alpha", "stable"); err != nil {
		t.Fatal(err)
	}
	if got := userAliases(t, f); !reflect.DeepEqual(got, map[string]map[string]string{"alpha": {"lts": "1.0.0"}}) {
		t.Errorf("aliases = %v", got)
	}
	if err := f.UnsetAlias("alpha", "lts"); err != nil {
		t.Fatal(err)
	}
	if got := userAliases(t, f); len(got) != 0 {
		t.Errorf("aliases = %v, want the SDK dropped once it has none", got)
	}

	if err := f.UnsetAlias("alpha", "stable"); err == nil || !strings.Contains(err.Error(), "has no alias stable") {
		t.Errorf("err = %v, want it to say there is no such alias", err)
	}
}

// What the shared config defines is not the user's to remove, and is never copied into the user's file.
func TestAliasesOfTheSharedConfigStayOutOfTheUserFile(t *testing.T) {
	f, _ := aliasFixture(t)
	// The loaded config is the shared and the user's combined.
	f.RuntimeEnvContext.UserConfig = &config.Config{Aliases: map[string]map[string]string{"alpha": {"team": "1.0.0"}}}

	err := f.UnsetAlias("alpha", "team")
	if err == nil || !strings.Contains(err.Error(), "shared config") {
		t.Errorf("err = %v, want it to say the alias comes from the shared config", err)
	}

	if err := f.SetAlias("alpha", "mine", "2.0.0"); err != nil {
		t.Fatal(err)
	}
	if got := userAliases(t, f); !reflect.DeepEqual(got, map[string]map[string]string{"alpha": {"mine": "2.0.0"}}) {
		t.Errorf("user config aliases = %v, want only the user's own", got)
	}
	if got := f.AliasesFor("alpha"); !reflect.DeepEqual(got, map[string]string{"team": "1.0.0", "mine": "2.0.0"}) {
		t.Errorf("AliasesFor = %v, want both", got)
	}
}

func TestRemoveAliasesFor(t *testing.T) {
	f, _ := aliasFixture(t)
	for alias, version := range map[string]sdk.Version{"stable": "1.0.0", "j1": "1.0.0", "lts": "2.0.0"} {
		if err := f.SetAlias("alpha", alias, version); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := f.RemoveAliasesFor("Alpha", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"j1", "stable"}; !reflect.DeepEqual(removed, want) {
		t.Errorf("removed = %v, want %v", removed, want)
	}
	if got := userAliases(t, f); !reflect.DeepEqual(got, map[string]map[string]string{"alpha": {"lts": "2.0.0"}}) {
		t.Errorf("aliases = %v, want only the one for the other version", got)
	}
	if got := f.AliasesFor("alpha"); !reflect.DeepEqual(got, map[string]string{"lts": "2.0.0"}) {
		t.Errorf("AliasesFor = %v", got)
	}

	removed, err = f.RemoveAliasesFor("alpha", "1.0.0")
	if err != nil || len(removed) != 0 {
		t.Errorf("a version without aliases: removed = %v, err = %v", removed, err)
	}
}

func TestAliasChangesReportAConfigTheyCannotUse(t *testing.T) {
	t.Run("a config file that cannot be read", func(t *testing.T) {
		f, _ := aliasFixture(t)
		if err := os.WriteFile(f.meta.User.Config, []byte("aliases: [not, a, map]\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := f.SetAlias("alpha", "stable", "1.0.0"); err == nil || !strings.Contains(err.Error(), "load user config") {
			t.Errorf("err = %v, want it to say the config could not be loaded", err)
		}
		if got, err := os.ReadFile(f.meta.User.Config); err != nil || string(got) != "aliases: [not, a, map]\n" {
			t.Errorf("the broken file was changed: %q, %v", got, err)
		}
	})

	t.Run("a config file that cannot be written", func(t *testing.T) {
		f, _ := aliasFixture(t)
		// There is no directory for the file to go to.
		if err := os.RemoveAll(f.meta.User.Home); err != nil {
			t.Fatal(err)
		}
		if err := f.SetAlias("alpha", "stable", "1.0.0"); err == nil || !strings.Contains(err.Error(), "save user config") {
			t.Errorf("err = %v, want it to say the config could not be saved", err)
		}
		if got := f.AliasesFor("alpha"); len(got) != 0 {
			t.Errorf("AliasesFor = %v, want the failed change not to show in memory", got)
		}
	})
}

func TestAliasFindings(t *testing.T) {
	f, s := aliasFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{Aliases: map[string]map[string]string{
		"alpha": {"stable": "1.0.0", "gone": "0.9.0", "old": "0.1.0"},
	}}

	findings := f.aliasFindings(s)
	if len(findings) != 2 {
		t.Fatalf("findings = %+v, want the two aliases whose version is not installed", findings)
	}
	// In the order of the alias names.
	gone, old := findings[0], findings[1]
	if gone.ID != sdk.FindingAliasDangling || gone.SDK != "alpha" || gone.Version != "0.9.0" || gone.Severity != sdk.SeverityWarn ||
		!strings.Contains(gone.Message, "gone") || !reflect.DeepEqual(gone.Fix, []string{"vfox alias unset alpha gone"}) {
		t.Errorf("finding = %+v", gone)
	}
	if old.Version != "0.1.0" || !reflect.DeepEqual(old.Fix, []string{"vfox alias unset alpha old"}) {
		t.Errorf("finding = %+v", old)
	}

	// `doctor` for the whole SDK reports them; for one version they are not about it.
	all, err := f.Diagnose("alpha", "")
	if err != nil {
		t.Fatal(err)
	}
	if findFinding(all, sdk.FindingAliasDangling, "alpha") == nil {
		t.Errorf("Diagnose lacks the dangling aliases: %+v", all)
	}
	one, err := f.Diagnose("alpha", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if findFinding(one, sdk.FindingAliasDangling, "alpha") != nil {
		t.Errorf("a version was asked for, the aliases of other versions are not its business: %+v", one)
	}
}

// The process may have loaded its config before the alias was written to the file.
func TestUnsetAliasThatTheLoadedConfigDoesNotKnow(t *testing.T) {
	f, _ := aliasFixture(t)
	if err := f.SetAlias("alpha", "stable", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	f.RuntimeEnvContext.UserConfig = &config.Config{}

	if err := f.UnsetAlias("alpha", "stable"); err != nil {
		t.Fatalf("the file has the alias, so it can be removed: %v", err)
	}
	if got := userAliases(t, f); len(got) != 0 {
		t.Errorf("user config aliases = %v, want none", got)
	}
}

func TestResolveVersionName(t *testing.T) {
	f, s := aliasFixture(t)
	f.install(t, "alpha", "vendor") // what `vfox link alpha@vendor` leaves behind
	f.RuntimeEnvContext.UserConfig = &config.Config{Aliases: map[string]map[string]string{
		"alpha": {"v1": "2.0.0", "stable": "1.0.0", "4": "2.0.0", "v3": "1.0.0"},
	}}
	f.install(t, "alpha", "v3") // an installed version beats an alias of the same name

	for _, tt := range []struct {
		name  string
		sdk   sdk.Sdk
		typed sdk.Version
		want  sdk.Version
	}{
		{"an alias that starts with a v is not the prefix 1", s, "v1", "2.0.0"},
		{"an installed version that starts with a v", s, "vendor", "vendor"},
		{"an installed version wins over an alias", s, "v3", "v3"},
		{"the usual v", s, "v1.0.0", "1.0.0"},
		{"no v", s, "1.0.0", "1.0.0"},
		{"an alias", s, "stable", "1.0.0"},
		{"the alias of the name without the v", s, "v4", "2.0.0"},
		{"a v on something that is not installed", s, "v9.9.9", "9.9.9"},
		{"nothing typed", s, "", ""},
		{"an SDK that is not known still has aliases", nil, "v1", "2.0.0"},
		{"an SDK that is not known", nil, "v9", "9"},
	} {
		if got := f.ResolveVersionName("alpha", tt.sdk, tt.typed); got != tt.want {
			t.Errorf("%s: ResolveVersionName(%q) = %q, want %q", tt.name, tt.typed, got, tt.want)
		}
	}
}

// Only the aliases are written. What the user's file does not declare stays out
// of it, so that it keeps following the shared config.
func TestAliasChangesWriteOnlyTheAliases(t *testing.T) {
	f, _ := aliasFixture(t)
	if err := os.WriteFile(f.meta.User.Config, []byte("# mine\ncache:\n    availableHookDuration: 1h\naliases: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	steps := []func() error{
		func() error { return f.SetAlias("alpha", "stable", "1.0.0") },
		func() error { return f.SetAlias("alpha", "lts", "2.0.0") },
		func() error { return f.UnsetAlias("alpha", "lts") },
		func() error { _, err := f.RemoveAliasesFor("alpha", "1.0.0"); return err },
	}
	for i, step := range steps {
		if err := step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		content, err := os.ReadFile(f.meta.User.Config)
		if err != nil {
			t.Fatal(err)
		}
		saved := string(content)
		for _, want := range []string{"# mine", "cache:", "availableHookDuration: 1h"} {
			if !strings.Contains(saved, want) {
				t.Errorf("step %d lost %q:\n%s", i, want, saved)
			}
		}
		for _, unwanted := range []string{"path:", "diagnostics:", "proxy:", "plugin:", "storage:", "registry:", "gitignore:", "legacyVersionFile:"} {
			if strings.Contains(saved, unwanted) {
				t.Errorf("step %d declared %q in the user's file:\n%s", i, unwanted, saved)
			}
		}
	}
}
