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

package plugin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
)

func developmentOptions() DevelopmentOptions {
	return DevelopmentOptions{RuntimeVersion: "1.0.12", Output: io.Discard}
}

func TestDevelopmentAllHooks(t *testing.T) {
	dir := t.TempDir()
	writePluginFile(t, dir, "main.lua", `
local http = require("http")
local json = require("json")
local html = require("html")
local initial_env = os.getenv("MIRROR")
local hidden = os.getenv("VFOX_PLUGIN_TEST_SECRET")
local boot = assert(http.get({url="https://example.com/boot"}))
assert(boot.body == "boot")
assert(RUNTIME == nil)
RUNTIME = {osType="overwritten"}
OS_TYPE, ARCH_TYPE = "overwritten", "overwritten"
PLUGIN = {name="example", version="0.1.0"}
function PLUGIN:Available(ctx)
  assert(initial_env == "fixture" and hidden == nil)
  assert(RUNTIME.osType == "windows" and RUNTIME.archType == "amd64")
  assert(OS_TYPE == "windows" and ARCH_TYPE == "amd64")
  assert(ctx.args[1] == "stable")
  local resp = assert(http.get({url="https://example.com/releases", headers={Accept="application/json"}}))
  local document = html.parse(json.decode(resp.body).html)
  return {{version=document:find("b"):text(), ignored="not in the typed result"}}
end
function PLUGIN:PreInstall(ctx)
  local resp = assert(http.head({url="https://example.com/archive", headers={["User-Agent"]="custom"}}))
  assert(resp.headers["X-Fixture"] == "yes")
  return {version=ctx.version, url="sdk.zip", sha256="main-hash", headers={Token="value"},
    addition={{name="extra", version="2.0", url="extra.zip", sha512="extra-hash"}}}
end
function PLUGIN:EnvKeys(ctx)
  assert(ctx.main.version == "1.2.3" and ctx.sdkInfo.extra.path == "extra-dir")
  return {{key="PATH", value=ctx.path.."/bin"}}
end
function PLUGIN:PreUse(ctx)
  if ctx.version == "none" then return nil end
  assert(ctx.cwd == "project-dir" and ctx.scope == "project" and ctx.previousVersion == "1.0")
  return {version=ctx.installedSdks["1.2.3"].version}
end
function PLUGIN:ParseLegacyFile(ctx)
  assert(ctx.filename == ".example-version" and ctx.filepath == "fixture-path" and ctx.strategy == "latest_installed")
  assert(type(ctx.getInstalledVersions) == "function")
  return {version=ctx.getInstalledVersions()[1]}
end
function PLUGIN:PostInstall(ctx)
  assert(ctx.sdkInfo.example.version == "1.2.3")
  assert(http.download_file({url="https://example.com/archive"}, ctx.rootPath.."/payload") == nil)
end
function PLUGIN:PreUninstall(ctx)
  assert(ctx.main.version == "1.2.3" and ctx.sdkInfo.extra.path == "extra-dir")
end
`)
	t.Setenv("VFOX_PLUGIN_TEST_SECRET", "host-secret")
	script := `
local test = require("vfox.test")
local requests = 0
local p = test.load({os="windows", arch="amd64", env={MIRROR="fixture", VFOX_PLUGIN_TEST_SECRET=false}, http=function(req)
  requests = requests + 1
  if req.url == "https://example.com/boot" then return {status_code=200, body="boot"} end
  if req.url == "https://example.com/releases" then
    assert(req.method == "GET" and req.headers.Accept == "application/json")
    assert(req.headers["User-Agent"]:find("vfox/1.0.12", 1, true))
    return {status_code=200, body='{"html":"<b>1.2.3</b>"}'}
  end
  assert(req.url == "https://example.com/archive")
  if req.method == "HEAD" then
    assert(req.headers["User-Agent"] == "custom")
    return {status_code=200, headers={["X-Fixture"]="yes"}}
  end
  assert(req.method == "GET")
  return {status_code=200, body="payload"}
end})
local versions = p:Available({args={"stable"}})
assert(versions[1].version == "1.2.3" and versions[1].ignored == nil)
local archive = p:PreInstall({version="1.2.3"})
assert(archive.url == "sdk.zip" and archive.sha256 == "main-hash" and archive.headers.Token == "value")
assert(archive.addition[1].name == "extra" and archive.addition[1].sha512 == "extra-hash")
local installed = {version="1.2.3", name="example", path="sdk-dir"}
local sdkInfo = {example=installed, extra={path="extra-dir"}}
assert(p:EnvKeys({path="sdk-dir", main=installed, sdkInfo=sdkInfo})[1].value == "sdk-dir/bin")
assert(p:PreUse({cwd="project-dir", scope="project", version="1.2", previousVersion="1.0", installedSdks={["1.2.3"]=installed}}).version == "1.2.3")
assert(p:PreUse({version="none"}) == nil)
assert(p:ParseLegacyFile({filename=".example-version", filepath="fixture-path", strategy="latest_installed", installedVersions={"1.2.3"}}).version == "1.2.3")
assert(p:PostInstall({rootPath=RUNTIME.pluginDirPath, sdkInfo=sdkInfo}) == nil)
assert(p:PreUninstall({main=installed, sdkInfo=sdkInfo}) == nil)
assert(requests == 4)
`
	file := writePluginFile(t, dir, "tests/all_test.lua", script)
	if err := TestDevelopmentFile(context.Background(), dir, file, developmentOptions()); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(filepath.Join(dir, "payload"))
	if err != nil || string(payload) != "payload" {
		t.Fatalf("download = %q, %v", payload, err)
	}
	if os.Getenv("VFOX_PLUGIN_TEST_SECRET") != "host-secret" {
		t.Fatal("host environment changed")
	}
}

func TestDevelopmentLayouts(t *testing.T) {
	for _, layout := range []string{"main", "metadata"} {
		t.Run(layout, func(t *testing.T) {
			dir := t.TempDir()
			available := `local util = require("util")
function PLUGIN:Available(ctx) return {{version=util.value}} end`
			if layout == "main" {
				writePluginFile(t, dir, "main.lua", minimalPlugin+available)
				writePluginFile(t, dir, "metadata.lua", `error("main.lua must take precedence")`)
				writePluginFile(t, dir, "util.lua", `return {value="main"}`)
			} else {
				writePluginFile(t, dir, "metadata.lua", `PLUGIN={name="example"}`)
				writePluginFile(t, dir, "hooks/available.lua", available)
				writePluginFile(t, dir, "hooks/pre_install.lua", `function PLUGIN:PreInstall(ctx) return {version=ctx.version} end`)
				writePluginFile(t, dir, "hooks/env_keys.lua", `function PLUGIN:EnvKeys(ctx) return {} end`)
				writePluginFile(t, dir, "hooks/pre_use.lua", `function PLUGIN:PreUse(ctx) return {version="optional"} end`)
				writePluginFile(t, dir, "hooks/util.lua", `return {value="metadata"}`)
				writePluginFile(t, dir, "lib/util.lua", `error("hooks must take precedence")`)
			}
			// Both development and normal loading use the same module paths.
			p, err := CreatePlugin(dir, &env.RuntimeEnvContext{UserConfig: config.DefaultConfig})
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			versions, err := p.Available(&AvailableHookCtx{})
			if err != nil || versions[0].Version != layout {
				t.Fatalf("normal loading = %v, %v", versions, err)
			}
			result, err := RunDevelopmentHook(context.Background(), dir, "Available", nil, developmentOptions())
			if err != nil || result.([]*AvailableHookResultItem)[0].Version != layout {
				t.Fatalf("development loading = %v, %v", result, err)
			}
			if layout == "metadata" {
				result, err := RunDevelopmentHook(context.Background(), dir, "PreUse", nil, developmentOptions())
				if err != nil || result.(*PreUseHookResult).Version != "optional" {
					t.Fatalf("optional hook = %v, %v", result, err)
				}
			}
		})
	}
}

func TestDevelopmentErrors(t *testing.T) {
	for _, tc := range []struct{ name, source, script, want string }{
		{"assert location", "", `local p=require("vfox.test").load()` + "\n" + `assert(false, "first failure")`, "case_test.lua:2: first failure"},
		{"exit", "", `require("vfox.test").load(); os.exit(0)`, "os.exit(0)"},
		{"execute", "", `require("vfox.test").load(); os.execute("echo forbidden")`, "os.execute is disabled"},
		{"popen", "", `require("vfox.test").load(); io.popen("echo forbidden")`, "io.popen is disabled"},
		{"second instance", "", `local t=require("vfox.test"); t.load(); t.load()`, "once per test file"},
		{"missing load", "", `assert(true)`, "did not load"},
		{"missing optional", "", `require("vfox.test").load():PreUse({})`, "PreUse"},
		{"bad input", "", `require("vfox.test").load():PreInstall({version={}})`, "version"},
		{"bad output", `function PLUGIN:Available(ctx) return "wrong" end`, `require("vfox.test").load():Available({})`, "cannot unmarshal string"},
		{"bad nested output", `function PLUGIN:PreInstall(ctx) return {addition={{headers="wrong"}}} end`, `require("vfox.test").load():PreInstall({})`, "headers"},
		{"bad option", "", `require("vfox.test").load({network=true})`, "unknown test.load option"},
		{"bad env", "", `require("vfox.test").load({env={TOKEN=true}})`, "string or false"},
		{"offline", `local http=require("http"); function PLUGIN:Available(ctx) local r,e=http.get({url="https://example.com"}); assert(r,e) end`, `require("vfox.test").load():Available({})`, "HTTP is disabled offline"},
		{"handler assertion", `local http=require("http"); function PLUGIN:Available(ctx) http.get({url="https://example.com"}); return {} end`, `require("vfox.test").load({http=function() assert(false,"wrong request") end}):Available({})`, "wrong request"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writePluginFile(t, dir, "main.lua", minimalPlugin+tc.source)
			file := writePluginFile(t, dir, "tests/case_test.lua", tc.script)
			err := TestDevelopmentFile(context.Background(), dir, file, developmentOptions())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestDevelopmentHTTPFailures(t *testing.T) {
	dir := t.TempDir()
	writePluginFile(t, dir, "main.lua", minimalPlugin+`
local http = require("http")
function PLUGIN:Available(ctx)
  local response, err = http.get({url="https://example.com/releases"})
  if err then error(err) end
  assert(response.status_code == 200, "HTTP "..response.status_code)
  return {{version=response.body}}
end
`)
	file := writePluginFile(t, dir, "tests/http_test.lua", `
local mode = "status"
local p = require("vfox.test").load({http=function()
  if mode == "status" then return {status_code=503,body="unavailable"} end
  if mode == "transport" then return nil, "connection reset" end
  return {status_code=200,body="1.2.3"}
end})
local ok, err = pcall(function() p:Available({}) end)
assert(not ok and tostring(err):find("HTTP 503", 1, true), tostring(err))
mode = "transport"
ok, err = pcall(function() p:Available({}) end)
assert(not ok and tostring(err):find("connection reset", 1, true), tostring(err))
mode = "success"
assert(p:Available({})[1].version == "1.2.3")
`)
	if err := TestDevelopmentFile(context.Background(), dir, file, developmentOptions()); err != nil {
		t.Fatal(err)
	}
}

func TestDevelopmentVMIsolationAndStubs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VFOX_PLUGIN_TEST_SECRET", "host")
	writePluginFile(t, dir, "main.lua", minimalPlugin+`
local http = require("http")
local counter = 0
function PLUGIN:Available(ctx)
  counter = counter + 1
  return {{version=tostring(counter)}}
end
function PLUGIN:PostInstall(ctx) assert(os.execute("pretend-install") == 0) end
`)
	first := writePluginFile(t, dir, "tests/first_test.lua", `
local calls = 0
os.execute = function(command) assert(command == "pretend-install"); calls=calls+1; return 0 end
local p=require("vfox.test").load({env={VFOX_PLUGIN_TEST_SECRET="override"}, http=function() return {status_code=200} end})
assert(os.getenv("VFOX_PLUGIN_TEST_SECRET") == "override")
assert(p:Available({})[1].version == "1")
assert(p:Available({})[1].version == "2")
p:PostInstall({})
assert(calls == 1)
GLOBAL_MARKER = true
package.loaded["marker"] = true
`)
	second := writePluginFile(t, dir, "tests/second_test.lua", `
assert(GLOBAL_MARKER == nil and package.loaded["marker"] == nil and PLUGIN == nil and RUNTIME == nil)
assert(os.getenv("VFOX_PLUGIN_TEST_SECRET") == nil)
local p=require("vfox.test").load()
assert(p:Available({})[1].version == "1")
local r,e=require("http").get({url="https://example.com"})
assert(r == nil and e:find("offline"))
local ok,err=pcall(function() p:PostInstall({}) end)
assert(not ok and tostring(err):find("disabled"))
`)
	for _, file := range []string{first, second} {
		if err := TestDevelopmentFile(context.Background(), dir, file, developmentOptions()); err != nil {
			t.Fatal(err)
		}
	}
	if os.Getenv("VFOX_PLUGIN_TEST_SECRET") != "host" {
		t.Fatal("host environment changed")
	}
}

func TestDevelopmentCoroutine(t *testing.T) {
	dir := t.TempDir()
	writePluginFile(t, dir, "main.lua", minimalPlugin+`local http=require("http")
function PLUGIN:Available(ctx) return {{version=assert(http.get({url="https://example.com"})).body}} end`)
	file := writePluginFile(t, dir, "tests/coroutine_test.lua", `
local p=require("vfox.test").load({http=function() return {status_code=200,body="1.2.3"} end})
local thread=coroutine.create(function() assert(p:Available({})[1].version == "1.2.3") end)
local ok,err=coroutine.resume(thread)
assert(ok,err)
`)
	if err := TestDevelopmentFile(context.Background(), dir, file, developmentOptions()); err != nil {
		t.Fatal(err)
	}
}

func TestDevelopmentLuaTimeouts(t *testing.T) {
	for _, tc := range []struct{ name, source, script string }{
		{"test", "", `require("vfox.test").load(); while true do end`},
		{"load", `while true do end`, `require("vfox.test").load()`},
		{"hook", `function PLUGIN:Available(ctx) while true do end end`, `require("vfox.test").load():Available({})`},
		{"handler", `local http=require("http"); function PLUGIN:Available(ctx) http.get({url="https://example.com"}) end`, `require("vfox.test").load({http=function() while true do end end}):Available({})`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writePluginFile(t, dir, "main.lua", minimalPlugin+tc.source)
			file := writePluginFile(t, dir, "tests/timeout_test.lua", tc.script)
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			err := TestDevelopmentFile(ctx, dir, file, developmentOptions())
			if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
				t.Fatalf("timeout error = %v", err)
			}
		})
	}
}

func TestDevelopmentHTTPDeadline(t *testing.T) {
	for _, method := range []string{"get", "head", "download_file"} {
		for _, stallBody := range []bool{false, true} {
			if method == "head" && stallBody {
				continue
			}
			t.Run(fmt.Sprintf("%s/body=%t", method, stallBody), func(t *testing.T) {
				release := make(chan struct{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if stallBody {
						w.WriteHeader(200)
						w.(http.Flusher).Flush()
					}
					<-release
				}))
				defer server.Close()
				defer close(release)
				dir := t.TempDir()
				writePluginFile(t, dir, "main.lua", minimalPlugin+fmt.Sprintf(`
local http=require("http")
function PLUGIN:Available(ctx)
  local r,e=http.%s({url=%q}, RUNTIME.pluginDirPath.."/download")
  error(e or r or "unexpected success")
end`, method, server.URL))
				ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
				defer cancel()
				options := developmentOptions()
				options.Online = true
				start := time.Now()
				_, err := RunDevelopmentHook(ctx, dir, "Available", nil, options)
				if err == nil || !strings.Contains(err.Error(), "deadline exceeded") {
					t.Fatalf("timeout error = %v", err)
				}
				if time.Since(start) > 3*time.Second {
					t.Fatal("HTTP did not respect Lua deadline")
				}
			})
		}
	}
}

func TestDevelopmentRunOutputAndEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VFOX_PLUGIN_TEST_SECRET", "host")
	writePluginFile(t, dir, "main.lua", minimalPlugin+`
print("load-output")
function PLUGIN:PostInstall(ctx)
  assert(os.getenv("VFOX_PLUGIN_TEST_SECRET") == "host")
  print("print-output")
  io.write("write-output\\n")
  io.stdout:write("stdout-output\\n")
  io.stderr:write("stderr-output\\n")
  io.output(io.stdout)
  io.output():write("default-output\\n")
  assert(io.type(io.output()) == "file")
  assert(io.flush())
  assert(io.stdout:setvbuf("no"))
  assert(os.execute("echo child-output") == 0)
  local old = io.output()
  io.output(RUNTIME.pluginDirPath.."/file-output")
  io.write("file-content")
  assert(io.flush())
  assert(io.close())
  io.output(old)
end`)
	var output bytes.Buffer
	options := developmentOptions()
	options.Output = &output
	result, err := RunDevelopmentHook(context.Background(), dir, "PostInstall", []byte(`{}`), options)
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Fatalf("void hook = %v", result)
	}
	for _, want := range []string{"load-output", "print-output", "write-output", "stdout-output", "stderr-output", "default-output", "child-output"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("diagnostics missing %q: %s", want, &output)
		}
	}
	content, err := os.ReadFile(filepath.Join(dir, "file-output"))
	if err != nil || string(content) != "file-content" {
		t.Fatalf("normal file IO = %q, %v", content, err)
	}
}
