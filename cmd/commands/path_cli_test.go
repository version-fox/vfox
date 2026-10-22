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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	exportedPath = regexp.MustCompile(`export PATH="([^"]*)"`)
	// bash quotes a value that holds ':' as $'...'
	exportedOrig = regexp.MustCompile(`export __VFOX_ORIG_PATH=(?:\$'([^']*)'|([^;']*));`)
)

// exportedBaseline returns the value the script exports as __VFOX_ORIG_PATH.
func exportedBaseline(output string) (string, bool) {
	match := exportedOrig.FindStringSubmatch(output)
	if match == nil {
		return "", false
	}
	return match[1] + match[2], true
}

// pathOf returns the entries of the PATH a command tells the shell to export.
func pathOf(t *testing.T, r cliResult) []string {
	t.Helper()
	match := exportedPath.FindStringSubmatch(r.stdout)
	if match == nil {
		t.Fatalf("no PATH export in:\n%s\nstderr: %s", r.stdout, r.stderr)
	}
	return strings.Split(match[1], ":")
}

// pathFixture is a shell that has vfox active with alpha selected globally, and
// a directory (what path_helper, the Windows machine PATH or `pnpm setup` put
// there) in front of vfox's.
type pathFixture struct {
	*cliFixture
	inherited, vfoxBin string
}

func newPathFixture(t *testing.T) pathFixture {
	t.Helper()
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	must(t, os.WriteFile(filepath.Join(f.userHome, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))
	p := pathFixture{
		cliFixture: f,
		inherited:  filepath.Join(f.root, "usr-local", "bin"),
		vfoxBin:    filepath.Join(f.userHome, "sdks", "alpha", "bin"),
	}
	f.env["PATH"] = strings.Join([]string{p.inherited, p.vfoxBin, f.system}, ":")
	return p
}

func indexOf(entries []string, want string) int {
	for i, entry := range entries {
		if entry == want {
			return i
		}
	}
	return -1
}

func TestEnvPathModeDecidesWhatStaysInFront(t *testing.T) {
	f := newPathFixture(t)
	f.env["__VFOX_ORIG_PATH"] = strings.Join([]string{f.inherited, f.system}, ":")

	// vfox 1.0.x: everything in front of vfox's directory stays there.
	legacy := pathOf(t, f.run(t, "env", "-s", "bash").mustSucceed(t, "env"))
	if want := []string{f.inherited, f.vfoxBin, f.system}; strings.Join(legacy, ":") != strings.Join(want, ":") {
		t.Errorf("legacy PATH = %v, want %v", legacy, want)
	}

	// Changing the setting must show at the next prompt, not when the PATH
	// happens to change: the output of the previous prompt is cached.
	f.writeUserConfig(t, "path:\n  mode: baseline\n")
	baseline := pathOf(t, f.run(t, "env", "-s", "bash").mustSucceed(t, "env"))
	if want := []string{f.vfoxBin, f.inherited, f.system}; strings.Join(baseline, ":") != strings.Join(want, ":") {
		t.Errorf("baseline PATH = %v, want %v (a cached legacy answer was served?)", baseline, want)
	}

	f.writeUserConfig(t, "path:\n  mode: legacy\n")
	back := pathOf(t, f.run(t, "env", "-s", "bash").mustSucceed(t, "env"))
	if strings.Join(back, ":") != strings.Join(legacy, ":") {
		t.Errorf("PATH after switching back = %v, want %v", back, legacy)
	}
}

func TestEnvKeepsWhatWasAddedAfterActivation(t *testing.T) {
	f := newPathFixture(t)
	pnpm := filepath.Join(f.root, "pnpm", "bin")
	f.env["PATH"] = strings.Join([]string{pnpm, f.inherited, f.vfoxBin, f.system}, ":")
	f.env["__VFOX_ORIG_PATH"] = strings.Join([]string{f.inherited, f.system}, ":")
	f.writeUserConfig(t, "path:\n  mode: baseline\n")

	got := pathOf(t, f.run(t, "env", "-s", "bash").mustSucceed(t, "env"))
	if want := []string{pnpm, f.vfoxBin, f.inherited, f.system}; strings.Join(got, ":") != strings.Join(want, ":") {
		t.Errorf("PATH = %v, want %v: pnpm was added after activation and stays in front, the inherited directory does not", got, want)
	}
}

func TestEnvKeepsAnActiveVirtualenvInFrontInEveryMode(t *testing.T) {
	f := newPathFixture(t)
	venv := filepath.Join(f.root, "venv")
	f.env["VIRTUAL_ENV"] = venv
	f.env["PATH"] = strings.Join([]string{filepath.Join(venv, "bin"), f.inherited, f.vfoxBin, f.system}, ":")
	f.env["__VFOX_ORIG_PATH"] = strings.Join([]string{filepath.Join(venv, "bin"), f.inherited, f.system}, ":")

	for _, mode := range []string{"legacy", "baseline", "aggressive"} {
		f.writeUserConfig(t, "path:\n  mode: "+mode+"\n")
		got := pathOf(t, f.run(t, "env", "-s", "bash").mustSucceed(t, "env"))
		if got[0] != filepath.Join(venv, "bin") {
			t.Errorf("%s: PATH = %v, the active virtualenv must stay first", mode, got)
		}
	}
}

func TestEnvAggressiveModeAndPreserve(t *testing.T) {
	f := newPathFixture(t)
	cargo := filepath.Join(f.root, "cargo", "bin")
	f.env["PATH"] = strings.Join([]string{cargo, f.inherited, f.vfoxBin, f.system}, ":")
	f.writeUserConfig(t, fmt.Sprintf("path:\n  mode: aggressive\n  preserve:\n    - %s\n", cargo))

	got := pathOf(t, f.run(t, "env", "-s", "bash").mustSucceed(t, "env"))
	if want := []string{cargo, f.vfoxBin, f.inherited, f.system}; strings.Join(got, ":") != strings.Join(want, ":") {
		t.Errorf("PATH = %v, want %v", got, want)
	}
}

func TestEnvWithoutABaselineBehavesLikeLegacyInEveryMode(t *testing.T) {
	f := newPathFixture(t)
	// An old activation script never recorded one.
	for _, mode := range []string{"baseline"} {
		f.writeUserConfig(t, "path:\n  mode: "+mode+"\n")
		got := pathOf(t, f.run(t, "env", "-s", "bash").mustSucceed(t, "env"))
		if got[0] != f.inherited {
			t.Errorf("%s without a baseline: PATH = %v, want legacy order", mode, got)
		}
	}
}

func TestActivateRecordsTheBaselineOncePerSession(t *testing.T) {
	f := newPathFixture(t)
	f.env["PATH"] = strings.Join([]string{f.inherited, f.system}, ":") // nothing of vfox's yet
	delete(f.env, "__VFOX_SHELL")

	// A fresh shell: the PATH it found is recorded.
	r := f.run(t, "activate", "bash").mustSucceed(t, "activate")
	if got, ok := exportedBaseline(r.stdout); !ok || got != f.inherited+":"+f.system {
		t.Fatalf("baseline export = %q (%v), want %s:%s\n%s", got, ok, f.inherited, f.system, r.stdout)
	}

	// The same session running activate again (a re-sourced rc file) keeps the
	// baseline it has, so that what was added since is not mistaken for inherited.
	f.env["__VFOX_ORIG_PATH"] = "/recorded/earlier"
	f.env["__VFOX_PID"] = fmt.Sprint(os.Getpid()) // the child's parent: the same session
	r = f.run(t, "activate", "bash").mustSucceed(t, "activate")
	mustNotContain(t, "activate in the same session", r.stdout, "export __VFOX_ORIG_PATH=")

	// A shell started from another shell inherited the variable, not the session.
	f.env["__VFOX_PID"] = "1"
	r = f.run(t, "activate", "bash").mustSucceed(t, "activate")
	if got, ok := exportedBaseline(r.stdout); !ok || got != f.inherited+":"+f.system {
		t.Fatalf("an inherited session must record its own baseline, got %q (%v)\n%s", got, ok, r.stdout)
	}
}

func TestActivateBaselineLeavesOutVfoxDirectories(t *testing.T) {
	f := newPathFixture(t) // PATH: inherited, vfox, system
	delete(f.env, "__VFOX_SHELL")
	r := f.run(t, "activate", "bash").mustSucceed(t, "activate")
	if got, ok := exportedBaseline(r.stdout); !ok || got != f.inherited+":"+f.system {
		t.Fatalf("baseline = %q (%v): vfox's own directory is not part of what the shell brought", got, ok)
	}
}

func TestActivateAppliesThePathMode(t *testing.T) {
	// The workaround for #337/#732 was to drop vfox's directories from PATH
	// before activating, so that nothing counted as "in front of vfox".
	// Activation itself now knows that everything it finds was already there.
	f := newPathFixture(t)
	delete(f.env, "__VFOX_SHELL")

	legacy := pathOf(t, f.run(t, "activate", "bash").mustSucceed(t, "activate"))
	if legacy[0] != f.inherited {
		t.Errorf("legacy: PATH = %v, want the inherited directory first as before", legacy)
	}
	f.writeUserConfig(t, "path:\n  mode: baseline\n")
	baseline := pathOf(t, f.run(t, "activate", "bash").mustSucceed(t, "activate"))
	if baseline[0] != f.vfoxBin || indexOf(baseline, f.inherited) < 1 {
		t.Errorf("baseline: PATH = %v, want vfox's directory first and the inherited one behind it", baseline)
	}
}

func TestActivateInAnIDEDoesNotLeakTheBaseline(t *testing.T) {
	f := newPathFixture(t)
	delete(f.env, "__VFOX_SHELL")
	f.env["VSCODE_RESOLVING_ENVIRONMENT"] = "1"
	r := f.run(t, "activate", "bash").mustSucceed(t, "activate")
	mustContain(t, "activate", r.stdout, "unset __VFOX_ORIG_PATH;")
	mustNotContain(t, "activate", r.stdout, "export __VFOX_ORIG_PATH=")
}

// The documentation tells people to use `vfox config` for the new settings.
func TestConfigCommandHandlesTheNewSettings(t *testing.T) {
	f := newCLIFixture(t, false)
	userConfig := filepath.Join(f.userHome, "config.yaml")

	if out := f.run(t, "config", "path.mode").mustSucceed(t, "config path.mode").stdout; strings.TrimSpace(out) != "legacy" {
		t.Errorf("default path.mode = %q", out)
	}
	if out := f.run(t, "config", "diagnostics.onUse").mustSucceed(t, "config diagnostics.onUse").stdout; strings.TrimSpace(out) != "true" {
		t.Errorf("default diagnostics.onUse = %q", out)
	}

	f.run(t, "config", "path.mode", "baseline").mustSucceed(t, "config path.mode baseline")
	f.run(t, "config", "diagnostics.onUse", "false").mustSucceed(t, "config diagnostics.onUse false")
	saved := readFile(t, userConfig)
	mustContain(t, "config.yaml", saved, "mode: baseline", "onUse: false")

	if out := f.run(t, "config", "path.mode").mustSucceed(t, "config path.mode").stdout; strings.TrimSpace(out) != "baseline" {
		t.Errorf("path.mode = %q after setting it", out)
	}
	list := f.run(t, "config", "--list").mustSucceed(t, "config --list").stdout
	mustContain(t, "config --list", list, "path.mode = baseline", "diagnostics.onUse = false", "path.preserve = []", "diagnostics.ignore = []")

	// What was set is what the other commands see.
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	f.env["PATH"] = strings.Join([]string{filepath.Join(f.root, "pnpm", "bin"), filepath.Join(f.userHome, "sdks", "alpha", "bin"), f.system}, ":")
	must(t, os.MkdirAll(filepath.Join(f.root, "pnpm", "bin"), 0755))
	must(t, os.WriteFile(filepath.Join(f.root, "pnpm", "bin", "alphatool"), []byte("#!/bin/sh\n"), 0755))
	mustNotContain(t, "use with diagnostics.onUse false", f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use").output(), "comes before vfox")

	f.run(t, "config", "--unset", "diagnostics.onUse").mustSucceed(t, "config --unset diagnostics.onUse")
	f.run(t, "config", "--unset", "path.mode").mustSucceed(t, "config --unset path.mode")
	list = f.run(t, "config", "--list").mustSucceed(t, "config --list").stdout
	mustContain(t, "config --list after unset", list, "path.mode = legacy", "diagnostics.onUse = true")
	mustContain(t, "use with the default settings", f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use").output(), "comes before vfox")
}
