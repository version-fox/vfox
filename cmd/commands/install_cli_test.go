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
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/shared/util"
)

// #717: one command that installs and switches, instead of install followed by use.
func TestInstallCommandUseFlag(t *testing.T) {
	f := newCLIFixture(t, false)

	out := f.run(t, "install", "--use", "alpha@1.0.0").mustSucceed(t, "install --use").output()
	mustContain(t, "install --use", out, "Now using", "alpha@1.0.0")

	// The session scope now carries the SDK, which is what `vfox use` writes.
	env := f.run(t, "env", "-s", "bash").mustSucceed(t, "env").output()
	mustContain(t, "env", env, "test-session", "alpha")
}

// A plugin may install something other than what was asked for: the switch has to
// follow the version InstallResolved hands back, the one that is really on disk.
const rewritingPlugin = `PLUGIN = {name="beta", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="lts"}, {version="2.0.0"}} end
function PLUGIN:PreInstall(ctx)
  if ctx.version == "lts" then return {version="2.0.0"} end
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`

func TestInstallCommandUseFollowsTheInstalledVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "beta", rewritingPlugin)

	out := f.run(t, "install", "-u", "beta@lts").mustSucceed(t, "install -u beta@lts").output()
	mustContain(t, "install -u", out, "Now using", "beta@2.0.0")
	mustNotContain(t, "install -u", out, "lts is not installed")
}

func TestInstallCommandUseOnShellWithoutHookSaysWhatFailed(t *testing.T) {
	if runtime.GOOS == "windows" {
		// There, a shell without a hook still gets the global scope and a new
		// shell, which vfox opens for the user and which would wait for input.
		t.Skip("use succeeds on Windows without a hook")
	}
	f := newCLIFixture(t, false)
	// What a container build step gives vfox: no activated shell, hence no hook.
	f.env["__VFOX_SHELL"] = "-"

	r := f.run(t, "install", "-u", "alpha@1.0.0").mustFail(t, "install -u without a hook")
	mustContain(t, "error", r.output(), "installed, but switching failed", "Install")
	mustContain(t, "error", r.output(), "failed to switch alpha")
}

func TestInstallCommandWithoutUseFlagStaysOutOfTheSession(t *testing.T) {
	f := newCLIFixture(t, false)

	out := f.run(t, "install", "alpha@1.0.0").mustSucceed(t, "install").output()
	mustNotContain(t, "install", out, "Now using")
	mustNotContain(t, "env", f.run(t, "env", "-s", "bash").output(), "test-session")
}

func TestInstallCommandUseFlagRefusedWithAll(t *testing.T) {
	f := newCLIFixture(t, false)

	r := f.run(t, "install", "--all", "--use").mustFail(t, "install --all --use")
	mustContain(t, "error", r.output(), "--use cannot be combined with --all")
}

// The request may resolve to a different version from one run to the next: the
// switch has to follow the current resolution, not the one from an earlier run.
const movingPlugin = `PLUGIN = {name="beta", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="lts"}} end
function PLUGIN:PreInstall(ctx)
  if ctx.version == "lts" then return {version=os.getenv("BETA_LTS")} end
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`

func TestInstallCommandUseFollowsAMovingRequest(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "beta", movingPlugin)

	t.Setenv("BETA_LTS", "9.0.0")
	f.run(t, "install", "-u", "beta@lts").mustSucceed(t, "first install -u")

	t.Setenv("BETA_LTS", "10.0.0")
	f.run(t, "install", "-u", "beta@lts").mustSucceed(t, "second install -u")
	mustContain(t, "session config", readFile(t, filepath.Join(f.userHome, "tmp", "test-session", ".vfox.toml")), `beta = "10.0.0"`)
}

func TestInstallCommandUseWhenTheResolvedVersionIsAlreadyInstalled(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "beta", movingPlugin)
	t.Setenv("BETA_LTS", "2.0.0")

	f.run(t, "install", "beta@2.0.0").mustSucceed(t, "install beta@2.0.0")
	f.run(t, "install", "-u", "beta@lts").mustSucceed(t, "install -u beta@lts")
	mustContain(t, "session config", readFile(t, filepath.Join(f.userHome, "tmp", "test-session", ".vfox.toml")), `beta = "2.0.0"`)
}

// A second run of a setup script asks to use a version that is already on disk:
// that takes the early return in InstallResolved, which has to hand the version
// back like an install does.
func TestInstallCommandUseOnAnInstalledVersion(t *testing.T) {
	f := newCLIFixture(t, false)
	f.run(t, "install", "alpha@1.0.0").mustSucceed(t, "install alpha@1.0.0")
	out := f.run(t, "install", "-u", "alpha@1.0.0").mustSucceed(t, "install -u alpha@1.0.0").output()
	mustContain(t, "install -u", out, "already installed", "Now using")
	mustContain(t, "session config", readFile(t, filepath.Join(f.userHome, "tmp", "test-session", ".vfox.toml")), `alpha = "1.0.0"`)
}

func TestInstallCommandUseFlagRefusesTwoVersionsOfOneSDK(t *testing.T) {
	f := newCLIFixture(t, false)

	r := f.run(t, "install", "-u", "alpha@1.0.0", "alpha@2.0.0").mustFail(t, "install -u alpha twice")
	mustContain(t, "error", r.output(), "--use cannot switch two versions of alpha")
}

func TestInstallCommandUseFlagNeedsAVersion(t *testing.T) {
	f := newCLIFixture(t, false)

	r := f.run(t, "install", "-u", "alpha").mustFail(t, "install -u without a version")
	mustContain(t, "error", r.output(), "install --use needs a version for alpha")
}

// `vfox exec` installs missing versions on demand; when PreInstall resolves a
// request to a different version on disk (or finds that target already installed),
// loading its environment must follow the installed version rather than looking up
// the unresolved request string.
func TestExecFollowsTheInstalledVersion(t *testing.T) {
	t.Run("fresh install", func(t *testing.T) {
		f := newCLIFixture(t, false)
		f.writePlugin(t, "beta", rewritingPlugin)
		manager := f.manager(t)

		envs, err := resolveExecSDKEnv(manager, execSDKSpec{Name: "beta", Version: "lts"})
		if err != nil {
			t.Fatalf("resolveExecSDKEnv(beta@lts) = %v, want nil", err)
		}
		wantBin := filepath.Join(f.shared, "cache", "beta", "v-2.0.0", "beta-2.0.0", "bin")
		if !slices.Contains(envs.Paths.Slice(), wantBin) {
			t.Errorf("PATH = %v, want entry %q", envs.Paths.Slice(), wantBin)
		}
	})

	t.Run("already installed target", func(t *testing.T) {
		f := newCLIFixture(t, false)
		f.writePlugin(t, "beta", movingPlugin)
		t.Setenv("BETA_LTS", "2.0.0")
		f.run(t, "install", "beta@2.0.0").mustSucceed(t, "install beta@2.0.0")
		manager := f.manager(t)

		envs, err := resolveExecSDKEnv(manager, execSDKSpec{Name: "beta", Version: "lts"})
		if err != nil {
			t.Fatalf("resolveExecSDKEnv(beta@lts) = %v, want nil", err)
		}
		wantBin := filepath.Join(f.shared, "cache", "beta", "v-2.0.0", "beta-2.0.0", "bin")
		if !slices.Contains(envs.Paths.Slice(), wantBin) {
			t.Errorf("PATH = %v, want entry %q", envs.Paths.Slice(), wantBin)
		}
	})

	t.Run("moving target", func(t *testing.T) {
		f := newCLIFixture(t, false)
		f.writePlugin(t, "beta", movingPlugin)
		manager := f.manager(t)

		t.Setenv("BETA_LTS", "3.0.0")
		envs, err := resolveExecSDKEnv(manager, execSDKSpec{Name: "beta", Version: "lts"})
		if err != nil {
			t.Fatalf("resolveExecSDKEnv(beta@lts) = %v, want nil", err)
		}
		wantBin := filepath.Join(f.shared, "cache", "beta", "v-3.0.0", "beta-3.0.0", "bin")
		if !slices.Contains(envs.Paths.Slice(), wantBin) {
			t.Errorf("PATH = %v, want entry %q", envs.Paths.Slice(), wantBin)
		}
	})
}

const printingAllPlugin = `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx)
  print("preinstall-print")
  io.write("preinstall-iowrite\n")
  io.stdout:write("preinstall-stdout\n")
  io.stderr:write("preinstall-stderr\n")
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`

func TestInstallAllWithPluginPrint(t *testing.T) {
	f := newIsolatedCLIFixture(t, false)
	f.writePlugin(t, "alpha", printingAllPlugin)
	must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))
	res := f.runProcess(t, "install", "--all", "-y").mustSucceed(t, "install --all -y")
	mustNotContain(t, "install --all stdout", res.stdout, "write output", "preinstall-print", "preinstall-iowrite", "preinstall-stdout", "preinstall-stderr", "Preinstalling ", "Install alpha@1.0.0 success!", "Please use ")
	mustNotContain(t, "install --all stderr", res.stderr, "Preinstalling ", "Install alpha@1.0.0 success!", "Please use ")
	mustContain(t, "install --all stderr", res.stderr, "preinstall-print", "preinstall-iowrite", "preinstall-stdout", "preinstall-stderr")
	if _, err := os.Stat(filepath.Join(f.shared, "cache", "alpha", "v-1.0.0")); err != nil {
		t.Errorf("alpha@1.0.0 was not installed: %v\noutput:\n%s", err, res.output())
	}
}

func TestInstallAllNeverWritesClipboard(t *testing.T) {
	origIsTTY := util.IsTTY
	util.IsTTY = func() bool { return true }
	t.Cleanup(func() { util.IsTTY = origIsTTY })

	f := newIsolatedCLIFixture(t, false)
	f.writePlugin(t, "alpha", printingAllPlugin)
	marker := filepath.Join(f.root, "clipboard.called")
	writeStubClipboardTool(t, f.system, marker)
	must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))

	res := f.runHere(t, "install", "--all", "-y").mustSucceed(t, "install --all -y under forced TTY")
	mustNotContain(t, "install --all clipboard message", res.output(), "Copied to clipboard")
	if exists(marker) {
		t.Fatalf("install --all invoked clipboard utility unexpectedly (marker %q created)", marker)
	}
	if _, err := os.Stat(filepath.Join(f.shared, "cache", "alpha", "v-1.0.0")); err != nil {
		t.Fatalf("alpha@1.0.0 was not installed: %v\noutput:\n%s", err, res.output())
	}
}

func TestInstallAllReturnsErrorAndWritesFailureToStderr(t *testing.T) {
	f := newIsolatedCLIFixture(t, false)
	f.writePlugin(t, "alpha", `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx)
  error("preinstall boom")
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`)
	must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))
	res := f.runProcess(t, "install", "--all", "-y").mustFail(t, "install --all -y with failing PreInstall")
	mustNotContain(t, "install --all stdout", res.stdout, "Installation completed.", "preinstall boom")
	mustContain(t, "install --all stdout", res.stdout, "Installation finished with failures.")
	mustContain(t, "install --all stderr", res.stderr, "preinstall boom")
}

func TestInstallAllFailsWhenDevNullOpenFails(t *testing.T) {
	f := newIsolatedCLIFixture(t, false)
	f.writePlugin(t, "alpha", printingAllPlugin)
	must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))

	origOpen := openInstallAllDevNull
	openInstallAllDevNull = func() (*os.File, error) {
		return nil, errors.New("simulated devnull open failure")
	}
	t.Cleanup(func() { openInstallAllDevNull = origOpen })

	res := f.runHere(t, "install", "--all", "-y").mustFail(t, "install --all -y when devnull fails")
	mustContain(t, "install --all devnull error", res.output(), "simulated devnull open failure")
}

func TestInstallAllSuppressesDownloadProgress(t *testing.T) {
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	w, err := zw.Create("tool.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(bytes.Repeat([]byte("x"), 64*1024)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	zipBytes := archive.Bytes()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipBytes)))
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(zipBytes)
	}))
	defer srv.Close()

	f := newIsolatedCLIFixture(t, false)
	f.env["VFOX_TEST_DL_URL"] = srv.URL + "/alpha-1.0.0.zip"
	f.writePlugin(t, "alpha", `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx)
  io.stderr:write("preinstall-stderr\n")
  return {version=ctx.version, url=os.getenv("VFOX_TEST_DL_URL")}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`)
	must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))

	res := f.runProcess(t, "install", "--all", "-y").mustSucceed(t, "install --all -y download")
	mustContain(t, "install --all stderr", res.stderr, "preinstall-stderr")
	mustNotContain(t, "install --all stderr", res.stderr, "Downloading")
	if _, err := os.Stat(filepath.Join(f.shared, "cache", "alpha", "v-1.0.0", "alpha-1.0.0", "tool.txt")); err != nil {
		t.Fatalf("downloaded archive was not extracted: %v\noutput:\n%s", err, res.output())
	}
}

func TestInstallAllSuppressesLuaHTTPDownloadProgress(t *testing.T) {
	payload := bytes.Repeat([]byte("a"), 2*1024*1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		w.Header().Set("Content-Type", "application/octet-stream")
		for off := 0; off < len(payload); off += len(payload) / 4 {
			_, _ = w.Write(payload[off : off+len(payload)/4])
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(25 * time.Millisecond)
		}
	}))
	defer srv.Close()

	for _, tc := range []struct {
		name   string
		script string
	}{
		{
			name: "require at hook time",
			script: `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx)
  local dl = require("http")
  local err = dl.download_file({url=os.getenv("VFOX_TEST_LUA_DL_URL")}, os.getenv("VFOX_TEST_LUA_DL_DEST"))
  assert(err == nil, tostring(err))
  io.stderr:write("preinstall-stderr\n")
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`,
		},
		{
			name: "require at plugin load time",
			script: `local dl = require("http")
PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx)
  local err = dl.download_file({url=os.getenv("VFOX_TEST_LUA_DL_URL")}, os.getenv("VFOX_TEST_LUA_DL_DEST"))
  assert(err == nil, tostring(err))
  io.stderr:write("preinstall-stderr\n")
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newIsolatedCLIFixture(t, false)
			dest := filepath.Join(f.work, "payload.bin")
			f.env["VFOX_TEST_LUA_DL_URL"] = srv.URL + "/payload.bin"
			f.env["VFOX_TEST_LUA_DL_DEST"] = dest
			f.writePlugin(t, "alpha", tc.script)
			must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))

			res := f.runProcess(t, "install", "--all", "-y").mustSucceed(t, "install --all -y lua download_file")
			mustContain(t, "install --all stderr", res.stderr, "preinstall-stderr")
			mustNotContain(t, "install --all stderr", res.stderr, "Downloading", "payload.bin", "MiB", "KiB", "%")
			if res.stderr != "preinstall-stderr\n" {
				t.Fatalf("stderr = %q, want %q", res.stderr, "preinstall-stderr\n")
			}
			got, err := os.ReadFile(dest)
			must(t, err)
			if len(got) != len(payload) {
				t.Fatalf("downloaded %d bytes, want %d", len(got), len(payload))
			}
		})
	}
}

func TestInstallFailsOnMalformedConfig(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "install single sdk", args: []string{"install", "alpha@1.0.0"}},
		{name: "install all", args: []string{"install", "--all", "-y"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newIsolatedCLIFixture(t, false)
			must(t, os.WriteFile(filepath.Join(f.userHome, "config.yaml"), []byte("path:\n  mode: [\n"), 0644))
			res := f.runProcess(t, tc.args...).mustFail(t, tc.name+" with malformed config.yaml")
			mustContain(t, tc.name+" stderr", res.stderr, "load config failed:")
		})
	}
}

func TestInstallAllTopLevelAndExecLuaOutputGoesToStderr(t *testing.T) {
	t.Run("already-added plugin top-level print and io.write", func(t *testing.T) {
		f := newIsolatedCLIFixture(t, false)
		f.writePlugin(t, "alpha", `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
print("top-level-print")
io.write("top-level-iowrite\n")
io.stdout:write("top-level-stdout\n")
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`)
		must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))

		res := f.runProcess(t, "install", "--all", "-y").mustSucceed(t, "install --all -y with top-level Lua output")
		mustNotContain(t, "install --all stdout", res.stdout, "top-level-print", "top-level-iowrite", "top-level-stdout")
		mustContain(t, "install --all stderr", res.stderr, "top-level-print", "top-level-iowrite", "top-level-stdout")
	})

	t.Run("plugin added from registry top-level io.write", func(t *testing.T) {
		f := newIsolatedCLIFixture(t, false)
		var registry *httptest.Server
		registry = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/remote.json":
				fmt.Fprintf(w, `{"name":"remote","downloadUrl":%q}`, registry.URL+"/plugins/remote.lua")
			case "/plugins/remote.lua":
				fmt.Fprint(w, `PLUGIN = {name="remote", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
io.write("registry-top-iowrite\n")
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`)
			default:
				http.NotFound(w, r)
			}
		}))
		defer registry.Close()

		f.writeUserConfig(t, "registry:\n  address: "+registry.URL+"\n")
		must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nremote = \"1.0.0\"\n"), 0644))

		res := f.runProcess(t, "install", "--all", "-y").mustSucceed(t, "install --all -y adding plugin from registry")
		mustNotContain(t, "install --all stdout", res.stdout, "registry-top-iowrite", "Preinstalling ", "Install remote@1.0.0 success!", "Please use ")
		mustContain(t, "install --all stderr", res.stderr, "registry-top-iowrite")
		if _, err := os.Stat(filepath.Join(f.shared, "cache", "remote", "v-1.0.0")); err != nil {
			t.Fatalf("remote@1.0.0 was not installed after adding plugin from registry: %v\noutput:\n%s", err, res.output())
		}
	})

	t.Run("plugin missing from registry reports failure and exits non-zero", func(t *testing.T) {
		f := newIsolatedCLIFixture(t, false)
		registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer registry.Close()

		f.writeUserConfig(t, "registry:\n  address: "+registry.URL+"\n")
		must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nremote = \"1.0.0\"\n"), 0644))

		res := f.runProcess(t, "install", "--all", "-y").mustFail(t, "install --all -y when plugin add fails")
		mustNotContain(t, "install --all stdout on add failure", res.stdout, "Installation completed.", "SDK: remote@1.0.0 installing...")
		mustContain(t, "install --all stdout on add failure", res.stdout, "SDK: remote@1.0.0 skipped", "Installation finished with failures.")
	})

	t.Run("category-prefixed and mixed-case keys install plugin and sdk", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			tomlKey string
		}{
			{name: "category prefix", tomlKey: `"cat/remote"`},
			{name: "mixed case", tomlKey: "Remote"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				f := newIsolatedCLIFixture(t, false)
				var registry *httptest.Server
				registry = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/remote.json":
						fmt.Fprintf(w, `{"name":"remote","downloadUrl":%q}`, registry.URL+"/plugins/remote.lua")
					case "/plugins/remote.lua":
						fmt.Fprint(w, `PLUGIN = {name="remote", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`)
					default:
						http.NotFound(w, r)
					}
				}))
				defer registry.Close()

				f.writeUserConfig(t, "registry:\n  address: "+registry.URL+"\n")
				must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte(fmt.Sprintf("[tools]\n%s = \"1.0.0\"\n", tc.tomlKey)), 0644))

				res := f.runProcess(t, "install", "--all", "-y").mustSucceed(t, "install --all -y for "+tc.name)
				if _, err := os.Stat(filepath.Join(f.shared, "cache", "remote", "v-1.0.0")); err != nil {
					t.Fatalf("remote@1.0.0 was not installed for %s: %v\noutput:\n%s", tc.name, err, res.output())
				}
			})
		}
	})

	t.Run("PreInstall os.execute output from PATH tool", func(t *testing.T) {
		f := newIsolatedCLIFixture(t, false)
		toolPath := filepath.Join(f.system, commandFile("emit-marker"))
		toolScript := "#!/bin/sh\nprintf 'preinstall-exec-marker\\n'\n"
		if runtime.GOOS == "windows" {
			toolScript = "@echo off\r\necho preinstall-exec-marker\r\n"
		}
		must(t, os.WriteFile(toolPath, []byte(toolScript), 0755))

		f.writePlugin(t, "alpha", `PLUGIN = {name="alpha", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx)
  local code = os.execute("emit-marker")
  if code ~= 0 then
    error("emit-marker exited with " .. tostring(code))
  end
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`)
		must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644))

		res := f.runProcess(t, "install", "--all", "-y").mustSucceed(t, "install --all -y with PreInstall os.execute")
		mustNotContain(t, "install --all stdout", res.stdout, "preinstall-exec-marker")
		mustContain(t, "install --all stderr", res.stderr, "preinstall-exec-marker")
	})
}

func TestInstallAllNormalizesKeysAndPreservesErrorChain(t *testing.T) {
	t.Run("category-prefixed and mixed-case keys install without registry skip", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			tomlKey string
		}{
			{name: "category prefix", tomlKey: `"cat/remote"`},
			{name: "mixed case", tomlKey: "Remote"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				home := t.TempDir()
				vfoxHome := filepath.Join(home, "vfox")
				work := filepath.Join(home, "work")
				must(t, os.MkdirAll(vfoxHome, 0o755))
				must(t, os.MkdirAll(work, 0o755))

				var registry *httptest.Server
				registry = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/remote.json":
						fmt.Fprintf(w, `{"name":"remote","downloadUrl":%q}`, registry.URL+"/plugins/remote.lua")
					case "/plugins/remote.lua":
						fmt.Fprint(w, `PLUGIN = {name="remote", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`)
					default:
						http.NotFound(w, r)
					}
				}))
				defer registry.Close()

				userHome := filepath.Join(home, ".version-fox")
				must(t, os.MkdirAll(userHome, 0o755))
				must(t, os.WriteFile(filepath.Join(userHome, "config.yaml"), []byte("registry:\n  address: "+registry.URL+"\n"), 0o644))
				must(t, os.WriteFile(filepath.Join(work, ".vfox.toml"), []byte(fmt.Sprintf("[tools]\n%s = \"1.0.0\"\n", tc.tomlKey)), 0o644))

				t.Setenv("HOME", home)
				t.Setenv("USERPROFILE", home)
				t.Setenv("VFOX_HOME", vfoxHome)
				t.Chdir(work)

				if err := installAll(true); err != nil {
					t.Fatalf("installAll failed for %s: %v", tc.name, err)
				}
				if _, err := os.Stat(filepath.Join(vfoxHome, "cache", "remote", "v-1.0.0")); err != nil {
					t.Fatalf("cache/remote/v-1.0.0 missing for %s: %v", tc.name, err)
				}
			})
		}
	})

	t.Run("preserves errors.Is chain and skips missing plugin sdk without installing message", func(t *testing.T) {
		home := t.TempDir()
		vfoxHome := filepath.Join(home, "vfox")
		work := filepath.Join(home, "work")
		userHome := filepath.Join(home, ".version-fox")
		must(t, os.MkdirAll(vfoxHome, 0o755))
		must(t, os.MkdirAll(work, 0o755))
		must(t, os.MkdirAll(userHome, 0o755))

		registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		defer registry.Close()

		must(t, os.WriteFile(filepath.Join(userHome, "config.yaml"), []byte("registry:\n  address: "+registry.URL+"\n"), 0o644))
		must(t, os.WriteFile(filepath.Join(work, ".vfox.toml"), []byte("[tools]\nmissing = \"1.0.0\"\n"), 0o644))

		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		t.Setenv("VFOX_HOME", vfoxHome)
		t.Chdir(work)

		outFile, fErr := os.CreateTemp(t.TempDir(), "stdout-*")
		must(t, fErr)
		oldStdout := os.Stdout
		os.Stdout = outFile
		t.Cleanup(func() {
			os.Stdout = oldStdout
			_ = outFile.Close()
		})

		err := installAll(true)
		os.Stdout = oldStdout
		_ = outFile.Close()
		outBytes, rErr := os.ReadFile(outFile.Name())
		must(t, rErr)
		outStr := string(outBytes)

		if !errors.Is(err, internal.ManifestNotFound) {
			t.Fatalf("expected error chain containing internal.ManifestNotFound, got %v", err)
		}
		mustNotContain(t, "installAll stdout", outStr, "SDK: missing@1.0.0 installing...")
		mustContain(t, "installAll stdout", outStr, "SDK: missing@1.0.0 skipped", "Installation finished with failures.")
	})
}
