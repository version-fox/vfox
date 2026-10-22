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
	"regexp"
	"testing"

	"github.com/version-fox/vfox/internal/sdk"
)

// A leading "v" is the usual way to spell a version (v20.1.0), but the names
// users give to versions (a linked "vendor", an alias "v1") can start with one
// as well. What exists under the name as typed wins; only otherwise is the "v"
// dropped.
type versionNameFixture struct {
	*cliFixture
}

func newVersionNameFixture(t *testing.T) versionNameFixture {
	t.Helper()
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	f.installOnDisk(t, "alpha", "2.0.0", "alphatool")
	f.run(t, "link", "alpha@vendor", f.external(t, "alphatool")).mustSucceed(t, "link alpha@vendor")
	f.run(t, "alias", "set", "alpha", "v1", "2.0.0").mustSucceed(t, "alias set alpha v1 2.0.0")
	return versionNameFixture{f}
}

var selectedAlpha = regexp.MustCompile(`alpha = "([^"]*)"`)

func TestUseTakesTheNameAsTypedBeforeDroppingAV(t *testing.T) {
	f := newVersionNameFixture(t)

	for _, tt := range []struct{ arg, want, why string }{
		{"alpha@v1", "2.0.0", "an alias named v1 must not become the prefix 1, which would select 1.0.0"},
		{"alpha@vendor", "vendor", "a linked version named vendor must not become endor"},
		{"alpha@v1.0.0", "1.0.0", "the usual way to spell a version still works"},
		{"alpha@v2.0.0", "2.0.0", "the usual way to spell a version still works"},
		{"alpha@1.0.0", "1.0.0", "a version without the v"},
	} {
		f.run(t, "use", "-g", tt.arg).mustSucceed(t, "use -g "+tt.arg)
		match := selectedAlpha.FindStringSubmatch(readFile(t, filepath.Join(f.userHome, ".vfox.toml")))
		if match == nil || match[1] != tt.want {
			t.Errorf("use -g %s selected %v, want %s: %s", tt.arg, match, tt.want, tt.why)
		}
	}
}

func TestExecTakesTheNameAsTypedBeforeDroppingAV(t *testing.T) {
	f := newVersionNameFixture(t)
	manager := f.manager(t)

	for _, tt := range []struct {
		arg  string
		want sdk.Version
	}{
		{"alpha@v1", "2.0.0"},
		{"alpha@vendor", "vendor"},
		{"alpha@v1.0.0", "1.0.0"},
		{"alpha@1.0.0", "1.0.0"},
		{"alpha@v9.9.9", "9.9.9"}, // nothing by that name: the v goes, as it always did
		{"alpha@vv", "v"},
	} {
		spec, err := parseExecSDKSpec(tt.arg)
		if err != nil {
			t.Fatal(err)
		}
		got, err := resolveExecSDKVersion(manager, spec)
		if err != nil || got != tt.want {
			t.Errorf("exec %s resolved to (%q, %v), want %q", tt.arg, got, err, tt.want)
		}
	}
}

func TestDoctorTakesTheNameAsTypedBeforeDroppingAV(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "3.0.0") // a version without a command, which doctor notices
	// A linked SDK that is gone, under a name that starts with a v.
	external := f.external(t, "alphatool")
	f.run(t, "link", "alpha@vendor", external).mustSucceed(t, "link alpha@vendor")
	must(t, os.RemoveAll(external))

	for _, tt := range []struct{ arg, finding string }{
		{"alpha@vendor", "error  sdk.link-dangling  alpha@vendor"},
		{"alpha@3.0.0", "warn   sdk.payload-empty  alpha@3.0.0"},
		{"alpha@v3.0.0", "warn   sdk.payload-empty  alpha@3.0.0"},
	} {
		r := f.run(t, "doctor", tt.arg)
		mustContain(t, "doctor "+tt.arg, r.stdout, tt.finding)
	}
}

func TestInstallKnowsALinkedVersionThatStartsWithAV(t *testing.T) {
	f := newVersionNameFixture(t)

	out := f.run(t, "install", "alpha@vendor").mustSucceed(t, "install alpha@vendor").output()
	mustContain(t, "install", out, "alpha@vendor is already installed")
}
