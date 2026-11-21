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

// gammaPlugin resolves 3.12 to 3.12.7, the way plugins resolve a partial version.
const gammaPlugin = `PLUGIN = {name="gamma", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="3.12.7"}} end
function PLUGIN:PreInstall(ctx)
  if ctx.version == "3.12" then return {version="3.12.7"} end
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`

// #668: a second copy of a version, kept under a name of its own, beside the
// installation of the version it copies.
func TestInstallAsKeepsACopyBesideTheVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "gamma", gammaPlugin)

	f.run(t, "install", "gamma@3.12").mustSucceed(t, "install gamma@3.12")
	out := f.run(t, "install", "--as", "3.12-db", "gamma@3.12").mustSucceed(t, "install --as").output()
	mustContain(t, "install --as", out, "gamma@3.12-db", "vfox use gamma@3.12-db")

	list := f.run(t, "list", "gamma").mustSucceed(t, "list gamma").output()
	mustContain(t, "list gamma", list, "3.12.7", "3.12-db (3.12.7)")
}

func TestInstallAsSwitchesWithUse(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "gamma", gammaPlugin)

	out := f.run(t, "install", "-u", "--as", "3.12-db", "gamma@3.12").mustSucceed(t, "install -u --as").output()
	mustContain(t, "install -u --as", out, "Now using", "gamma@3.12-db")
	mustContain(t, "session config", readFile(t, filepath.Join(f.userHome, "tmp", "test-session", ".vfox.toml")), `gamma = "3.12-db"`)
}

// `vfox use` picks the installation a version names, not a copy whose name merely
// starts with the same digits.
func TestUseOfTheVersionIgnoresItsCopies(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "gamma", gammaPlugin)

	f.run(t, "install", "--as", "3.12-db", "gamma@3.12").mustSucceed(t, "install --as")
	f.run(t, "install", "gamma@3.12").mustSucceed(t, "install gamma@3.12")
	// The plugin installs no commands, so give the plain version one for use to find.
	bin := filepath.Join(f.shared, "cache", "gamma", "v-3.12.7", "gamma-3.12.7", "bin")
	must(t, os.MkdirAll(bin, 0755))
	writeCommand(t, bin, "gamma")

	out := f.run(t, "use", "-g", "gamma@3.12").mustSucceed(t, "use -g gamma@3.12").output()
	mustContain(t, "use -g gamma@3.12", out, "Now using", "gamma@3.12.7")
	mustNotContain(t, "use -g gamma@3.12", out, "3.12-db")
}

func TestInstallAsRefusals(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "gamma", gammaPlugin)

	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{"with --all", []string{"install", "--all", "--as", "3.12-db"}, "--as cannot be combined with --all"},
		{"with --reinstall", []string{"install", "--reinstall", "--as", "3.12-db", "gamma@3.12"}, "--as cannot be combined with --reinstall"},
		{"unusable name", []string{"install", "--as", "a/b", "gamma@3.12"}, `"a/b" is not a usable name for --as`},
		{"name latest", []string{"install", "--as", "latest", "gamma@3.12"}, `"latest" is not a usable name for --as`},
		{"no version", []string{"install", "--as", "3.12-db", "gamma"}, "install --as needs a version for gamma"},
		{"two installations", []string{"install", "--as", "3.12-db", "gamma@3.12", "gamma@3.12"}, "--as names one installation"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := f.run(t, tt.args...).mustFail(t, "install --as")
			mustContain(t, "error", r.output(), tt.want)
		})
	}
}

func TestInstallAsRefusesANameInUse(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "gamma", gammaPlugin)

	f.run(t, "install", "--as", "3.12-db", "gamma@3.12").mustSucceed(t, "first copy")
	r := f.run(t, "install", "--as", "3.12-db", "gamma@3.12").mustFail(t, "second copy under the same name")
	mustContain(t, "error", r.output(), "gamma@3.12-db is already installed", "choose another name")
}

func TestInstallAsRefusesTheVersionItself(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "gamma", gammaPlugin)

	r := f.run(t, "install", "--as", "3.12.7", "gamma@3.12").mustFail(t, "copy named after its own version")
	mustContain(t, "error", r.output(), "cannot be named 3.12.7")
}

// A copy the SDK changed in place is reinstalled under its own name.
func TestInstallReinstallOfACopy(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "gamma", gammaPlugin)

	f.run(t, "install", "--as", "3.12-db", "gamma@3.12").mustSucceed(t, "first copy")
	out := f.run(t, "install", "--reinstall", "gamma@3.12-db").mustSucceed(t, "install --reinstall copy").output()
	mustContain(t, "install --reinstall copy", out, "gamma@3.12-db")
}
