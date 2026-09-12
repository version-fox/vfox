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

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPluginCLIProcess(t *testing.T) {
	if os.Getenv("VFOX_PLUGIN_CLI_PROCESS") != "1" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		os.Exit(2)
	}
	os.Args = append([]string{"vfox"}, os.Args[separator+1:]...)
	Execute(os.Args)
	os.Exit(0)
}

func writeCLIPluginFile(t *testing.T, root, name, source string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const cliTestPlugin = `PLUGIN={name="cli-test",version="0.0.1"}
function PLUGIN:Available(ctx) return {{version="1.2.3"}} end
function PLUGIN:PreInstall(ctx)
  print("hook log")
  io.write("io log")
  io.stdout:write("stdout log")
  return {version=ctx.version, url=RUNTIME.osType.."-"..RUNTIME.archType..".zip", unknown="drop me"}
end
function PLUGIN:EnvKeys(ctx) return {} end
function PLUGIN:PreUse(ctx) return nil end
function PLUGIN:PostInstall(ctx) assert(os.execute("echo child log") == 0) end
function PLUGIN:ParseLegacyFile(ctx) return {version=ctx.getInstalledVersions()[1]} end
`

func runPluginCLI(t *testing.T, root string, args ...string) (string, string, int) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, append([]string{"-test.run=^TestPluginCLIProcess$", "--"}, args...)...)
	command.Dir = root
	command.Env = append(os.Environ(), "VFOX_PLUGIN_CLI_PROCESS=1",
		"HOME="+filepath.Join(root, "user"), "USERPROFILE="+filepath.Join(root, "user"),
		"VFOX_HOME="+filepath.Join(root, "shared"))
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err = command.Run()
	code := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	if ctx.Err() != nil {
		t.Fatal("plugin CLI hung")
	}
	for _, path := range []string{"user", "shared", ".vfox", ".vfox.toml", "vfox.toml"} {
		if _, err := os.Stat(filepath.Join(root, path)); !os.IsNotExist(err) {
			t.Errorf("development command touched %s: %v", path, err)
		}
	}
	return stdout.String(), stderr.String(), code
}

func TestPluginRunCLI(t *testing.T) {
	root := t.TempDir()
	writeCLIPluginFile(t, root, "plugin/main.lua", cliTestPlugin)
	writeCLIPluginFile(t, root, ".tool-versions", "sample 1.2.3\n")
	writeCLIPluginFile(t, root, "input.json", `{"version":"2.0.0"}`)

	stdout, stderr, code := runPluginCLI(t, root, "--debug", "plugin", "run", "plugin", "PreInstall", "--input", `{"version":"1.2.3"}`, "--os", "windows", "--arch", "arm64", "--json", "--offline")
	if code != 0 {
		t.Fatalf("run exit = %d: %s", code, stderr)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("stdout is not a single JSON result: %q, %v", stdout, err)
	}
	if result["version"] != "1.2.3" || result["url"] != "windows-arm64.zip" || result["unknown"] != nil {
		t.Fatalf("result = %v", result)
	}
	for _, message := range []string{"hook log", "io log", "stdout log"} {
		if !strings.Contains(stderr, message) {
			t.Errorf("stderr missing %q: %q", message, stderr)
		}
	}

	stdout, stderr, code = runPluginCLI(t, root, "plugin", "run", "plugin", "PreInstall", "--input-file", "input.json", "--json")
	if code != 0 || !strings.Contains(stdout, `"version":"2.0.0"`) {
		t.Fatalf("input file: %q, %q, %d", stdout, stderr, code)
	}
	for _, hook := range []string{"PostInstall", "PreUse", "EnvKeys"} {
		stdout, stderr, code = runPluginCLI(t, root, "plugin", "run", "plugin", hook, "--json")
		if code != 0 || strings.TrimSpace(stdout) != "null" {
			t.Fatalf("%s: %q, %q, %d", hook, stdout, stderr, code)
		}
		if hook == "PostInstall" && !strings.Contains(stderr, "child log") {
			t.Fatalf("child stdout was not redirected: %q", stderr)
		}
	}
	stdout, stderr, code = runPluginCLI(t, root, "plugin", "run", "plugin", "ParseLegacyFile", "--json", "--input", `{"installedVersions":["3.0.0"]}`)
	if code != 0 || strings.TrimSpace(stdout) != `{"version":"3.0.0"}` {
		t.Fatalf("legacy: %q, %q, %d", stdout, stderr, code)
	}

	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"plugin", "run", "plugin", "PreInstall", "--input", "[]", "--json"}, "JSON object"},
		{[]string{"plugin", "run", "plugin", "PreInstall", "--input", "null", "--json"}, "JSON object"},
		{[]string{"plugin", "run", "plugin", "PreInstall", "--input", "{", "--json"}, "JSON object"},
		{[]string{"plugin", "run", "plugin", "PreInstall", "--input", "{}", "--input-file", "input.json"}, "either --input or --input-file"},
		{[]string{"plugin", "run", "plugin", "preUse", "--json"}, "unknown hook"},
		{[]string{"plugin", "run", "plugin", "Available", "--timeout", "0s", "--json"}, "timeout must be positive"},
		{[]string{"plugin", "run", "plugin", "PreUninstall", "--json"}, "function not found"},
	} {
		stdout, stderr, code := runPluginCLI(t, root, tc.args...)
		if code != 1 || stdout != "" || !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: %q, %q, %d", tc.args, stdout, stderr, code)
		}
	}
}

func TestPluginTestCLI(t *testing.T) {
	root := t.TempDir()
	writeCLIPluginFile(t, root, "plugin/main.lua", cliTestPlugin)
	writeCLIPluginFile(t, root, ".tool-versions", "sample 1.2.3\n")
	writeCLIPluginFile(t, root, "plugin/tests/a_test.lua", `require("vfox.test").load()`+"\n"+`assert(false,"first failure")`+"\n"+`error("must not reach")`)
	writeCLIPluginFile(t, root, "plugin/tests/nested/b_test.lua", `assert(PLUGIN == nil); local p=require("vfox.test").load(); assert(p:Available({})[1].version == "1.2.3")`)
	writeCLIPluginFile(t, root, "plugin/tests/ignored.lua", `error("not a test file")`)
	stdout, stderr, code := runPluginCLI(t, root, "plugin", "test", "plugin")
	if code != 1 || stdout != "FAIL tests/a_test.lua\nPASS tests/nested/b_test.lua\n" || !strings.Contains(stderr, "a_test.lua:2: first failure") || strings.Contains(stderr, "must not reach") {
		t.Fatalf("discovery: %q, %q, %d", stdout, stderr, code)
	}
	stdout, stderr, code = runPluginCLI(t, root, "plugin", "test", "plugin", "--file", "tests/nested/b_test.lua")
	if code != 0 || stdout != "PASS tests/nested/b_test.lua\n" || stderr != "" {
		t.Fatalf("filter: %q, %q, %d", stdout, stderr, code)
	}
	writeCLIPluginFile(t, root, "plugin/tests/a_test.lua", `require("vfox.test").load(); while true do end`)
	stdout, stderr, code = runPluginCLI(t, root, "plugin", "test", "plugin", "--timeout", "50ms")
	if code != 1 || !strings.Contains(stdout, "PASS tests/nested/b_test.lua") || !strings.Contains(stderr, "deadline exceeded") {
		t.Fatalf("timeout continuation: %q, %q, %d", stdout, stderr, code)
	}

	writeCLIPluginFile(t, root, "empty/main.lua", cliTestPlugin)
	stdout, stderr, code = runPluginCLI(t, root, "plugin", "test", "empty")
	if code != 1 || stdout != "" || !strings.Contains(stderr, "no test files") {
		t.Fatalf("empty: %q, %q, %d", stdout, stderr, code)
	}
	stdout, stderr, code = runPluginCLI(t, root, "plugin", "test", "plugin", "--file", "missing.lua")
	if code != 1 || stdout != "" {
		t.Fatalf("missing filter: %q, %q, %d", stdout, stderr, code)
	}
}

func TestPluginExampleCLI(t *testing.T) {
	example, err := filepath.Abs(filepath.Join("..", "examples", "plugins", "sample"))
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code := runPluginCLI(t, t.TempDir(), "plugin", "test", example)
	if code != 0 || stderr != "" || strings.Count(stdout, "PASS ") != 2 {
		t.Fatalf("example: %q, %q, %d", stdout, stderr, code)
	}
}

func TestPluginNetworkCLI(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		fmt.Fprint(w, "1.2.3")
	}))
	defer server.Close()
	root := t.TempDir()
	writeCLIPluginFile(t, root, "plugin/main.lua", cliTestPlugin+fmt.Sprintf(`
local http=require("http")
function PLUGIN:Available(ctx)
  local response, err=http.get({url=%q})
  assert(response, err)
  return {{version=response.body}}
end`, server.URL))
	writeCLIPluginFile(t, root, "plugin/tests/network_test.lua", `local p=require("vfox.test").load(); assert(p:Available({})[1].version=="1.2.3")`)
	_, stderr, code := runPluginCLI(t, root, "plugin", "test", "plugin")
	if code != 1 || !strings.Contains(stderr, "offline") || requests.Load() != 0 {
		t.Fatalf("offline test: %d, %q, %d requests", code, stderr, requests.Load())
	}
	_, stderr, code = runPluginCLI(t, root, "plugin", "test", "plugin", "--online")
	if code != 0 || requests.Load() != 1 {
		t.Fatalf("online test: %d, %q, %d requests", code, stderr, requests.Load())
	}
	stdout, stderr, code := runPluginCLI(t, root, "plugin", "run", "plugin", "Available", "--json")
	if code != 0 || !strings.Contains(stdout, "1.2.3") || requests.Load() != 2 {
		t.Fatalf("online run: %d, %q, %q", code, stdout, stderr)
	}
	stdout, stderr, code = runPluginCLI(t, root, "plugin", "run", "plugin", "Available", "--json", "--offline")
	if code != 1 || stdout != "" || !strings.Contains(stderr, "offline") || requests.Load() != 2 {
		t.Fatalf("offline run: %d, %q, %q", code, stdout, stderr)
	}
	writeCLIPluginFile(t, root, "plugin/tests/network_test.lua", `
local p=require("vfox.test").load({http=function() return nil,"fixture failure" end})
local ok,err=pcall(function() p:Available({}) end)
assert(not ok and tostring(err):find("fixture failure",1,true))
`)
	_, stderr, code = runPluginCLI(t, root, "plugin", "test", "plugin", "--online")
	if code != 0 || requests.Load() != 2 {
		t.Fatalf("handler must not fall through to real HTTP: %d, %q, %d requests", code, stderr, requests.Load())
	}
}
