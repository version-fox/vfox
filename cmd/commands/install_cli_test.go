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
// follow the receipt to the version that is really on disk.
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
