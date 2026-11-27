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
 *
 */

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIncludeAndLibPathOfAnInstalledVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	home := f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	must(t, os.MkdirAll(filepath.Join(home, "include"), 0755))
	must(t, os.MkdirAll(filepath.Join(home, "lib"), 0755))

	// A Makefile gets the path and nothing else.
	if out := f.run(t, "include-path", "alpha@1.0.0").mustSucceed(t, "include-path").stdout; out != filepath.Join(home, "include")+"\n" {
		t.Errorf("include-path = %q, want the include directory alone", out)
	}
	if out := f.run(t, "lib-path", "alpha@1.0.0").mustSucceed(t, "lib-path").stdout; out != filepath.Join(home, "lib")+"\n" {
		t.Errorf("lib-path = %q, want the lib directory alone", out)
	}
}

func TestIncludePathOfTheCurrentVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	home := f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	must(t, os.MkdirAll(filepath.Join(home, "include"), 0755))

	r := f.run(t, "include-path", "alpha").mustFail(t, "include-path with no current version")
	mustContain(t, "stderr", r.stderr, "no current version of alpha")

	f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use -g alpha@1.0.0")
	if out := strings.TrimSpace(f.run(t, "include-path", "alpha").mustSucceed(t, "include-path alpha").stdout); out != filepath.Join(home, "include") {
		t.Errorf("include-path alpha = %q, want the directory of the current version", out)
	}
}

func TestIncludePathOfALinkedVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	external := f.external(t, "alphatool")
	must(t, os.MkdirAll(filepath.Join(external, "include"), 0755))
	f.run(t, "link", "alpha@sys", external).mustSucceed(t, "link")

	// The plugin reports the directory under the version's own path, which is a
	// link to the linked SDK, so the path printed leads into that SDK.
	home := filepath.Join(f.shared, "cache", "alpha", "v-sys", "alpha-sys")
	if out := strings.TrimSpace(f.run(t, "include-path", "alpha@sys").mustSucceed(t, "include-path of a linked version").stdout); out != filepath.Join(home, "include") {
		t.Errorf("include-path alpha@sys = %q, want %q", out, filepath.Join(home, "include"))
	}
}

func TestPathCommandsUnderstandAliasesAndTheConventionalV(t *testing.T) {
	f := newCLIFixture(t, false)
	home := f.installOnDisk(t, "alpha", "21.0.2+13", "alphatool")
	must(t, os.MkdirAll(filepath.Join(home, "include"), 0755))
	f.run(t, "alias", "set", "alpha", "j21", "21.0.2+13").mustSucceed(t, "alias set")

	want := filepath.Join(home, "include")
	for _, arg := range []string{"alpha@v21.0.2+13", "alpha@j21"} {
		if out := strings.TrimSpace(f.run(t, "include-path", arg).mustSucceed(t, "include-path "+arg).stdout); out != want {
			t.Errorf("include-path %s = %q, want %q", arg, out, want)
		}
	}
}

func TestPathCommandsReportWhatIsMissing(t *testing.T) {
	f := newCLIFixture(t, false)

	r := f.run(t, "include-path", "alpha@nope").mustFail(t, "include-path of a version that is not installed")
	mustContain(t, "stderr", r.stderr, "alpha@nope is not installed")

	f.installOnDisk(t, "plain", "1.0.0")
	r = f.run(t, "lib-path", "plain@1.0.0").mustFail(t, "lib-path of a plugin without the hook")
	mustContain(t, "stderr", r.stderr, "plain@1.0.0 has no lib directory")

	// The plugin reports include, but the directory is not on disk.
	f.installOnDisk(t, "alpha", "2.0.0", "alphatool")
	r = f.run(t, "include-path", "alpha@2.0.0").mustFail(t, "include-path of a directory that is not there")
	mustContain(t, "stderr", r.stderr, "not an existing directory")

	f.run(t, "include-path").mustFail(t, "include-path without an SDK")
}

func TestPathCommandsRejectMalformedAndUnknownArguments(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")

	r := f.run(t, "include-path", "alpha@").mustFail(t, "include-path with an empty version")
	mustContain(t, "stderr", r.stderr, "invalid arguments")

	r = f.run(t, "lib-path", "nosuch").mustFail(t, "lib-path of an SDK without a plugin")
	mustContain(t, "stderr", r.stderr, "nosuch not supported")

	r = f.run(t, "include-path", "nosuch@1.0.0").mustFail(t, "include-path of an SDK without a plugin")
	mustContain(t, "stderr", r.stderr, "nosuch not supported")

	// vfox cannot load its settings, so it cannot find any SDK either.
	f.writeUserConfig(t, "proxy: [\n")
	r = f.run(t, "include-path", "alpha@1.0.0").mustFail(t, "include-path with a broken config.yaml")
	mustContain(t, "stderr", r.stderr, "load config failed")
}
