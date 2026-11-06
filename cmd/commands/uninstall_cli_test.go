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
