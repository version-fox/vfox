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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type infoJSON struct {
	Name        string
	Version     string
	Installed   bool
	Origin      string
	Target      string
	Home        string
	PackagePath string
	Additions   []struct{ Name, Version, Path string }
	Env         *struct {
		Paths     []string
		Variables map[string]string
	}
	Receipt *struct {
		Origin, Target, SDK, Version, Requested string
	}
}

func parseInfo(t *testing.T, r cliResult) infoJSON {
	t.Helper()
	var info infoJSON
	if err := json.Unmarshal([]byte(r.stdout), &info); err != nil {
		t.Fatalf("info --json printed %q: %v", r.stdout, err)
	}
	return info
}

func TestInfoForAVersionThatIsNotThere(t *testing.T) {
	f := newCLIFixture(t, false)
	if out := strings.TrimSpace(f.run(t, "info", "alpha@nope").mustSucceed(t, "info").stdout); out != "notfound" {
		t.Errorf("info = %q, want the 'notfound' it always printed", out)
	}
	if out := f.run(t, "info", "--format", "{{.Home}}|{{.Path}}", "alpha@nope").mustSucceed(t, "info --format").stdout; out != "notfound|notfound" {
		t.Errorf("info --format = %q", out)
	}
	info := parseInfo(t, f.run(t, "info", "--json", "alpha@nope").mustSucceed(t, "info --json"))
	if info.Installed || info.Name != "alpha" || info.Version != "nope" {
		t.Errorf("info = %+v", info)
	}
	if out := strings.TrimSpace(f.run(t, "info", "nosuch@1").mustSucceed(t, "info").stdout); out != "notfound" {
		t.Errorf("an SDK without a plugin: %q", out)
	}
	if out := f.run(t, "info", "--format", "{{.Name}}|{{.Home}}|{{.Path}}", "nosuch@1").mustSucceed(t, "info --format").stdout; out != "nosuch|notfound|notfound" {
		t.Errorf("an SDK without a plugin, as a template: %q", out)
	}
	if info := parseInfo(t, f.run(t, "info", "--json", "nosuch@1").mustSucceed(t, "info --json")); info.Installed || info.Name != "nosuch" || info.Version != "1" {
		t.Errorf("an SDK without a plugin, as JSON: %+v", info)
	}
	f.run(t, "info", "--json", "--format", "{{.Home}}", "alpha@1").mustFail(t, "info with both --json and --format")
}

func TestInfoJSONOfAPlugin(t *testing.T) {
	f := newCLIFixture(t, false)
	var plugin struct{ Name, Version, Description, InstallPath string }
	r := f.run(t, "info", "--json", "alpha").mustSucceed(t, "info --json alpha")
	if err := json.Unmarshal([]byte(r.stdout), &plugin); err != nil {
		t.Fatalf("%q: %v", r.stdout, err)
	}
	if plugin.Name != "alpha" || plugin.Version != "0.0.1" || plugin.Description != "test" {
		t.Errorf("plugin = %+v", plugin)
	}
}

func TestInfoOfAnInstalledVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	home := f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	pkg := filepath.Dir(home)

	// Without options it prints the version directory, as it always did.
	if out := strings.TrimSpace(f.run(t, "info", "alpha@1.0.0").mustSucceed(t, "info").stdout); out != pkg {
		t.Errorf("info = %q, want the version directory %q", out, pkg)
	}
	// Home is the main runtime directory, the one JAVA_HOME and the like want.
	out := f.run(t, "info", "--format", "{{.Home}}|{{.Path}}", "alpha@1.0.0").mustSucceed(t, "info --format").stdout
	if want := home + "|" + pkg; out != want {
		t.Errorf("info --format = %q, want %q", out, want)
	}

	info := parseInfo(t, f.run(t, "info", "--json", "alpha@1.0.0").mustSucceed(t, "info --json"))
	if !info.Installed || info.Name != "alpha" || info.Version != "1.0.0" || info.Home != home || info.PackagePath != pkg {
		t.Errorf("info = %+v", info)
	}
	if info.Env == nil || len(info.Env.Paths) != 1 || info.Env.Paths[0] != filepath.Join(home, "bin") {
		t.Errorf("env = %+v, want what the plugin puts on PATH", info.Env)
	}
}

// What #669 describes: an SDK that was linked into the install directory by
// hand. `list` used to show it while `info` and `use` said it was not there.
func TestAHandMadeLinkIsFoundByEveryCommand(t *testing.T) {
	f := newCLIFixture(t, false)
	external := f.external(t, "alphatool")
	pkg := filepath.Join(f.shared, "cache", "alpha", "v-ext")
	home := filepath.Join(pkg, "alpha-ext")
	must(t, os.MkdirAll(pkg, 0755))
	must(t, os.Symlink(external, home))

	mustContain(t, "list", f.run(t, "list", "alpha").mustSucceed(t, "list").output(), "ext")
	if out := strings.TrimSpace(f.run(t, "info", "alpha@ext").mustSucceed(t, "info").stdout); out != pkg {
		t.Errorf("info = %q, want the version directory %q", out, pkg)
	}
	if out := f.run(t, "info", "--format", "{{.Home}}", "alpha@ext").mustSucceed(t, "info --format").stdout; out != home {
		t.Errorf("info --format = %q, want %q", out, home)
	}

	out := f.run(t, "use", "-g", "alpha@ext").mustSucceed(t, "use").output()
	mustContain(t, "use", out, "Now using", "alpha@ext")
	mustContain(t, "global config", readFile(t, filepath.Join(f.userHome, ".vfox.toml")), `alpha = "ext"`)
	if !exists(filepath.Join(f.userHome, "sdks", "alpha", "bin", "alphatool")) {
		t.Error("the global link does not reach the linked SDK")
	}
}

// richPlugin puts a variable in the environment next to PATH.
const richPlugin = `PLUGIN = {name="rich", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}, {key="RICH_HOME", value=ctx.path}} end
`

func TestInfoJSONListsTheAdditionalRuntimesAndTheVariables(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "rich", richPlugin)
	home := f.installOnDisk(t, "rich", "1.0.0", "richtool")
	extra := filepath.Join(filepath.Dir(home), "add-extra-1.0.0")
	must(t, os.MkdirAll(extra, 0755))

	info := parseInfo(t, f.run(t, "info", "--json", "rich@1.0.0").mustSucceed(t, "info --json"))
	if len(info.Additions) != 1 || info.Additions[0].Path != extra {
		t.Errorf("additions = %+v, want the directory of the additional runtime", info.Additions)
	}
	if info.Env == nil || info.Env.Variables["RICH_HOME"] != home || len(info.Env.Paths) != 1 || info.Env.Paths[0] != filepath.Join(home, "bin") {
		t.Errorf("env = %+v, want the PATH entry and the variable the plugin sets", info.Env)
	}
}
