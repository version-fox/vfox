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
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readConfigFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// topLevelKeys lists the sections a config file declares, as the file has them.
func topLevelKeys(t *testing.T, path string) []string {
	t.Helper()
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(readConfigFile(t, path)), &raw); err != nil {
		t.Fatal(err)
	}
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func saveAliases(t *testing.T, path string, change func(aliases map[string]map[string]string)) {
	t.Helper()
	file, err := LoadAliasFile(path)
	if err != nil {
		t.Fatal(err)
	}
	change(file.Aliases)
	if err := file.Save(); err != nil {
		t.Fatal(err)
	}
}

func TestAliasFileChangesOnlyTheAliases(t *testing.T) {
	path := writeConfigFile(t, `# my vfox settings
proxy:
    url: http://proxy.example.com:8080 # the corporate one
    enable: true
legacyVersionFile:
    enable: false
aliases:
    java:
        j17: 17.0.9-tem
`)
	saveAliases(t, path, func(aliases map[string]map[string]string) {
		aliases["java"]["j21"] = "21.0.1"
		aliases["go"] = map[string]string{"stable": "1.25.6"}
	})

	saved := readConfigFile(t, path)
	for _, want := range []string{
		"# my vfox settings", "# the corporate one", "url: http://proxy.example.com:8080", "enable: true",
		"j17: 17.0.9-tem", "j21: 21.0.1", "stable: 1.25.6",
	} {
		if !strings.Contains(saved, want) {
			t.Errorf("the saved file lost or lacks %q:\n%s", want, saved)
		}
	}
	// Nothing was added: not a default, not a section the user never wrote.
	if got, want := topLevelKeys(t, path), []string{"aliases", "legacyVersionFile", "proxy"}; !reflect.DeepEqual(got, want) {
		t.Errorf("sections = %v, want %v", got, want)
	}

	// And it still loads as a config, with the aliases.
	loaded, err := NewConfigWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]map[string]string{"java": {"j17": "17.0.9-tem", "j21": "21.0.1"}, "go": {"stable": "1.25.6"}}
	if !reflect.DeepEqual(loaded.Aliases, want) || !loaded.Proxy.Enable || loaded.LegacyVersionFile.Enable {
		t.Errorf("loaded = %+v (aliases %v)", loaded, loaded.Aliases)
	}
}

// The case that is easy to get wrong: a file that declares nothing but aliases
// must still declare nothing but aliases afterwards.
func TestAliasFileDoesNotDeclareTheDefaults(t *testing.T) {
	for name, content := range map[string]string{
		"an empty mapping": "aliases: {}\n",
		"no content":       "",
		"only a comment":   "# nothing here yet\n",
		"a null document":  "~\n",
		"no aliases":       "{}\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := writeConfigFile(t, content)
			saveAliases(t, path, func(aliases map[string]map[string]string) {
				aliases["alpha"] = map[string]string{"stable": "1.0.0"}
			})
			if got, want := topLevelKeys(t, path), []string{"aliases"}; !reflect.DeepEqual(got, want) {
				t.Errorf("sections = %v, want %v:\n%s", got, want, readConfigFile(t, path))
			}
			loaded, err := LoadAliasFile(path)
			if err != nil || loaded.Aliases["alpha"]["stable"] != "1.0.0" {
				t.Errorf("reloaded = %+v, %v", loaded, err)
			}
		})
	}
}

// A file that holds comments and nothing else keeps them.
func TestAliasFileKeepsTheCommentsOfAFileWithoutContent(t *testing.T) {
	for _, content := range []string{"# nothing here yet\n", "# no newline at the end"} {
		path := writeConfigFile(t, content)
		saveAliases(t, path, func(aliases map[string]map[string]string) {
			aliases["alpha"] = map[string]string{"stable": "1.0.0"}
		})
		saved := readConfigFile(t, path)
		if !strings.HasPrefix(saved, strings.TrimSuffix(content, "\n")+"\n") {
			t.Errorf("the comments of %q are gone:\n%s", content, saved)
		}
		if loaded, err := LoadAliasFile(path); err != nil || loaded.Aliases["alpha"]["stable"] != "1.0.0" {
			t.Errorf("reloaded = %+v, %v", loaded, err)
		}
	}
}

func TestAliasFileCreatesAMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	file, err := LoadAliasFile(path)
	if err != nil || len(file.Aliases) != 0 {
		t.Fatalf("a missing file has no aliases: %+v, %v", file, err)
	}
	file.Aliases["alpha"] = map[string]string{"stable": "1.0.0"}
	if err := file.Save(); err != nil {
		t.Fatal(err)
	}
	if got, want := topLevelKeys(t, path), []string{"aliases"}; !reflect.DeepEqual(got, want) {
		t.Errorf("sections = %v, want %v", got, want)
	}
}

func TestAliasFileDropsTheKeyWhenNoAliasIsLeft(t *testing.T) {
	path := writeConfigFile(t, "cache:\n    availableHookDuration: 1h\naliases:\n    alpha:\n        stable: 1.0.0\n")
	saveAliases(t, path, func(aliases map[string]map[string]string) {
		delete(aliases["alpha"], "stable") // leaves an SDK without aliases
	})
	if got, want := topLevelKeys(t, path), []string{"cache"}; !reflect.DeepEqual(got, want) {
		t.Errorf("sections = %v, want %v", got, want)
	}

	// Nothing to remove from a file that has no aliases.
	saveAliases(t, path, func(map[string]map[string]string) {})
	if got, want := topLevelKeys(t, path), []string{"cache"}; !reflect.DeepEqual(got, want) {
		t.Errorf("sections = %v, want %v", got, want)
	}
}

// A version that reads as a number must not turn into one.
func TestAliasFileKeepsVersionsAsStrings(t *testing.T) {
	path := writeConfigFile(t, "")
	saveAliases(t, path, func(aliases map[string]map[string]string) {
		aliases["alpha"] = map[string]string{"major": "21", "minor": "1.0", "yes": "true"}
	})
	loaded, err := LoadAliasFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"major": "21", "minor": "1.0", "yes": "true"}
	if !reflect.DeepEqual(loaded.Aliases["alpha"], want) {
		t.Errorf("aliases = %v, want %v:\n%s", loaded.Aliases["alpha"], want, readConfigFile(t, path))
	}

	// Versions written by hand without quotes are read as the text they are.
	hand := writeConfigFile(t, "aliases:\n  alpha:\n    minor: 1.0\n    major: 21\n")
	file, err := LoadAliasFile(hand)
	if err != nil || file.Aliases["alpha"]["minor"] != "1.0" || file.Aliases["alpha"]["major"] != "21" {
		t.Errorf("hand-written aliases = %v, %v", file.Aliases, err)
	}
}

func TestAliasFileRefusesWhatItCannotChangeSafely(t *testing.T) {
	for name, content := range map[string]string{
		"invalid YAML":             "aliases: [unclosed\n",
		"aliases that are no map":  "aliases: [not, a, map]\n",
		"a list at the top":        "- one\n- two\n",
		"a scalar at the top":      "just text\n",
		"a version that is a list": "aliases:\n  alpha:\n    stable: [1, 2]\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := writeConfigFile(t, content)
			if file, err := LoadAliasFile(path); err == nil {
				t.Fatalf("loaded %+v, want an error", file)
			}
			if got := readConfigFile(t, path); got != content {
				t.Errorf("the file was changed: %q", got)
			}
		})
	}

	t.Run("a path that cannot be read", func(t *testing.T) {
		if _, err := LoadAliasFile(t.TempDir()); err == nil { // a directory
			t.Error("want an error for a directory")
		}
	})

	t.Run("a file that cannot be written", func(t *testing.T) {
		file, err := LoadAliasFile(filepath.Join(t.TempDir(), "missing", "config.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		file.Aliases["alpha"] = map[string]string{"stable": "1.0.0"}
		if err := file.Save(); err == nil {
			t.Error("want an error when the directory is not there")
		}
	})
}
