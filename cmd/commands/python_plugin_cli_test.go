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
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
)

// pythonPlugin stands in for the python plugin in these tests. It answers the hooks
// of a real plugin the way a user meets them: a partial version resolves to the
// newest release it matches, a version file is answered by an installed release,
// and detection reports what the test describes. Nothing is downloaded, so an
// installation holds only its directory; pythonPayload adds the command.
const pythonPlugin = `PLUGIN = {name="python", version="0.6.3", description="simulated python plugin", author="test", minRuntimeVersion="0.0.0", legacyFilenames={".python-version"}}

-- The releases the plugin lists, newest first, the way a release index does.
local releases = {"3.13.1", "3.12.7", "3.12.6", "3.11.9"}

-- A request is a release or a prefix of one: "3" and "3.12" both match 3.12.7. The
-- newest release a request matches is the one it stands for.
local function matches(release, request)
  return release == request or string.sub(release, 1, string.len(request) + 1) == request .. "."
end

function PLUGIN:Available(ctx)
  local result = {}
  for _, release in ipairs(releases) do
    table.insert(result, {version=release})
  end
  return result
end

function PLUGIN:PreInstall(ctx)
  for _, release in ipairs(releases) do
    if matches(release, ctx.version) then
      return {version=release}
    end
  end
  error("no python release matches " .. ctx.version)
end

function PLUGIN:EnvKeys(ctx)
  return {{key="PATH", value=ctx.path .. "/bin"}}
end

-- A version file holds a request, which the newest installed release answers.
function PLUGIN:ParseLegacyFile(ctx)
  local file = io.open(ctx.filepath, "r")
  if file == nil then
    return {version=""}
  end
  local request = file:read("*l")
  file:close()
  if request == nil then
    return {version=""}
  end
  local installed = {}
  for _, version in ipairs(ctx.getInstalledVersions()) do
    installed[version] = true
  end
  for _, release in ipairs(releases) do
    if installed[release] and matches(release, request) then
      return {version=release}
    end
  end
  return {version=""}
end

-- Detect reports the installation the test describes in the environment, if any.
function PLUGIN:Detect(ctx)
  local dir = os.getenv("PYTHON_DETECT_DIR")
  if dir == nil then
    return {}
  end
  return {{version=os.getenv("PYTHON_DETECT_VERSION"), path=dir, note="found by the test"}}
end
`

// newPythonFixture is a vfox home with the simulated python plugin added.
func newPythonFixture(t *testing.T) *cliFixture {
	t.Helper()
	f := newCLIFixture(t, false)
	f.writePlugin(t, "python", pythonPlugin)
	return f
}

// pythonPayload puts the python command in the bin directory of an installation,
// as the installed release would hold it.
func pythonPayload(t *testing.T, f *cliFixture, version string) {
	t.Helper()
	bin := filepath.Join(f.shared, "cache", "python", "v-"+version, "python-"+version, "bin")
	must(t, os.MkdirAll(bin, 0755))
	writeCommand(t, bin, "python")
}

// enableLegacyFiles names the SDK in the record of the SDKs that read version files
// (~/.legacy_filenames), which is how vfox knows to ask the plugin about them.
func enableLegacyFiles(t *testing.T, f *cliFixture, sdk string) {
	t.Helper()
	must(t, os.WriteFile(filepath.Join(f.userHome, ".legacy_filenames"), []byte(sdk+" true\n"), 0644))
}

// assertLinkedTo checks that the link is a link into an installation whose directory
// name contains want. A scope's link is what the PATH of that scope points at, so this
// says which installation an environment uses.
func assertLinkedTo(t *testing.T, link, want string) {
	t.Helper()
	target, err := env.ReadDirSymlink(link)
	if err != nil {
		t.Fatalf("%s is not a link to an installation: %v", link, err)
	}
	if !strings.Contains(target, want) {
		t.Errorf("%s points to %s, want an installation of %s", link, target, want)
	}
}

// A partial version gets the newest release it matches, and the same request again
// finds that release installed. A release the plugin does not list is refused.
func TestPythonInstallResolvesAPartialVersion(t *testing.T) {
	f := newPythonFixture(t)

	out := f.run(t, "install", "python@3.12").mustSucceed(t, "install python@3.12").output()
	mustContain(t, "install python@3.12", out, "python@3.12.7")
	list := f.run(t, "list", "python").mustSucceed(t, "list python").output()
	mustContain(t, "list python", list, "3.12.7")

	again := f.run(t, "install", "python@3.12").mustSucceed(t, "install python@3.12 again").output()
	mustContain(t, "install python@3.12 again", again, "python@3.12.7 is already installed")

	latest := f.run(t, "install", "python@latest").mustSucceed(t, "install python@latest").output()
	mustContain(t, "install python@latest", latest, "Using latest version", "3.13.1")

	r := f.run(t, "install", "python@3.99").mustFail(t, "install python@3.99")
	mustContain(t, "install python@3.99", r.output(), "no python release matches 3.99")
}

// A copy is listed beside the version it copies. A partial version still means the
// installation, and the copy is used by its own name.
func TestPythonCopyBesideTheVersion(t *testing.T) {
	f := newPythonFixture(t)
	f.run(t, "install", "python@3.12").mustSucceed(t, "install python@3.12")
	f.run(t, "install", "--as", "3.12-db", "python@3.12").mustSucceed(t, "install --as 3.12-db")
	pythonPayload(t, f, "3.12.7")
	pythonPayload(t, f, "3.12-db")

	list := f.run(t, "list", "python").mustSucceed(t, "list python").output()
	mustContain(t, "list python", list, "3.12.7", "3.12-db (3.12.7)")

	global := filepath.Join(f.userHome, "sdks", "python")
	out := f.run(t, "use", "-g", "python@3.12").mustSucceed(t, "use -g python@3.12").output()
	mustContain(t, "use -g python@3.12", out, "Now using", "python@3.12.7")
	mustNotContain(t, "use -g python@3.12", out, "3.12-db")
	assertLinkedTo(t, global, "python-3.12.7")

	out = f.run(t, "use", "-g", "python@3.12-db").mustSucceed(t, "use -g python@3.12-db").output()
	mustContain(t, "use -g python@3.12-db", out, "Now using", "python@3.12-db")
	assertLinkedTo(t, global, "python-3.12-db")

	f.run(t, "doctor", "python").mustSucceed(t, "doctor python")
}

// A version file is answered by an installed release, and the plugin is not given the
// copy to answer with (sdk.TestPluginsAreNotGivenCopies checks that list directly).
func TestPythonLegacyFileChoosesAnInstallation(t *testing.T) {
	f := newPythonFixture(t)
	enableLegacyFiles(t, f, "python")
	f.run(t, "install", "python@3.12").mustSucceed(t, "install python@3.12")
	f.run(t, "install", "--as", "3.12-db", "python@3.12").mustSucceed(t, "install --as 3.12-db")
	pythonPayload(t, f, "3.12.7")
	pythonPayload(t, f, "3.12-db")
	must(t, os.WriteFile(filepath.Join(f.work, ".python-version"), []byte("3.12\n"), 0644))

	f.run(t, "env", "-s", "bash").mustSucceed(t, "env -s bash")
	// The project scope links the installation the file asks for, and env reads it.
	assertLinkedTo(t, filepath.Join(f.work, ".vfox", "sdks", "python"), "python-3.12.7")
}

// An alias can name a copy, and the copy's name cannot be taken by a link or a copy
// while it is an alias: that would hide the alias without a word.
func TestPythonAliasOfACopy(t *testing.T) {
	f := newPythonFixture(t)
	f.run(t, "install", "python@3.12").mustSucceed(t, "install python@3.12")
	f.run(t, "install", "--as", "3.12-db", "python@3.12").mustSucceed(t, "install --as 3.12-db")
	pythonPayload(t, f, "3.12-db")
	f.run(t, "alias", "set", "python", "cuda", "3.12-db").mustSucceed(t, "alias set python cuda 3.12-db")

	out := f.run(t, "use", "-g", "python@cuda").mustSucceed(t, "use -g python@cuda").output()
	mustContain(t, "use -g python@cuda", out, "Now using", "python@3.12-db")

	r := f.run(t, "link", "python@cuda", f.external(t, "python")).mustFail(t, "link python@cuda")
	mustContain(t, "link python@cuda", r.output(), "cuda is an alias of python@3.12-db")
	r = f.run(t, "install", "--as", "cuda", "python@3.12").mustFail(t, "install --as cuda")
	mustContain(t, "install --as cuda", r.output(), "cuda is an alias of python@3.12-db")

	aliases := f.run(t, "alias", "list", "python").mustSucceed(t, "alias list python").output()
	mustContain(t, "alias list python", aliases, "cuda")
}

// A link is registered under a name of the user's choice, which may be a version
// number as the documentation gives 17.0.9 as an example. uninstall removes the link
// and leaves the directory it points to.
func TestPythonLinkBesideTheInstallations(t *testing.T) {
	f := newPythonFixture(t)
	f.run(t, "install", "python@3.12").mustSucceed(t, "install python@3.12")
	pythonPayload(t, f, "3.12.7")
	system := f.external(t, "python")

	out := f.run(t, "link", "python@3.12-sys", system).mustSucceed(t, "link python@3.12-sys").output()
	mustContain(t, "link python@3.12-sys", out, "Linked", "python@3.12-sys")
	list := f.run(t, "list", "python").mustSucceed(t, "list python").output()
	mustContain(t, "list python", list, "3.12.7", "3.12-sys (linked → "+system+")")

	out = f.run(t, "use", "-g", "python@3.12-sys").mustSucceed(t, "use -g python@3.12-sys").output()
	mustContain(t, "use -g python@3.12-sys", out, "Now using", "python@3.12-sys")

	f.run(t, "link", "python@3.11.9", f.external(t, "python")).mustSucceed(t, "link python@3.11.9")

	out = f.run(t, "uninstall", "python@3.12-sys").mustSucceed(t, "uninstall python@3.12-sys").output()
	mustContain(t, "uninstall python@3.12-sys", out, "left untouched", "Auto switch", "python@3.12.7")
	if !exists(filepath.Join(system, "bin", commandFile("python"))) {
		t.Error("uninstalling the link removed the directory it points to")
	}
}

// A copy is uninstalled by its own name, and the version it copies stays. A copy can
// be reinstalled by its name as well.
func TestPythonCopyUninstallAndReinstall(t *testing.T) {
	f := newPythonFixture(t)
	f.run(t, "install", "python@3.12").mustSucceed(t, "install python@3.12")
	f.run(t, "install", "--as", "3.12-db", "python@3.12").mustSucceed(t, "install --as 3.12-db")

	f.run(t, "uninstall", "python@3.12-db").mustSucceed(t, "uninstall python@3.12-db")
	list := f.run(t, "list", "python").mustSucceed(t, "list python").output()
	mustContain(t, "list python", list, "3.12.7")
	mustNotContain(t, "list python", list, "3.12-db")

	f.run(t, "install", "--as", "3.12-db", "python@3.12").mustSucceed(t, "install --as 3.12-db again")
	out := f.run(t, "install", "--reinstall", "python@3.12-db").mustSucceed(t, "install --reinstall python@3.12-db").output()
	mustContain(t, "install --reinstall python@3.12-db", out, "python@3.12-db")
}

// link --detect offers what the plugin finds, and skips a name that is an alias of
// the SDK. A detected version number is offered, since it is a name a link may take.
func TestPythonDetectSkipsAnAlias(t *testing.T) {
	f := newPythonFixture(t)
	f.run(t, "install", "python@3.12").mustSucceed(t, "install python@3.12")
	f.run(t, "alias", "set", "python", "cuda", "3.12.7").mustSucceed(t, "alias set python cuda 3.12.7")
	found := f.external(t, "python")
	t.Setenv("PYTHON_DETECT_DIR", found)

	t.Setenv("PYTHON_DETECT_VERSION", "cuda")
	out := f.run(t, "link", "--detect", "-y", "python").mustSucceed(t, "link --detect cuda").output()
	mustContain(t, "link --detect cuda", out, "Skipping cuda", "an alias of that name already points to 3.12.7", "No new python installations found")

	t.Setenv("PYTHON_DETECT_VERSION", "3.11.9")
	out = f.run(t, "link", "--detect", "-y", "python").mustSucceed(t, "link --detect 3.11.9").output()
	mustContain(t, "link --detect 3.11.9", out, "Linked", "python@3.11.9")
}
