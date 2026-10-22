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
	"testing"
)

// A section the user's file declares replaces the shared one, so the file vfox
// creates on its first run must not declare the sections that are only the
// user's when they set them. Otherwise the first run would write the defaults
// of `path` and `diagnostics` into the file, and from the second run on they
// would override what a shared VFOX_HOME says.
func TestANewConfigFileLeavesOutTheSectionsThatAreSetOnDemand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	created, err := NewConfigWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if created.Path.EffectiveMode() != PathModeLegacy || !created.Diagnostics.OnUse {
		t.Errorf("the first run works with the defaults: %+v %+v", created.Path, created.Diagnostics)
	}

	want := []string{"cache", "gitignore", "legacyVersionFile", "plugin", "proxy", "registry", "storage"}
	if got := topLevelKeys(t, path); !reflect.DeepEqual(got, want) {
		t.Errorf("sections of the new file = %v, want %v", got, want)
	}

	// The file still loads into the same defaults.
	loaded, err := NewConfigWithPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Path.EffectiveMode() != PathModeLegacy || !loaded.Diagnostics.OnUse || loaded.Aliases != nil {
		t.Errorf("the second run: path %+v, diagnostics %+v, aliases %v", loaded.Path, loaded.Diagnostics, loaded.Aliases)
	}
	if loaded.LegacyVersionFile.Enable != DefaultConfig.LegacyVersionFile.Enable || loaded.Cache.AvailableHookDuration != DefaultConfig.Cache.AvailableHookDuration {
		t.Errorf("the sections that are in the file changed: %+v %+v", loaded.LegacyVersionFile, loaded.Cache)
	}
}

// The user of a shared VFOX_HOME: the administrator's settings must still apply
// after any number of runs.
func TestSharedSettingsApplyOverANewUserConfigFile(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, "shared.yaml")
	if err := os.WriteFile(shared, []byte("path:\n  mode: baseline\ndiagnostics:\n  onUse: false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	user := filepath.Join(dir, "config.yaml") // does not exist yet

	for run := 1; run <= 3; run++ {
		c, err := LoadConfigWithFallback(shared, user)
		if err != nil {
			t.Fatal(err)
		}
		if mode := c.Path.EffectiveMode(); mode != PathModeBaseline {
			t.Errorf("run %d: path.mode = %q, want the shared baseline", run, mode)
		}
		if c.Diagnostics.OnUse {
			t.Errorf("run %d: diagnostics.onUse is on, but the shared config switched it off", run)
		}
	}

	// What the user sets in their own file still wins.
	if err := os.WriteFile(user, []byte("path:\n  mode: aggressive\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfigWithFallback(shared, user)
	if err != nil {
		t.Fatal(err)
	}
	if mode := c.Path.EffectiveMode(); mode != PathModeAggressive {
		t.Errorf("path.mode = %q, want the user's own setting to win", mode)
	}
}
