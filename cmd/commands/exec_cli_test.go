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
	"testing"

	"github.com/version-fox/vfox/internal/sdk"
)

// `vfox exec cmake@4.5` must run the installed 4.5.0, as `vfox use cmake@4.5`
// selects it, instead of trying to install a version called 4.5.
func TestExecResolvesAPartialVersionToTheInstalledOne(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "4.5.0", "alphatool")
	f.installOnDisk(t, "alpha", "4.6.1", "alphatool")
	manager := f.manager(t)

	for _, tt := range []struct {
		typed, want sdk.Version
		why         string
	}{
		{"4.5.0", "4.5.0", "an exact version is taken as is"},
		{"4.5", "4.5.0", "a partial version stands for the installed version it extends"},
		{"4", "4.6.1", "the newest installed version wins, as with use"},
		{"4.7", "4.7", "nothing installed extends it, so it is left to be installed"},
		{"4.5.0-rc1", "4.5.0-rc1", "a longer name is not a prefix of the installed one"},
	} {
		got, err := resolveExecSDKVersion(manager, execSDKSpec{Name: "alpha", Version: tt.typed})
		if err != nil || got != tt.want {
			t.Errorf("exec alpha@%s resolved to (%q, %v), want %q: %s", tt.typed, got, err, tt.want, tt.why)
		}
	}
}
