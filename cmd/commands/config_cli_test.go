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
)

// The first run creates the user's config file. It must not pin the settings
// that a shared VFOX_HOME decides, or they stop applying from the second run on.
func TestAFreshUserFollowsTheSharedConfig(t *testing.T) {
	f := newCLIFixture(t, true)
	must(t, os.WriteFile(filepath.Join(f.shared, "config.yaml"), []byte("path:\n  mode: baseline\ndiagnostics:\n  onUse: false\n"), 0644))
	userConfig := filepath.Join(f.userHome, "config.yaml")
	if exists(userConfig) {
		t.Fatal("the user has a config file already: this is not a first run")
	}

	for run := 1; run <= 3; run++ {
		f.run(t, "list").mustSucceed(t, "list")
		settings := f.manager(t).RuntimeEnvContext.UserConfig
		if mode := settings.Path.EffectiveMode(); mode != "baseline" {
			t.Errorf("run %d: path.mode = %q, want the shared baseline", run, mode)
		}
		if settings.Diagnostics.OnUse {
			t.Errorf("run %d: diagnostics.onUse is on, but the shared config switched it off", run)
		}
	}

	user := readFile(t, userConfig)
	mustContain(t, "user config", user, "proxy:", "cache:", "legacyVersionFile:")
	mustNotContain(t, "user config", user, "path:", "diagnostics:", "aliases:")
}

// `vfox config` changes the user's own file, and in it only the key it sets. The
// merged config it lists holds the defaults and the shared values too, and
// saving that would turn them into the user's own: the user's sections replace
// the shared ones, so a later change of the shared config would not apply.
func TestConfigWritesOnlyTheKeyItSets(t *testing.T) {
	f := newCLIFixture(t, true)
	sharedConfig := filepath.Join(f.shared, "config.yaml")
	userConfig := filepath.Join(f.userHome, "config.yaml")
	must(t, os.WriteFile(sharedConfig, []byte(`path:
  mode: baseline
  preserve: [/opt/corp/bin]
diagnostics:
  onUse: false
aliases:
  alpha:
    team: 1.0.0
cache:
  availableHookDuration: 24h
`), 0644))
	f.writeUserConfig(t, "# my vfox settings\nproxy:\n    url: http://proxy.example.com:8080 # the corporate one\n    enable: true\n")

	f.run(t, "config", "gitignore.enable", "false").mustSucceed(t, "config gitignore.enable false")
	f.run(t, "config", "plugin.http.timeout", "45s").mustSucceed(t, "config plugin.http.timeout 45s")

	user := readFile(t, userConfig)
	mustContain(t, "user config", user, "# my vfox settings", "# the corporate one", "url: http://proxy.example.com:8080",
		"enable: true", "gitignore:", "enable: false", "timeout: 45s")
	mustNotContain(t, "user config", user, "path:", "diagnostics:", "aliases:", "cache:", "legacyVersionFile:", "registry:", "storage:")

	// What the shared config decides still applies,
	list := f.run(t, "config", "--list").mustSucceed(t, "config --list").stdout
	mustContain(t, "config --list", list,
		"path.mode = baseline", "path.preserve = [/opt/corp/bin]", "diagnostics.onUse = false",
		"aliases = map[alpha:map[team:1.0.0]]", "cache.availableHookDuration = 24h",
		"gitignore.enable = false", "plugin.http.timeout = 45s", "proxy.url = http://proxy.example.com:8080")

	// and so does a later change of it.
	must(t, os.WriteFile(sharedConfig, []byte("path:\n  mode: aggressive\ndiagnostics:\n  onUse: true\ncache:\n  availableHookDuration: 1h\n"), 0644))
	list = f.run(t, "config", "--list").mustSucceed(t, "config --list").stdout
	mustContain(t, "config --list after the shared config changed", list,
		"path.mode = aggressive", "diagnostics.onUse = true", "cache.availableHookDuration = 1h", "gitignore.enable = false")
}

// A user who has no config file yet gets the usual first-run file, and then
// only the key that was set.
func TestAFreshUserWhoSetsOneKeyStillFollowsTheSharedConfig(t *testing.T) {
	f := newCLIFixture(t, true)
	must(t, os.WriteFile(filepath.Join(f.shared, "config.yaml"), []byte("path:\n  mode: baseline\ndiagnostics:\n  onUse: false\naliases:\n  alpha:\n    team: 1.0.0\n"), 0644))
	userConfig := filepath.Join(f.userHome, "config.yaml")
	if exists(userConfig) {
		t.Fatal("the user has a config file already: this is not a first run")
	}

	f.run(t, "config", "gitignore.enable", "false").mustSucceed(t, "config gitignore.enable false")

	user := readFile(t, userConfig)
	mustContain(t, "user config", user, "gitignore:", "enable: false", "proxy:", "cache:")
	mustNotContain(t, "user config", user, "path:", "diagnostics:", "aliases:")
	list := f.run(t, "config", "--list").mustSucceed(t, "config --list").stdout
	mustContain(t, "config --list", list, "path.mode = baseline", "diagnostics.onUse = false", "aliases = map[alpha:map[team:1.0.0]]", "gitignore.enable = false")
}

// --unset takes the key out of the file. Writing the default instead would pin
// it over the shared value.
func TestConfigUnsetRemovesTheKey(t *testing.T) {
	f := newCLIFixture(t, true)
	userConfig := filepath.Join(f.userHome, "config.yaml")
	must(t, os.WriteFile(filepath.Join(f.shared, "config.yaml"), []byte("path:\n  mode: baseline\n"), 0644))
	f.writeUserConfig(t, "# keep me\nlegacyVersionFile:\n    enable: false\n    strategy: latest_installed\n")

	f.run(t, "config", "path.mode", "aggressive").mustSucceed(t, "config path.mode aggressive")
	mustContain(t, "user config", readFile(t, userConfig), "path:", "mode: aggressive")
	if out := f.run(t, "config", "path.mode").mustSucceed(t, "config path.mode").stdout; strings.TrimSpace(out) != "aggressive" {
		t.Errorf("path.mode = %q after setting it", out)
	}

	f.run(t, "config", "--unset", "path.mode").mustSucceed(t, "config --unset path.mode")
	mustNotContain(t, "user config", readFile(t, userConfig), "path", "mode")
	if out := f.run(t, "config", "path.mode").mustSucceed(t, "config path.mode").stdout; strings.TrimSpace(out) != "baseline" {
		t.Errorf("path.mode = %q after unset, want the shared baseline and not the default", out)
	}

	// A section that has other keys keeps them, and the comments stay.
	f.run(t, "config", "--unset", "legacyVersionFile.strategy").mustSucceed(t, "config --unset legacyVersionFile.strategy")
	user := readFile(t, userConfig)
	mustContain(t, "user config", user, "# keep me", "legacyVersionFile:", "enable: false")
	mustNotContain(t, "user config", user, "strategy", "latest_installed")
	list := f.run(t, "config", "--list").mustSucceed(t, "config --list").stdout
	mustContain(t, "config --list", list, "legacyVersionFile.enable = false", "legacyVersionFile.strategy = specified")

	// A key that is not set is not an error, and the file stays as it is.
	before := readFile(t, userConfig)
	for _, key := range []string{"path.mode", "diagnostics", "diagnostics.onUse", "proxy.url"} {
		f.run(t, "config", "--unset", key).mustSucceed(t, "config --unset "+key)
	}
	if after := readFile(t, userConfig); after != before {
		t.Errorf("the file changed though it had none of the keys:\n%s\n---\n%s", before, after)
	}

	// A whole section can go.
	f.run(t, "config", "--unset", "legacyVersionFile").mustSucceed(t, "config --unset legacyVersionFile")
	mustNotContain(t, "user config", readFile(t, userConfig), "legacyVersionFile", "enable")
	list = f.run(t, "config", "--list").mustSucceed(t, "config --list").stdout
	mustContain(t, "config --list", list, "legacyVersionFile.enable = true")
}

// A key or value that does not fit leaves the file alone and says why.
func TestConfigRefusesWhatItCannotSet(t *testing.T) {
	f := newCLIFixture(t, false)
	userConfig := filepath.Join(f.userHome, "config.yaml")
	f.writeUserConfig(t, "# keep\nproxy:\n    enable: true\n")
	before := readFile(t, userConfig)

	for name, c := range map[string]struct {
		args []string
		want string
	}{
		"a key that does not exist": {[]string{"proxy.ulr", "x"}, "unknown config key: proxy.ulr"},
		"unsetting such a key":      {[]string{"--unset", "proxy.ulr"}, "unknown config key: proxy.ulr"},
		"a section":                 {[]string{"proxy", "x"}, "proxy is a section, not a setting"},
		"a boolean that is none":    {[]string{"proxy.enable", "yes"}, `proxy.enable: "yes" is neither true nor false`},
		"a duration that is none":   {[]string{"plugin.http.timeout", "soon"}, "plugin.http.timeout: time: invalid duration"},
		"a list":                    {[]string{"path.preserve", "/a"}, "path.preserve holds a list or a table"},
		"the aliases":               {[]string{"aliases", "x"}, "aliases holds a list or a table"},
	} {
		t.Run(name, func(t *testing.T) {
			r := f.run(t, append([]string{"config"}, c.args...)...).mustFail(t, "config "+strings.Join(c.args, " "))
			mustContain(t, "the error", r.output(), c.want)
			if after := readFile(t, userConfig); after != before {
				t.Errorf("the file changed:\n%s", after)
			}
		})
	}
}
