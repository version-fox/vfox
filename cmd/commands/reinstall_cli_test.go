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

func TestInstallReinstallReplacesTheVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	f.run(t, "install", "alpha@1.0.0").mustSucceed(t, "install")
	marker := filepath.Join(f.shared, "cache", "alpha", "v-1.0.0", "alpha-1.0.0", "changed-in-place")
	must(t, os.WriteFile(marker, nil, 0644))

	// A plain install leaves an installed version as it is.
	out := f.run(t, "install", "alpha@1.0.0").mustSucceed(t, "install again").output()
	mustContain(t, "install", out, "alpha@1.0.0 is already installed")
	if !exists(marker) {
		t.Fatal("a plain install touched the installed version")
	}

	out = f.run(t, "install", "--reinstall", "alpha@1.0.0").mustSucceed(t, "install --reinstall").output()
	mustContain(t, "install --reinstall", out, "alpha@1.0.0")
	if exists(marker) {
		t.Error("--reinstall kept what the SDK changed in place")
	}
}
