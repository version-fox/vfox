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

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/version-fox/vfox/internal/shared/cache"
)

// setKey loads the file, sets one key and saves it, as `vfox config <key> <value>` does.
func setKey(t *testing.T, path, key, value string) {
	t.Helper()
	file, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Set(key, value); err != nil {
		t.Fatalf("set %s %s: %v", key, value, err)
	}
	if err := file.Save(); err != nil {
		t.Fatal(err)
	}
}

// unsetKey is `vfox config --unset <key>`: it saves only when the key was there.
func unsetKey(t *testing.T, path, key string) bool {
	t.Helper()
	file, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	removed, err := file.Unset(key)
	if err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}
	if removed {
		if err := file.Save(); err != nil {
			t.Fatal(err)
		}
	}
	return removed
}

// declared is what the file declares, as the plain data it is: the sections
// and the keys in them, with nothing filled in.
func declared(t *testing.T, path string) map[string]any {
	t.Helper()
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(readConfigFile(t, path)), &raw); err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestSetChangesOnlyThatKey(t *testing.T) {
	path := writeConfigFile(t, `# my vfox settings

proxy:
    # the corporate one
    url: http://proxy.example.com:8080 # do not change the port
    enable: true
legacyVersionFile:
    enable: false
cache:
    availableHookDuration: 1h
`)
	setKey(t, path, "proxy.url", "http://proxy.example.com:3128")

	saved := readConfigFile(t, path)
	for _, want := range []string{
		"# my vfox settings", "# the corporate one", "# do not change the port",
		"url: http://proxy.example.com:3128", "enable: true", "availableHookDuration: 1h",
	} {
		if !strings.Contains(saved, want) {
			t.Errorf("the saved file lost or lacks %q:\n%s", want, saved)
		}
	}
	// Nothing was added: not a default, not a section the user never wrote.
	if got, want := topLevelKeys(t, path), []string{"cache", "legacyVersionFile", "proxy"}; !reflect.DeepEqual(got, want) {
		t.Errorf("sections = %v, want %v", got, want)
	}
	loaded, err := NewConfigWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Proxy.Url != "http://proxy.example.com:3128" || !loaded.Proxy.Enable || loaded.LegacyVersionFile.Enable ||
		loaded.Cache.AvailableHookDuration != cache.Duration(time.Hour) {
		t.Errorf("loaded = %+v %+v %+v", loaded.Proxy, loaded.LegacyVersionFile, loaded.Cache)
	}
}

// A section the file does not declare replaces the shared one only once the
// file declares it, and then it replaces it as a whole: so the file declares a
// section with the one key that was set, and no other.
func TestSetDeclaresOnlyTheKeyAndItsSections(t *testing.T) {
	path := writeConfigFile(t, "cache:\n    availableHookDuration: 1h\n")
	setKey(t, path, "path.mode", "baseline")
	setKey(t, path, "plugin.http.timeout", "45s")

	want := map[string]any{
		"cache":  map[string]any{"availableHookDuration": "1h"},
		"path":   map[string]any{"mode": "baseline"},
		"plugin": map[string]any{"http": map[string]any{"timeout": "45s"}},
	}
	if got := declared(t, path); !reflect.DeepEqual(got, want) {
		t.Errorf("declared = %v, want %v:\n%s", got, want, readConfigFile(t, path))
	}

	loaded, err := NewConfigWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Path.EffectiveMode() != PathModeBaseline || len(loaded.Path.Preserve) != 0 || !loaded.Diagnostics.OnUse {
		t.Errorf("loaded path %+v, diagnostics %+v", loaded.Path, loaded.Diagnostics)
	}
	if request, _ := loaded.Plugin.HTTP.Timeouts(); request != 45*time.Second {
		t.Errorf("timeout = %v", request)
	}
}

// The same keys end up in the file whatever the file looked like before.
func TestSetOnFilesWithLittleContent(t *testing.T) {
	for name, content := range map[string]string{
		"no content":                  "",
		"only a comment":              "# nothing here yet\n",
		"a null document":             "~\n",
		"an empty mapping":            "{}\n",
		"a section without content":   "path:\n",
		"a section that is null":      "path: ~\n",
		"an empty section":            "path: {}\n",
		"a document start":            "---\n",
		"an unrelated section only":   "storage:\n    sdkPath: /tmp\n",
		"a section in the flow style": "path: {preserve: [/a]}\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := writeConfigFile(t, content)
			setKey(t, path, "path.mode", "baseline")

			loaded, err := NewConfigWithPath(path)
			if err != nil {
				t.Fatalf("%v:\n%s", err, readConfigFile(t, path))
			}
			if loaded.Path.EffectiveMode() != PathModeBaseline {
				t.Errorf("path.mode = %q:\n%s", loaded.Path.EffectiveMode(), readConfigFile(t, path))
			}
			if _, ok := declared(t, path)["diagnostics"]; ok {
				t.Errorf("the file declares a section nobody set:\n%s", readConfigFile(t, path))
			}
			if strings.HasPrefix(content, "#") && !strings.Contains(readConfigFile(t, path), strings.TrimSpace(content)) {
				t.Errorf("the comment of a file that held nothing else is gone:\n%s", readConfigFile(t, path))
			}
		})
	}
}

func TestSetWritesTheValueAsTheTypeOfTheSetting(t *testing.T) {
	path := writeConfigFile(t, "")
	for key, value := range map[string]string{
		"gitignore.enable":            "false",
		"proxy.enable":                "TRUE", // ParseBool takes more than lower case
		"plugin.http.timeout":         "90s",
		"plugin.http.downloadTimeout": "1h",
		"cache.availableHookDuration": "2H",  // read in lower case, as it always was
		"legacyVersionFile.strategy":  "yes", // text that is no string to YAML 1.1
		"storage.sdkPath":             "21",  // nor is this
		"registry.address":            "true",
		"proxy.url":                   "http://localhost:7890",
		"diagnostics.onUse":           "0",
		"legacyVersionFile.enable":    "f",
		"path.mode":                   "aggressive",
	} {
		setKey(t, path, key, value)
	}

	loaded, err := NewConfigWithPath(path)
	if err != nil {
		t.Fatalf("%v:\n%s", err, readConfigFile(t, path))
	}
	request, download := loaded.Plugin.HTTP.Timeouts()
	checks := []struct {
		name      string
		got, want any
	}{
		{"gitignore.enable", loaded.Gitignore.Enable, false},
		{"proxy.enable", loaded.Proxy.Enable, true},
		{"plugin.http.timeout", request, 90 * time.Second},
		{"plugin.http.downloadTimeout", download, time.Hour},
		{"cache.availableHookDuration", loaded.Cache.AvailableHookDuration, cache.Duration(2 * time.Hour)},
		{"legacyVersionFile.strategy", loaded.LegacyVersionFile.Strategy, "yes"},
		{"storage.sdkPath", loaded.Storage.SdkPath, "21"},
		{"registry.address", loaded.Registry.Address, "true"},
		{"proxy.url", loaded.Proxy.Url, "http://localhost:7890"},
		{"diagnostics.onUse", loaded.Diagnostics.OnUse, false},
		{"legacyVersionFile.enable", loaded.LegacyVersionFile.Enable, false},
		{"path.mode", loaded.Path.Mode, "aggressive"},
	}
	for _, c := range checks {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s = %#v, want %#v", c.name, c.got, c.want)
		}
	}
	if t.Failed() {
		t.Logf("the file:\n%s", readConfigFile(t, path))
	}
}

func TestSetHTTPTimeouts(t *testing.T) {
	path := writeConfigFile(t, "")
	for _, key := range []string{"plugin.http.timeout", "plugin.http.downloadTimeout"} {
		setKey(t, path, key, "45m")
		for _, value := range []string{"invalid", "-1s"} {
			file, err := LoadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := file.Set(key, value); err == nil {
				t.Fatalf("accepted %q for %s", value, key)
			}
		}
	}
	loaded, err := NewConfigWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Plugin.HTTP.Timeout != 45*time.Minute || loaded.Plugin.HTTP.DownloadTimeout != 45*time.Minute {
		t.Fatalf("HTTP = %+v", loaded.Plugin.HTTP)
	}

	// Unsetting takes the timeouts out of the file, which gives the defaults.
	for _, key := range []string{"plugin.http.timeout", "plugin.http.downloadTimeout"} {
		if !unsetKey(t, path, key) {
			t.Fatalf("%s was set, so unset removes it", key)
		}
	}
	if got := declared(t, path); len(got) != 0 {
		t.Errorf("the file still declares %v:\n%s", got, readConfigFile(t, path))
	}
	loaded, err = NewConfigWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if request, download := loaded.Plugin.HTTP.Timeouts(); request != 30*time.Second || download != 30*time.Minute {
		t.Fatalf("unset timeouts = %v, %v", request, download)
	}
}

func TestSetCacheDuration(t *testing.T) {
	for value, want := range map[string]cache.Duration{
		"-1":    cache.Duration(-1),
		"0":     cache.Duration(0),
		"0s":    cache.Duration(0),
		"30m":   cache.Duration(30 * time.Minute),
		"1h30m": cache.Duration(90 * time.Minute),
		"24H":   cache.Duration(24 * time.Hour),
	} {
		path := writeConfigFile(t, "")
		setKey(t, path, "cache.availableHookDuration", value)
		loaded, err := NewConfigWithPath(path)
		if err != nil {
			t.Fatalf("%s: %v:\n%s", value, err, readConfigFile(t, path))
		}
		if loaded.Cache.AvailableHookDuration != want {
			t.Errorf("cache.availableHookDuration %s = %v, want %v", value, loaded.Cache.AvailableHookDuration, want)
		}
	}
}

// An error leaves the file, and what is loaded of it, as it was.
func TestSetRefusesWhatItCannotSet(t *testing.T) {
	const content = "# keep\nproxy:\n    url: http://proxy.example.com:8080\n"
	for name, c := range map[string]struct{ key, value, want string }{
		"a key that does not exist":          {"proxy.ulr", "x", "unknown config key: proxy.ulr"},
		"a section that does not exist":      {"nope", "x", "unknown config key: nope"},
		"a key below a section that is not":  {"proxy.url.more", "x", "unknown config key: proxy.url.more"},
		"an empty key":                       {"", "x", "unknown config key"},
		"an empty name in the key":           {"proxy..url", "x", "unknown config key"},
		"a trailing dot":                     {"proxy.", "x", "unknown config key"},
		"a section":                          {"proxy", "x", "proxy is a section, not a setting"},
		"a section of sections":              {"plugin", "x", "plugin is a section, not a setting"},
		"a section inside a section":         {"plugin.http", "x", "plugin.http is a section, not a setting"},
		"a list":                             {"path.preserve", "/a", "path.preserve holds a list or a table"},
		"another list":                       {"diagnostics.ignore", "x", "diagnostics.ignore holds a list or a table"},
		"the aliases":                        {"aliases", "x", "aliases holds a list or a table"},
		"an alias":                           {"aliases.java", "x", "unknown config key: aliases.java"},
		"a boolean that is none":             {"proxy.enable", "yes", `proxy.enable: "yes" is neither true nor false`},
		"an empty boolean":                   {"gitignore.enable", "", `gitignore.enable: "" is neither true nor false`},
		"a duration that is none":            {"plugin.http.timeout", "invalid", "plugin.http.timeout: time: invalid duration"},
		"a negative duration":                {"plugin.http.timeout", "-1s", "plugin.http.timeout: must not be negative"},
		"a negative download duration":       {"plugin.http.downloadTimeout", "-5m", "plugin.http.downloadTimeout: must not be negative"},
		"a cache duration that is none":      {"cache.availableHookDuration", "forever", "cache.availableHookDuration: time: invalid duration"},
		"a negative cache duration":          {"cache.availableHookDuration", "-5m", "cache.availableHookDuration: must be -1"},
		"a cache duration too short to keep": {"cache.availableHookDuration", "500ms", "cache.availableHookDuration: must be at least 1s"},
	} {
		t.Run(name, func(t *testing.T) {
			path := writeConfigFile(t, content)
			file, err := LoadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			err = file.Set(c.key, c.value)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Set(%q, %q) = %v, want an error with %q", c.key, c.value, err, c.want)
			}

			// The document did not change either: saving it gives what was there.
			if err := file.Save(); err != nil {
				t.Fatal(err)
			}
			if got, want := declared(t, path), map[string]any{"proxy": map[string]any{"url": "http://proxy.example.com:8080"}}; !reflect.DeepEqual(got, want) {
				t.Errorf("declared = %v, want %v", got, want)
			}
			if saved := readConfigFile(t, path); !strings.Contains(saved, "# keep") {
				t.Errorf("the comment is gone:\n%s", saved)
			}
		})
	}
}

func TestSetRefusesASectionThatIsNoMapping(t *testing.T) {
	path := writeConfigFile(t, "path: baseline\n")
	file, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Set("path.mode", "aggressive"); err == nil || !strings.Contains(err.Error(), "path is not a section") {
		t.Errorf("Set = %v, want an error that path is not a section", err)
	}
	if got := readConfigFile(t, path); got != "path: baseline\n" {
		t.Errorf("the file was changed: %q", got)
	}
}

func TestUnsetRemovesTheKeyAndTheSectionsItLeavesEmpty(t *testing.T) {
	path := writeConfigFile(t, `# my vfox settings

proxy:
    url: http://proxy.example.com:8080
    enable: true # on purpose
legacyVersionFile:
    enable: false
    strategy: latest_installed
plugin:
    http:
        timeout: 45s
path:
    mode: aggressive
    preserve: [/opt/tools/bin]
`)
	steps := []struct {
		key  string
		want map[string]any
	}{
		{"proxy.url", map[string]any{
			"proxy":             map[string]any{"enable": true},
			"legacyVersionFile": map[string]any{"enable": false, "strategy": "latest_installed"},
			"plugin":            map[string]any{"http": map[string]any{"timeout": "45s"}},
			"path":              map[string]any{"mode": "aggressive", "preserve": []any{"/opt/tools/bin"}},
		}},
		{"proxy.enable", map[string]any{ // the last key of a section takes the section along
			"legacyVersionFile": map[string]any{"enable": false, "strategy": "latest_installed"},
			"plugin":            map[string]any{"http": map[string]any{"timeout": "45s"}},
			"path":              map[string]any{"mode": "aggressive", "preserve": []any{"/opt/tools/bin"}},
		}},
		{"plugin.http.timeout", map[string]any{ // and the section above that one too
			"legacyVersionFile": map[string]any{"enable": false, "strategy": "latest_installed"},
			"path":              map[string]any{"mode": "aggressive", "preserve": []any{"/opt/tools/bin"}},
		}},
		{"legacyVersionFile.strategy", map[string]any{ // another key of the section stays
			"legacyVersionFile": map[string]any{"enable": false},
			"path":              map[string]any{"mode": "aggressive", "preserve": []any{"/opt/tools/bin"}},
		}},
		{"path", map[string]any{ // a section goes as a whole
			"legacyVersionFile": map[string]any{"enable": false},
		}},
		{"legacyVersionFile.enable", map[string]any{}},
	}
	for _, step := range steps {
		if !unsetKey(t, path, step.key) {
			t.Fatalf("unset %s: the file had the key, but it was not removed", step.key)
		}
		if got := declared(t, path); !reflect.DeepEqual(got, step.want) {
			t.Fatalf("after unset %s: declared = %v, want %v:\n%s", step.key, got, step.want, readConfigFile(t, path))
		}
		if _, err := NewConfigWithPath(path); err != nil {
			t.Fatalf("after unset %s the file does not load: %v", step.key, err)
		}
	}
}

func TestUnsetKeepsTheCommentsOfTheOtherKeys(t *testing.T) {
	path := writeConfigFile(t, `# my vfox settings

proxy:
    # the corporate one
    url: http://proxy.example.com:8080 # do not change the port
    enable: true # on purpose
cache:
    # a day is enough
    availableHookDuration: 24h
`)
	unsetKey(t, path, "proxy.enable")
	unsetKey(t, path, "cache.availableHookDuration")

	saved := readConfigFile(t, path)
	for _, want := range []string{"# my vfox settings", "# the corporate one", "# do not change the port", "url: http://proxy.example.com:8080"} {
		if !strings.Contains(saved, want) {
			t.Errorf("the saved file lost %q:\n%s", want, saved)
		}
	}
	// The comments of the keys that were removed go with them.
	for _, gone := range []string{"# on purpose", "# a day is enough", "enable", "cache"} {
		if strings.Contains(saved, gone) {
			t.Errorf("the saved file still has %q:\n%s", gone, saved)
		}
	}
}

// The key is not there: nothing is removed, and that is no error.
func TestUnsetOfAKeyThatIsNotSet(t *testing.T) {
	const content = "# keep\nproxy:\n    enable: true\n"
	for _, key := range []string{"proxy.url", "path.mode", "path", "plugin.http.timeout", "aliases"} {
		path := writeConfigFile(t, content)
		file, err := LoadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		removed, err := file.Unset(key)
		if err != nil || removed {
			t.Errorf("Unset(%q) = %v, %v, want false and no error", key, removed, err)
		}
		if got := readConfigFile(t, path); got != content {
			t.Errorf("the file was changed: %q", got)
		}
	}

	file, err := LoadFile(filepath.Join(t.TempDir(), "config.yaml")) // there is no file at all
	if err != nil {
		t.Fatal(err)
	}
	if removed, err := file.Unset("proxy.url"); err != nil || removed {
		t.Errorf("Unset in a missing file = %v, %v", removed, err)
	}
}

func TestUnsetRefusesAKeyThatDoesNotExist(t *testing.T) {
	file, err := LoadFile(writeConfigFile(t, "proxy:\n    enable: true\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"proxy.ulr", "nope", "", "proxy.enable.more", "aliases.java"} {
		if removed, err := file.Unset(key); err == nil || removed || !strings.Contains(err.Error(), "unknown config key") {
			t.Errorf("Unset(%q) = %v, %v, want an unknown key", key, removed, err)
		}
	}
	// A list, a table and a section can be removed even though they cannot be set.
	for _, key := range []string{"path.preserve", "diagnostics.ignore", "aliases", "proxy"} {
		if _, err := file.Unset(key); err != nil {
			t.Errorf("Unset(%q): %v", key, err)
		}
	}
}

// Setting and unsetting are meant for a user who shares VFOX_HOME: what the
// user did not set follows the shared config, also when that changes later.
func TestOnlyWhatIsSetLeavesTheSharedConfig(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, "shared.yaml")
	user := filepath.Join(dir, "config.yaml")
	writeShared := func(content string) {
		t.Helper()
		if err := os.WriteFile(shared, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	load := func() *Config {
		t.Helper()
		c, err := LoadConfigWithFallback(shared, user)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	writeShared("path:\n  mode: baseline\ndiagnostics:\n  onUse: false\naliases:\n  alpha:\n    team: 1.0.0\n")

	// The first run creates the file; then one setting is changed.
	load()
	setKey(t, user, "gitignore.enable", "false")
	for _, section := range []string{"path", "diagnostics", "aliases"} {
		if _, ok := declared(t, user)[section]; ok {
			t.Errorf("the user's file declares %s, which only the shared config set:\n%s", section, readConfigFile(t, user))
		}
	}
	c := load()
	if c.Gitignore.Enable || c.Path.EffectiveMode() != PathModeBaseline || c.Diagnostics.OnUse || c.Aliases["alpha"]["team"] != "1.0.0" {
		t.Errorf("gitignore %+v, path %+v, diagnostics %+v, aliases %v", c.Gitignore, c.Path, c.Diagnostics, c.Aliases)
	}

	// A later change of the shared config still applies.
	writeShared("path:\n  mode: aggressive\ndiagnostics:\n  onUse: true\naliases:\n  alpha:\n    team: 2.0.0\n")
	c = load()
	if c.Path.EffectiveMode() != PathModeAggressive || !c.Diagnostics.OnUse || c.Aliases["alpha"]["team"] != "2.0.0" {
		t.Errorf("path %+v, diagnostics %+v, aliases %v", c.Path, c.Diagnostics, c.Aliases)
	}

	// The user's own choice wins, and unsetting it hands the setting back.
	setKey(t, user, "path.mode", "legacy")
	if mode := load().Path.EffectiveMode(); mode != PathModeLegacy {
		t.Errorf("path.mode = %q, want the user's own", mode)
	}
	if !unsetKey(t, user, "path.mode") {
		t.Fatal("path.mode was set, so unset removes it")
	}
	if _, ok := declared(t, user)["path"]; ok {
		t.Errorf("the user's file still declares path:\n%s", readConfigFile(t, user))
	}
	if mode := load().Path.EffectiveMode(); mode != PathModeAggressive {
		t.Errorf("path.mode = %q after unset, want the shared aggressive", mode)
	}
}

// A section that holds only some of its keys keeps the defaults of the others.
func TestASectionWithOneKeyKeepsTheDefaultsOfTheOthers(t *testing.T) {
	path := writeConfigFile(t, "")
	setKey(t, path, "legacyVersionFile.strategy", "latest_installed")

	loaded, err := NewConfigWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.LegacyVersionFile.Enable || loaded.LegacyVersionFile.Strategy != LatestInstalledStrategy {
		t.Errorf("legacyVersionFile = %+v: setting the strategy must not switch the parsing off", loaded.LegacyVersionFile)
	}

	setKey(t, path, "legacyVersionFile.enable", "false")
	loaded, err = NewConfigWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LegacyVersionFile.Enable || loaded.LegacyVersionFile.Strategy != LatestInstalledStrategy {
		t.Errorf("legacyVersionFile = %+v", loaded.LegacyVersionFile)
	}

	// And a file that never mentions the strategy has the default one.
	hand := writeConfigFile(t, "legacyVersionFile:\n  enable: false\n")
	loaded, err = NewConfigWithPath(hand)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.LegacyVersionFile.Enable || loaded.LegacyVersionFile.Strategy != SpecifiedStrategy {
		t.Errorf("legacyVersionFile = %+v", loaded.LegacyVersionFile)
	}
}

// A file that vfox cannot load stops every command, so Save does not write one.
func TestSaveRefusesWhatVfoxCouldNotLoad(t *testing.T) {
	const content = "proxy:\n    enable: true\n"
	for name, change := range map[string]func(f *File) error{
		"a negative timeout": func(f *File) error { return f.put([]string{"plugin", "http", "timeout"}, -5*time.Second) },
		"a cache duration of 0.5s": func(f *File) error {
			return f.put([]string{"cache", "availableHookDuration"}, cache.Duration(500*time.Millisecond))
		},
		"a section that is a text": func(f *File) error { return f.put([]string{"path"}, "baseline") },
	} {
		t.Run(name, func(t *testing.T) {
			path := writeConfigFile(t, content)
			file, err := LoadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := change(file); err != nil {
				t.Fatal(err)
			}
			err = file.Save()
			if err == nil || !strings.Contains(err.Error(), "would not load") {
				t.Fatalf("Save = %v, want a refusal", err)
			}
			if got := readConfigFile(t, path); got != content {
				t.Errorf("the file was changed: %q", got)
			}
		})
	}
}

func TestSetTakesTheFirstKeyOfAnEmptyMappingAsABlock(t *testing.T) {
	path := writeConfigFile(t, "{}\n")
	setKey(t, path, "path.mode", "baseline")
	if saved := readConfigFile(t, path); strings.Contains(saved, "{") {
		t.Errorf("the file is still in the flow style:\n%s", saved)
	}
}

func TestSetCreatesAMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	setKey(t, path, "path.mode", "baseline")
	if got, want := declared(t, path), map[string]any{"path": map[string]any{"mode": "baseline"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("declared = %v, want %v", got, want)
	}
}

func TestLoadFileRefusesWhatItCannotEditSafely(t *testing.T) {
	for name, content := range map[string]string{
		"invalid YAML":        "proxy: [unclosed\n",
		"a list at the top":   "- one\n- two\n",
		"a scalar at the top": "just text\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := writeConfigFile(t, content)
			if file, err := LoadFile(path); err == nil {
				t.Fatalf("loaded %+v, want an error", file)
			}
			if got := readConfigFile(t, path); got != content {
				t.Errorf("the file was changed: %q", got)
			}
		})
	}
	if _, err := LoadFile(t.TempDir()); err == nil { // a directory
		t.Error("want an error for a directory")
	}
}

func TestFileTopMapping(t *testing.T) {
	for name, doc := range map[string]yaml.Node{
		"a scalar instead of a document": {Kind: yaml.ScalarNode, Value: "x"},
		"a document without content":     {Kind: yaml.DocumentNode},
		"a document of two nodes":        {Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}, {Kind: yaml.MappingNode}}},
		"a document holding a sequence":  {Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.SequenceNode}}},
	} {
		f := &File{doc: doc}
		if root, err := f.topMapping(); err == nil {
			t.Errorf("%s: root = %+v, want an error", name, root)
		}
	}
	empty := &File{}
	if root, err := empty.topMapping(); err != nil || root.Kind != yaml.MappingNode {
		t.Errorf("an empty file: root = %+v, %v, want an empty mapping", root, err)
	}
}
