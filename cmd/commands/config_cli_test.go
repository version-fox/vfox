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

// The first run creates the user's config file. It must not pin the settings
// that a shared VFOX_HOME decides, or they stop applying from the second run on.
func TestAFreshUserFollowsTheSharedConfig(t *testing.T) {
	f := newCLIFixture(t, true)
	// Settings that differ from the defaults, so that losing them shows.
	must(t, os.WriteFile(filepath.Join(f.shared, "config.yaml"), []byte("path:\n  mode: legacy\ndiagnostics:\n  onUse: false\n"), 0644))
	userConfig := filepath.Join(f.userHome, "config.yaml")
	if exists(userConfig) {
		t.Fatal("the user has a config file already: this is not a first run")
	}

	for run := 1; run <= 3; run++ {
		f.run(t, "list").mustSucceed(t, "list")
		settings := f.manager(t).RuntimeEnvContext.UserConfig
		if mode := settings.Path.EffectiveMode(); mode != "legacy" {
			t.Errorf("run %d: path.mode = %q, want the shared legacy", run, mode)
		}
		if settings.Diagnostics.OnUse {
			t.Errorf("run %d: diagnostics.onUse is on, but the shared config switched it off", run)
		}
	}

	user := readFile(t, userConfig)
	mustContain(t, "user config", user, "proxy:", "cache:", "legacyVersionFile:")
	mustNotContain(t, "user config", user, "path:", "diagnostics:", "aliases:")
}
