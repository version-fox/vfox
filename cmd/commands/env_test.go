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
)

// Run the real command in a fresh process, as a shell hook would. This also
// isolates CLI state, environment variables and SDK initialization between runs.
func TestEnvCommandProcess(t *testing.T) {
	if os.Getenv("VFOX_TEST_ENV_COMMAND") != "1" {
		return
	}
	app := &cli.Command{Name: "vfox", Commands: []*cli.Command{Env}}
	if err := app.Run(context.Background(), []string{"vfox", "env", "-s", "zsh"}); err != nil {
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
		script := fmt.Sprintf(`PLUGIN = {name=%q, version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {} end
function PLUGIN:EnvKeys(ctx)
 local log = assert(io.open(ctx.path .. "/calls", "a"))
 log:write("called\n")
 log:close()
 return {{key="PATH", value=ctx.path .. "/bin"}}
end
`, name)
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

func (f *envCommandFixture) run(t *testing.T) string {
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
