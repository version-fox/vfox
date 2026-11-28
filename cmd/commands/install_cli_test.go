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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
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

func TestInstallAllAddsMissingPluginAndInstallsSDK(t *testing.T) {
	const pluginTpl = `PLUGIN = {name=%q, version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ninja.json":
			fmt.Fprintf(w, `{"name":"ninja","downloadUrl":%q}`, srv.URL+"/plugins/ninja.lua")
		case "/plugins/ninja.lua":
			fmt.Fprintf(w, pluginTpl, "ninja")
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	t.Run("adds missing plugin and installs recorded SDK version", func(t *testing.T) {
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		f.writeUserConfig(t, "registry:\n  address: "+srv.URL+"\n")
		must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nninja = \"1.12.1\"\n"), 0644))

		out := f.run(t, "install", "--all", "--yes").mustSucceed(t, "install --all --yes").output()
		mustContain(t, "install --all output", out, "Plugin:", "ninja", "SDK:", "ninja@1.12.1")

		if !exists(filepath.Join(f.shared, "plugin", "ninja", "main.lua")) {
			t.Fatalf("expected ninja plugin to be installed")
		}
		if !exists(filepath.Join(f.shared, "cache", "ninja", "v-1.12.1", "ninja-1.12.1")) {
			t.Fatalf("expected ninja@1.12.1 SDK directory to be installed")
		}
		listOut := f.run(t, "list", "ninja").mustSucceed(t, "vfox list ninja").output()
		mustContain(t, "vfox list ninja", listOut, "1.12.1")
	})

	t.Run("exits non-zero when missing plugin cannot be added", func(t *testing.T) {
		f := newCLIFixtureWithoutRegistryGuard(t, false)
		f.writeUserConfig(t, "registry:\n  address: "+srv.URL+"\n")
		must(t, os.WriteFile(filepath.Join(f.work, ".vfox.toml"), []byte("[tools]\nmissing = \"1.0.0\"\n"), 0644))

		res := f.run(t, "install", "--all", "--yes").mustFail(t, "install --all --yes missing plugin")
		mustContain(t, "install --all failure output", res.output(), "Plugin:", "missing", "SDK:", "missing@1.0.0", "not found in remote registry")
	})
}
