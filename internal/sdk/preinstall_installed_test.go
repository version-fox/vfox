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

package sdk

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/version-fox/vfox/internal/plugin"
)

// renamedRelease is a PreInstall hook for a plugin that renamed its versions: what
// was installed as OTP-29.1.2 is now called 29.1.2. When the old name is installed,
// it returns that name, so the release is not installed a second time.
func renamedRelease(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
	version := ctx.Version
	if _, ok := ctx.InstalledSdks["OTP-"+version]; ok {
		version = "OTP-" + version
	}
	return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: version}}, nil
}

// PreInstall is given the installed versions, keyed by version as in PreUse, so a
// plugin can map a new name to an installation made under an old one.
func TestPreInstallIsGivenTheInstalledVersions(t *testing.T) {
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	mustInstall(t, s, "OTP-29.1.2")

	var seen map[string]*plugin.InstalledPackageItem
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		seen = ctx.InstalledSdks
		return renamedRelease(ctx)
	}
	postInstall := p.postInstall
	postInstalls := 0
	p.postInstall = func(ctx *plugin.PostInstallHookCtx) error {
		postInstalls++
		return postInstall(ctx)
	}

	got, err := s.InstallResolved("29.1.2", false)
	if err != nil {
		t.Fatalf("InstallResolved(29.1.2): %v", err)
	}
	if got != "OTP-29.1.2" {
		t.Errorf("InstallResolved(29.1.2) = %q, want the installation OTP-29.1.2", got)
	}
	item := seen["OTP-29.1.2"]
	if len(seen) != 1 || item == nil {
		t.Fatalf("PreInstall was given %v, want OTP-29.1.2 only", seen)
	}
	if item.Version != "OTP-29.1.2" || item.Name != "test-sdk" || item.Path != s.packagePath("OTP-29.1.2") {
		t.Errorf("PreInstall was given %+v, want version OTP-29.1.2, name test-sdk, path %s", item, s.packagePath("OTP-29.1.2"))
	}
	if postInstalls != 0 {
		t.Errorf("PostInstall ran %d times, want the release left as installed", postInstalls)
	}
	if exists(s.packagePath("29.1.2")) {
		t.Errorf("29.1.2 was installed a second time")
	}
	if got := s.InstalledList(); !slices.Equal(got, []Version{"OTP-29.1.2"}) {
		t.Errorf("installed versions = %v, want OTP-29.1.2 only", got)
	}
}

// With nothing installed, PreInstall is given an empty table, not nil, so a plugin
// can walk it without a check.
func TestPreInstallIsGivenAnEmptyListBeforeTheFirstInstall(t *testing.T) {
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	var seen map[string]*plugin.InstalledPackageItem
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		seen = ctx.InstalledSdks
		return renamedRelease(ctx)
	}
	mustInstall(t, s, "29.1.2")
	if seen == nil || len(seen) != 0 {
		t.Errorf("PreInstall was given %v, want an empty map", seen)
	}
}

// A copy made with InstallAs is selected only by its own name, so PreInstall is not
// given it, as PreUse is not: a plugin returning it would pass the copy off as the
// version asked for.
func TestPreInstallIsNotGivenCopies(t *testing.T) {
	s, p := newCopyTestSDK(t)
	mustInstall(t, s, "3.12.7")
	if _, err := s.InstallAs("3.12", "3.12-db"); err != nil {
		t.Fatalf("InstallAs: %v", err)
	}
	var seen map[string]*plugin.InstalledPackageItem
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		seen = ctx.InstalledSdks
		return resolves312(ctx)
	}
	mustInstall(t, s, "3.13")
	var names []string
	for name := range seen {
		names = append(names, name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"3.12.7"}) {
		t.Errorf("PreInstall was given %v, want the installation of 3.12.7 only", names)
	}
}

// renamingPlugin is the Lua side of renamedRelease. It also walks the whole table,
// which fails if the table is missing.
const renamingPlugin = `PLUGIN = {name="erlang", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="29.1.2"}} end
function PLUGIN:PreInstall(ctx)
  for _ in pairs(ctx.installedSdks) do end
  local old = ctx.installedSdks["OTP-" .. ctx.version]
  if old ~= nil and old.version == "OTP-" .. ctx.version then
    return {version=old.version}
  end
  return {version=ctx.version}
end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`

// A Lua plugin reads the installed versions from ctx.installedSdks.
func TestLuaPreInstallReadsTheInstalledVersions(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	dir := filepath.Join(t.TempDir(), "erlang")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.lua"), []byte(renamingPlugin), 0644); err != nil {
		t.Fatal(err)
	}
	plug, err := plugin.CreatePlugin(dir, s.envContext)
	if err != nil {
		t.Fatalf("CreatePlugin: %v", err)
	}
	defer plug.Close()
	s.plugin = plug
	s.Name = "erlang"

	mustInstall(t, s, "OTP-29.1.2")
	got, err := s.InstallResolved("29.1.2", false)
	if err != nil {
		t.Fatalf("InstallResolved(29.1.2): %v", err)
	}
	if got != "OTP-29.1.2" {
		t.Errorf("InstallResolved(29.1.2) = %q, want the installation OTP-29.1.2", got)
	}
	if installed := s.InstalledList(); !slices.Equal(installed, []Version{"OTP-29.1.2"}) {
		t.Errorf("installed versions = %v, want OTP-29.1.2 only", installed)
	}
}
