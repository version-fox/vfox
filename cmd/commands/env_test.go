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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
)

// Run the real command in a fresh process, as a shell hook would. This also
// isolates CLI state, environment variables and SDK initialization between runs.
func TestEnvCommandProcess(t *testing.T) {
	if os.Getenv("VFOX_TEST_ENV_COMMAND") != "1" {
		return
	}
	app := &cli.Command{Name: "vfox", Commands: []*cli.Command{Env}}
	shellName := os.Getenv("VFOX_TEST_TARGET_SHELL")
	if shellName == "" {
		shellName = "zsh"
	}
	if err := app.Run(context.Background(), []string{"vfox", "env", "-s", shellName}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

type envCommandFixture struct {
	home, work, state, system string
	names                     []string
}

func newEnvCommandFixture(t *testing.T) *envCommandFixture {
	t.Helper()
	// SDK directory links use Windows-specific shims; this fixture tests the
	// Unix PATH exported for a zsh hook, without requiring zsh to be installed.
	if runtime.GOOS == "windows" {
		t.Skip("Unix shell hook integration test")
	}
	root := t.TempDir()
	f := &envCommandFixture{
		home: filepath.Join(root, ".vfox"), work: filepath.Join(root, "project"),
		state:  filepath.Join(root, ".vfox", "tmp", "test-session", "env-state.json"),
		system: filepath.Join(root, "system-bin"), names: []string{"alpha", "beta", "delta", "gamma"},
	}
	for _, dir := range []string{f.home, f.work, f.system} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	config := "[tools]\n"
	for _, name := range f.names {
		pluginDir := filepath.Join(f.home, "plugin", name)
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatal(err)
		}
		extra := ""
		if name == "alpha" {
			extra = `, {key="ALPHA_VERSION", value=ctx.main.version}`
		}
		script := fmt.Sprintf(`PLUGIN = {name=%q, version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {} end
function PLUGIN:EnvKeys(ctx)
 local log = assert(io.open(ctx.path .. "/calls", "a"))
 log:write("called\n")
 log:close()
 return {{key="PATH", value=ctx.path .. "/bin"}%s}
end
`, name, extra)
		if err := os.WriteFile(filepath.Join(pluginDir, "main.lua"), []byte(script), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(f.home, "cache", name, "v-1.0.0", name+"-1.0.0", "bin"), 0755); err != nil {
			t.Fatal(err)
		}
		config += fmt.Sprintf("%s = \"1.0.0\"\n", name)
	}
	if err := os.WriteFile(filepath.Join(f.home, "vfox.toml"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", root)
	t.Setenv("VFOX_HOME", f.home)
	t.Setenv("__VFOX_CURTMPPATH", filepath.Dir(f.state))
	t.Setenv("VFOX_TEST_ENV_COMMAND", "1")
	t.Setenv(env.PathVarName, f.system)
	return f
}

func (f *envCommandFixture) runOutput(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestEnvCommandProcess$")
	cmd.Dir = f.work
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("env command: %v\n%s", err, output)
	}
	return string(output)
}

func (f *envCommandFixture) run(t *testing.T) string {
	t.Helper()
	output := []byte(f.runOutput(t))
	match := regexp.MustCompile(`export PATH="([^"]*)"`).FindSubmatch(output)
	if match == nil {
		t.Fatalf("missing PATH export: %s", output)
	}
	return string(match[1])
}

func (f *envCommandFixture) calls(t *testing.T) []string {
	t.Helper()
	var result []string
	for _, name := range f.names {
		data, err := os.ReadFile(filepath.Join(f.home, "cache", name, "v-1.0.0", name+"-1.0.0", "calls"))
		if err != nil {
			t.Fatal(err)
		}
		result = append(result, string(data))
	}
	return result
}

func TestEnvPathStableAcrossRebuilds(t *testing.T) {
	f := newEnvCommandFixture(t)
	var paths []string
	for _, name := range f.names {
		paths = append(paths, filepath.Join(f.home, "sdks", name, "bin"))
	}
	paths = append(paths, f.system)
	want := strings.Join(paths, string(os.PathListSeparator))
	// Delete only cache state, keeping identical input PATH and SDK configuration.
	// Check the explicit expected order, not merely equality with the first run.
	for i := 0; i < 12; i++ {
		if err := os.Remove(f.state); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if got := f.run(t); got != want {
			t.Fatalf("rebuild %d PATH = %q, want %q", i, got, want)
		}
	}
}

func TestEnvCacheHitsAfterApplyingPath(t *testing.T) {
	f := newEnvCommandFixture(t)
	first := f.run(t)
	t.Setenv(env.PathVarName, first) // Model eval of the shell's PATH export.
	// Cache stores the input PATH, so one further rebuild records convergence.
	if got := f.run(t); got != first {
		t.Fatalf("PATH did not converge: %q -> %q", first, got)
	}
	calls := f.calls(t)
	for _, log := range calls {
		if log != "called\ncalled\n" {
			t.Fatalf("expected two initial SDK evaluations, got %q", log)
		}
	}
	// A fixed old mtime detects rewrites even on filesystems with coarse timestamps.
	stamp := time.Unix(1234567890, 0)
	if err := os.Chtimes(f.state, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(f.state)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if got := f.run(t); got != first {
			t.Fatalf("cache hit changed PATH: %q", got)
		}
		after, err := os.Stat(f.state)
		if err != nil {
			t.Fatal(err)
		}
		if !after.ModTime().Equal(before.ModTime()) {
			t.Fatal("cache hit rewrote env-state.json")
		}
	}
	if got := f.calls(t); !reflect.DeepEqual(got, calls) {
		t.Fatalf("cache hits evaluated SDK hooks again: %q", got)
	}
}

func TestEnvCacheRespectsTargetShell(t *testing.T) {
	f := newEnvCommandFixture(t)
	for i, target := range []struct{ name, prefix string }{
		{"pwsh", "$env:ALPHA_VERSION="}, {"fish", "set -x -g 'ALPHA_VERSION' "}, {"bash", "export ALPHA_VERSION="}, {"pwsh", "$env:ALPHA_VERSION="},
	} {
		t.Setenv("VFOX_TEST_TARGET_SHELL", target.name)
		output := f.runOutput(t)
		// Exports are not ordered, so look for the line anywhere in the output.
		if !strings.Contains(output, target.prefix) {
			t.Fatalf("shell %s: unexpected output %q", target.name, output)
		}
		calls := f.calls(t)
		for _, log := range calls {
			if log != strings.Repeat("called\n", i+1) {
				t.Fatalf("shell switch did not recalculate hooks: %q", log)
			}
		}
		// Shell names are case-insensitive, including when looking up cached output.
		t.Setenv("VFOX_TEST_TARGET_SHELL", strings.ToUpper(target.name))
		if got := f.runOutput(t); got != output {
			t.Fatalf("cached output = %q, want %q", got, output)
		}
		if got := f.calls(t); !reflect.DeepEqual(got, calls) {
			t.Fatal("same shell evaluated hooks again")
		}
	}
}

func TestEnvCacheWithoutShellIsInvalidated(t *testing.T) {
	f := newEnvCommandFixture(t)
	f.run(t)
	data, err := os.ReadFile(f.state)
	if err != nil {
		t.Fatal(err)
	}
	var state map[string]any
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	delete(state, "cached_shell")
	state["cached_output"] = "old shell script"
	data, err = json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.state, data, 0600); err != nil {
		t.Fatal(err)
	}
	f.run(t)
	for _, log := range f.calls(t) {
		if log != "called\ncalled\n" {
			t.Fatalf("old cache did not recalculate hooks: %q", log)
		}
	}
}

// A legacy version file (.nvmrc and the like) is not a tracked config file, so
// the cached output must not survive a cd into or out of a directory with one,
// an edit of it in place, or a cd to another directory with the same one.
func TestEnvCacheFollowsLegacyVersionFiles(t *testing.T) {
	f := newEnvCommandFixture(t)
	pluginDir := filepath.Join(f.home, "plugin", "legacy")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	script := `PLUGIN = {name="legacy", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0", legacyFilenames={".legacy-version"}}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {} end
function PLUGIN:EnvKeys(ctx) return {{key="LEGACY_VERSION", value=ctx.main.version}, {key="LEGACY_PATH", value=ctx.path}} end
function PLUGIN:ParseLegacyFile(ctx)
  local f = io.open(ctx.filepath, "r")
  local content = f:read("*a")
  f:close()
  return {version=content:gsub("%s+", "")}
end
`
	if err := os.WriteFile(filepath.Join(pluginDir, "main.lua"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"1.0.0", "2.0.0", "3.0.0"} {
		if err := os.MkdirAll(filepath.Join(f.home, "cache", "legacy", "v-"+v, "legacy-"+v), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(f.home, ".legacy_filenames"), []byte("legacy true\n"), 0644); err != nil {
		t.Fatal(err)
	}
	global, err := os.ReadFile(filepath.Join(f.home, "vfox.toml"))
	if err != nil {
		t.Fatal(err)
	}
	global = append(global, []byte("legacy = \"1.0.0\"\n")...)
	if err := os.WriteFile(filepath.Join(f.home, "vfox.toml"), global, 0600); err != nil {
		t.Fatal(err)
	}

	plain := f.work
	withLegacy := filepath.Join(filepath.Dir(f.work), "with-legacy")
	if err := os.MkdirAll(withLegacy, 0755); err != nil {
		t.Fatal(err)
	}
	legacyFile := filepath.Join(withLegacy, ".legacy-version")
	if err := os.WriteFile(legacyFile, []byte("2.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	version := func(dir string) string {
		t.Helper()
		f.work = dir
		output := f.runOutput(t)
		match := regexp.MustCompile(`LEGACY_VERSION="?([0-9.]+)`).FindStringSubmatch(output)
		if match == nil {
			t.Fatalf("missing LEGACY_VERSION export in %s: %s", dir, output)
		}
		return match[1]
	}

	for i, step := range []struct {
		dir, want string
	}{
		{plain, "1.0.0"},      // global version
		{plain, "1.0.0"},      // cache hit
		{withLegacy, "2.0.0"}, // cd into a directory with a legacy file
		{withLegacy, "2.0.0"}, // cache hit
		{plain, "1.0.0"},      // cd out of it
		{withLegacy, "2.0.0"}, // and back in
	} {
		if got := version(step.dir); got != step.want {
			t.Fatalf("step %d in %s: LEGACY_VERSION = %s, want %s", i, step.dir, got, step.want)
		}
	}

	// Edit the legacy file in place; the directory and config files stay the same.
	if err := os.WriteFile(legacyFile, []byte("3.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := version(withLegacy); got != "3.0.0" {
		t.Fatalf("after editing the legacy file: LEGACY_VERSION = %s, want 3.0.0", got)
	}

	// A second directory whose legacy file resolves to the same tools still
	// exports them through its own .vfox/sdks.
	withLegacyToo := filepath.Join(filepath.Dir(f.work), "with-legacy-too")
	if err := os.MkdirAll(withLegacyToo, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(withLegacyToo, ".legacy-version"), []byte("3.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	pathRe := regexp.MustCompile(`LEGACY_PATH=\$?['"]?([^'"\s;]+)`)
	for _, dir := range []string{withLegacy, withLegacyToo, withLegacy, withLegacyToo} {
		want := filepath.Join(dir, ".vfox", "sdks", "legacy")
		for i := 0; i < 3; i++ {
			f.work = dir
			output := f.runOutput(t)
			match := pathRe.FindStringSubmatch(output)
			if match == nil {
				t.Fatalf("missing LEGACY_PATH export in %s: %s", dir, output)
			}
			if match[1] != want {
				t.Fatalf("LEGACY_PATH in %s = %s, want %s", dir, match[1], want)
			}
		}
	}
}

func TestProjectToolsKey(t *testing.T) {
	if key := projectToolsKey(env.NewVfoxTomlChain(), "/a/.vfox/sdks"); key != "" {
		t.Errorf("chain without a project scope: key = %q, want empty", key)
	}

	empty := env.NewVfoxTomlChain()
	empty.Add(pathmeta.NewVfoxToml(), env.Project)
	if key := projectToolsKey(empty, "/a/.vfox/sdks"); key != "" {
		t.Errorf("project scope without tools: key = %q, want empty", key)
	}

	keyFor := func(version string, dir ...string) string {
		chain := env.NewVfoxTomlChain()
		project := pathmeta.NewVfoxToml()
		project.Tools.Set("nodejs", version)
		chain.Add(project, env.Project)
		global := pathmeta.NewVfoxToml()
		global.Tools.Set("python", "3.12")
		chain.Add(global, env.Global)
		sdkDir := "/a/.vfox/sdks"
		if len(dir) > 0 {
			sdkDir = dir[0]
		}
		return projectToolsKey(chain, sdkDir)
	}
	k20 := keyFor("20")
	if k20 == "" || !strings.Contains(k20, "nodejs") || strings.Contains(k20, "python") {
		t.Errorf("key should describe only the project tools, got %q", k20)
	}
	if k20 != keyFor("20") {
		t.Error("the same project tools must give the same key")
	}
	if k20 == keyFor("22") {
		t.Error("a different project version must give a different key")
	}
	if k20 == keyFor("20", "/b/.vfox/sdks") {
		t.Error("the same project tools in another directory must give a different key")
	}
}

// converge runs env until the PATH it exports is the one it was given, as at
// the prompts of a shell that evals the output, and returns the last output.
func (f *envCommandFixture) converge(t *testing.T) string {
	t.Helper()
	var output string
	for i := 0; i < 3; i++ {
		output = f.runOutput(t)
		if match := regexp.MustCompile(`export PATH="([^"]*)"`).FindStringSubmatch(output); match != nil {
			t.Setenv(env.PathVarName, match[1])
		}
	}
	return output
}

func (f *envCommandFixture) writeProjectConfig(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func alphaVersion(t *testing.T, output string) string {
	t.Helper()
	match := regexp.MustCompile(`ALPHA_VERSION="?([0-9.]+)`).FindStringSubmatch(output)
	if match == nil {
		t.Fatalf("missing ALPHA_VERSION export: %s", output)
	}
	return match[1]
}

// touchAhead moves the modification time of dir a minute ahead, so that the
// change is seen also on file systems with coarse clocks.
func touchAhead(t *testing.T, dir string) {
	t.Helper()
	ahead := time.Now().Add(time.Minute)
	if err := os.Chtimes(dir, ahead, ahead); err != nil {
		t.Fatal(err)
	}
}

// Installing or removing a version changes no tracked file and not the PATH,
// so the cached output must also be keyed by what is installed.
func TestEnvCacheFollowsInstalls(t *testing.T) {
	f := newEnvCommandFixture(t)
	f.writeProjectConfig(t, "[tools]\nalpha = \"2.0.0\"\n")
	if got := alphaVersion(t, f.converge(t)); got != "1.0.0" {
		t.Fatalf("before installing 2.0.0: ALPHA_VERSION = %s, want 1.0.0", got)
	}
	installs := filepath.Join(f.home, "cache", "alpha")
	v2 := filepath.Join(installs, "v-2.0.0")
	if err := os.MkdirAll(filepath.Join(v2, "alpha-2.0.0", "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	touchAhead(t, installs)
	if got := alphaVersion(t, f.converge(t)); got != "2.0.0" {
		t.Fatalf("after installing 2.0.0: ALPHA_VERSION = %s, want 2.0.0", got)
	}
	if err := os.RemoveAll(v2); err != nil {
		t.Fatal(err)
	}
	ahead := time.Now().Add(2 * time.Minute)
	if err := os.Chtimes(installs, ahead, ahead); err != nil {
		t.Fatal(err)
	}
	if got := alphaVersion(t, f.runOutput(t)); got != "1.0.0" {
		t.Fatalf("after removing 2.0.0: ALPHA_VERSION = %s, want 1.0.0", got)
	}
}

// sdk.NewSdk falls back to <installs>/cache/<name> for installs made by
// v1.0.2; an install there must be seen at the next prompt as well.
func TestEnvCacheFollowsInstallsInTheFallbackLayout(t *testing.T) {
	f := newEnvCommandFixture(t)
	cache := filepath.Join(f.home, "cache")
	installs := filepath.Join(cache, "cache", "alpha")
	if err := os.MkdirAll(filepath.Dir(installs), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(cache, "alpha"), installs); err != nil {
		t.Fatal(err)
	}
	f.writeProjectConfig(t, "[tools]\nalpha = \"2.0.0\"\n")
	if got := alphaVersion(t, f.converge(t)); got != "1.0.0" {
		t.Fatalf("before installing 2.0.0 into the fallback layout: ALPHA_VERSION = %s, want 1.0.0", got)
	}
	if err := os.MkdirAll(filepath.Join(installs, "v-2.0.0", "alpha-2.0.0", "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	touchAhead(t, installs)
	if got := alphaVersion(t, f.converge(t)); got != "2.0.0" {
		t.Fatalf("after installing 2.0.0 into the fallback layout: ALPHA_VERSION = %s, want 2.0.0", got)
	}
}

// Adding the plugin of a configured tool must be seen at the next prompt too.
func TestEnvCacheFollowsAddedPlugins(t *testing.T) {
	f := newEnvCommandFixture(t)
	if err := os.MkdirAll(filepath.Join(f.home, "cache", "epsilon", "v-1.0.0", "epsilon-1.0.0", "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	f.writeProjectConfig(t, "[tools]\nepsilon = \"1.0.0\"\n")
	want := filepath.Join(f.work, ".vfox", "sdks", "epsilon", "bin")
	if path := f.run(t); strings.Contains(path, want) {
		t.Fatalf("epsilon is on the PATH before its plugin is added: %s", path)
	}
	f.converge(t)
	pluginDir := filepath.Join(f.home, "plugin", "epsilon")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatal(err)
	}
	script := `PLUGIN = {name="epsilon", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`
	if err := os.WriteFile(filepath.Join(pluginDir, "main.lua"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	if path := f.run(t); !strings.Contains(path, want) {
		t.Fatalf("after adding the plugin the PATH still lacks %s: %s", want, path)
	}
}

// A cache hit does not make the links again, so their removal must count as
// a change: rm -rf .vfox in the project, or of the global sdks directory.
func TestEnvCacheRepairsRemovedLinks(t *testing.T) {
	f := newEnvCommandFixture(t)
	f.writeProjectConfig(t, "[tools]\nalpha = \"1.0.0\"\n")
	f.converge(t)
	for _, c := range []struct{ remove, link string }{
		{filepath.Join(f.work, ".vfox"), filepath.Join(f.work, ".vfox", "sdks", "alpha")},
		{filepath.Join(f.home, "sdks"), filepath.Join(f.home, "sdks", "beta")},
	} {
		if _, err := os.Lstat(c.link); err != nil {
			t.Fatalf("the link %s was not made: %v", c.link, err)
		}
		if err := os.RemoveAll(c.remove); err != nil {
			t.Fatal(err)
		}
		f.runOutput(t)
		if _, err := os.Lstat(c.link); err != nil {
			t.Fatalf("the link %s was not made again: %v", c.link, err)
		}
	}
}

// In the home directory the project link directory would be the global one;
// a project config there must leave the global links alone.
func TestEnvProjectInHomeKeepsGlobalLinks(t *testing.T) {
	testEnvProjectInHomeKeepsGlobalLinks(t, false)
}

// The same with the working directory a symlink to the home directory.
func TestEnvProjectInHomeThroughSymlinkKeepsGlobalLinks(t *testing.T) {
	testEnvProjectInHomeKeepsGlobalLinks(t, true)
}

func testEnvProjectInHomeKeepsGlobalLinks(t *testing.T, throughSymlink bool) {
	f := newEnvCommandFixture(t)
	if err := os.MkdirAll(filepath.Join(f.home, "cache", "alpha", "v-2.0.0", "alpha-2.0.0", "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	f.runOutput(t)
	link := filepath.Join(f.home, "sdks", "alpha")
	before, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	f.work = filepath.Dir(f.home) // HOME of the fixture
	if throughSymlink {
		link := filepath.Join(t.TempDir(), "home-link")
		if err := os.Symlink(f.work, link); err != nil {
			t.Skipf("symlinks are not available: %v", err)
		}
		f.work = link
	}
	f.writeProjectConfig(t, "[tools]\nalpha = \"2.0.0\"\n")
	if got := alphaVersion(t, f.runOutput(t)); got != "2.0.0" {
		t.Fatalf("in the home directory: ALPHA_VERSION = %s, want 2.0.0", got)
	}
	after, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("the global link of alpha was repointed from %s to %s", before, after)
	}
}
