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

// The scenario of #706 and #669: a JDK that is already on the machine.
func TestLinkCommandEndToEnd(t *testing.T) {
	f := newCLIFixture(t, false)
	external := f.external(t, "alphatool")
	data := filepath.Join(external, "data")
	must(t, os.WriteFile(data, []byte("precious"), 0644))
	home := filepath.Join(f.shared, "cache", "alpha", "v-sys", "alpha-sys")
	pkg := filepath.Dir(home)

	out := f.run(t, "link", "alpha@sys", external).mustSucceed(t, "link").output()
	mustContain(t, "link output", out, "Linked", "alpha@sys", external, "vfox use alpha@sys")

	t.Run("list marks it", func(t *testing.T) {
		out := f.run(t, "list", "alpha").mustSucceed(t, "list").output()
		mustContain(t, "list", out, "sys (linked → "+external+")")
	})

	t.Run("info tells where it is", func(t *testing.T) {
		info := parseInfo(t, f.run(t, "info", "--json", "alpha@sys").mustSucceed(t, "info --json"))
		if !info.Installed || info.Name != "alpha" || info.Version != "sys" || info.Origin != "linked" ||
			info.Target != external || info.Home != home || info.PackagePath != pkg {
			t.Errorf("info = %+v", info)
		}
		if info.Env == nil || len(info.Env.Paths) != 1 || info.Env.Paths[0] != filepath.Join(home, "bin") {
			t.Errorf("env = %+v, want what the plugin puts on PATH", info.Env)
		}
		if info.Receipt == nil || info.Receipt.Origin != "linked" || info.Receipt.Target != external || info.Receipt.Requested != "sys" {
			t.Errorf("receipt = %+v", info.Receipt)
		}

		// The template gets Home (what JAVA_HOME wants) next to the Path scripts already use.
		out := f.run(t, "info", "--format", "{{.Home}}|{{.Path}}|{{.Origin}}|{{.Target}}", "alpha@sys").mustSucceed(t, "info --format").stdout
		if want := strings.Join([]string{home, pkg, "linked", external}, "|"); out != want {
			t.Errorf("info --format = %q, want %q", out, want)
		}
		// Without options it prints what it always printed.
		if out := strings.TrimSpace(f.run(t, "info", "alpha@sys").mustSucceed(t, "info").stdout); out != pkg {
			t.Errorf("info = %q, want the version directory %q", out, pkg)
		}
	})

	t.Run("installing it again downloads nothing", func(t *testing.T) {
		out := f.run(t, "install", "alpha@sys").mustSucceed(t, "install").output()
		mustContain(t, "install", out, "alpha@sys is already installed")
	})

	t.Run("a healthy link is no problem", func(t *testing.T) {
		out := f.run(t, "doctor").mustSucceed(t, "doctor").output()
		mustContain(t, "doctor", out, "No problems found.")
	})

	t.Run("uninstalling removes the link and nothing else", func(t *testing.T) {
		out := f.run(t, "uninstall", "alpha@sys").mustSucceed(t, "uninstall").output()
		mustContain(t, "uninstall", out, "Unlinked", "alpha@sys", "left untouched")
		if content, err := os.ReadFile(data); err != nil || string(content) != "precious" {
			t.Errorf("the linked SDK's file: %q, %v", content, err)
		}
		if !exists(filepath.Join(external, "bin", "alphatool")) {
			t.Error("the linked SDK lost a command")
		}
		if exists(pkg) {
			t.Error("the version is still there")
		}
	})
}

func TestLinkCommandRefusesWhatItCannotDo(t *testing.T) {
	f := newCLIFixture(t, false)
	external := f.external(t, "alphatool")
	notAnSDK := f.external(t) // no bin directory

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"no version", []string{"link", "alpha", external}, "format: <sdk-name>@<version>"},
		{"no path", []string{"link", "alpha@sys"}, "usage: vfox link"},
		{"too many", []string{"link", "alpha@sys", external, "extra"}, "usage: vfox link"},
		{"no plugin", []string{"link", "nosuch@1", external}, "vfox add nosuch"},
		{"reserved name", []string{"link", "alpha@latest", external}, "not a usable version name"},
		{"name that is a path", []string{"link", "alpha@../x", external}, "not a usable version name"},
		{"missing directory", []string{"link", "alpha@sys", filepath.Join(external, "missing")}, "cannot link"},
		{"not an SDK", []string{"link", "alpha@sys", notAnSDK}, "does not look like"},
		{"detect without an SDK", []string{"link", "--detect"}, "usage: vfox link --detect"},
		{"reinstall with --all", []string{"install", "--all", "--reinstall"}, "--reinstall cannot be combined with --all"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := f.run(t, tc.args...).mustFail(t, strings.Join(tc.args, " "))
			mustContain(t, "error", r.output(), tc.want)
		})
	}
	// None of the refusals may leave a version behind.
	entries, _ := os.ReadDir(filepath.Join(f.shared, "cache", "alpha"))
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "v-") {
			t.Errorf("a refused link left %s behind", entry.Name())
		}
	}
	f.run(t, "list", "alpha").mustFail(t, "list of an SDK without versions")
}

func TestLinkCommandForce(t *testing.T) {
	f := newCLIFixture(t, false)
	first, second := f.external(t, "alphatool"), f.external(t, "alphatool")
	f.run(t, "link", "alpha@sys", first).mustSucceed(t, "link")

	r := f.run(t, "link", "alpha@sys", second).mustFail(t, "link again")
	mustContain(t, "error", r.output(), "already linked", "--force")

	f.run(t, "link", "--force", "alpha@sys", second).mustSucceed(t, "link --force")
	out := f.run(t, "list", "alpha").mustSucceed(t, "list").output()
	mustContain(t, "list", out, "(linked → "+second+")")
	if !exists(filepath.Join(first, "bin", "alphatool")) {
		t.Error("the old target must stay as it was")
	}

	// A version vfox installed itself is never replaced, --force or not.
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	r = f.run(t, "link", "--force", "alpha@1.0.0", first).mustFail(t, "link over an installed version")
	mustContain(t, "error", r.output(), "already installed")
}

func TestLinkDetect(t *testing.T) {
	f := newCLIFixture(t, false)
	found := f.external(t, "alphatool")
	f.env["ALPHA_DETECT_DIR"] = found

	t.Run("it asks first, and cannot in a script", func(t *testing.T) {
		r := f.run(t, "link", "--detect", "alpha").mustFail(t, "link --detect without -y")
		mustContain(t, "output", r.output(), "found-1", found, "from the test", "-y")
		if exists(filepath.Join(f.shared, "cache", "alpha", "v-found-1")) {
			t.Error("nothing may be linked without a yes")
		}
	})
	t.Run("it links what the plugin found", func(t *testing.T) {
		out := f.run(t, "link", "--detect", "-y", "alpha").mustSucceed(t, "link --detect -y").output()
		mustContain(t, "output", out, "Linked", "alpha@found-1")
		mustContain(t, "list", f.run(t, "list", "alpha").mustSucceed(t, "list").output(), "found-1 (linked → "+found+")")
	})
	t.Run("a second run finds nothing new", func(t *testing.T) {
		out := f.run(t, "link", "--detect", "-y", "alpha").mustSucceed(t, "link --detect -y").output()
		mustContain(t, "output", out, "Skipping found-1", "already installed", "No new alpha installations found.")
	})
	t.Run("the same directory is not linked under a second name", func(t *testing.T) {
		f.installOnDisk(t, "alpha", "unrelated", "alphatool") // does not matter, just a second version
		out := f.run(t, "link", "--detect", "-y", "alpha").mustSucceed(t, "link --detect -y").output()
		mustContain(t, "output", out, "No new alpha installations found.")
	})
	t.Run("a plugin that cannot detect says so", func(t *testing.T) {
		r := f.run(t, "link", "--detect", "-y", "plain").mustFail(t, "link --detect on a plugin without the hook")
		mustContain(t, "output", r.output(), "cannot detect")
	})
	t.Run("what cannot be linked is reported", func(t *testing.T) {
		f2 := newCLIFixture(t, false)
		f2.env["ALPHA_DETECT_DIR"] = f2.external(t) // not an SDK
		r := f2.run(t, "link", "--detect", "-y", "alpha").mustFail(t, "link --detect of a directory that is not an SDK")
		mustContain(t, "output", r.output(), "Could not link alpha@found-1", "does not look like")
	})
	t.Run("a plugin that finds nothing", func(t *testing.T) {
		f3 := newCLIFixture(t, false)
		out := f3.run(t, "link", "--detect", "-y", "alpha").mustSucceed(t, "link --detect with nothing to find").output()
		mustContain(t, "output", out, "No new alpha installations found.")
	})
}

func TestLinkedVersionCanBeSelected(t *testing.T) {
	f := newCLIFixture(t, false)
	external := f.external(t, "alphatool")
	f.run(t, "link", "alpha@sys", external).mustSucceed(t, "link")

	out := f.run(t, "use", "-g", "alpha@sys").mustSucceed(t, "use").output()
	mustContain(t, "use", out, "Now using", "alpha@sys")
	mustContain(t, "global config", readFile(t, filepath.Join(f.userHome, ".vfox.toml")), `alpha = "sys"`)

	// The shell gets the linked SDK's commands through the global link.
	link := filepath.Join(f.userHome, "sdks", "alpha")
	if !exists(filepath.Join(link, "bin", "alphatool")) {
		t.Errorf("%s does not reach the linked SDK", link)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("global link: %v, %v", info, err)
	}

	// And `list` says it is the one in use.
	mustContain(t, "list", f.run(t, "list", "alpha").mustSucceed(t, "list").output(), "sys (linked → "+external+")", "<— current")
}

func TestListShowsEveryInstalledVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	external := f.external(t, "alphatool")
	f.run(t, "link", "alpha@sys", external).mustSucceed(t, "link")

	out := f.run(t, "list").mustSucceed(t, "list").output()
	mustContain(t, "list", out, "alpha", "1.0.0", "sys (linked → "+external+")")
}
