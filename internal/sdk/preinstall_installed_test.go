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
	"runtime"
	"slices"
	"strings"
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

// unreadable makes the version directory of version unlistable, as an installation
// owned by another user can be, and restores it when the test ends.
func unreadable(t *testing.T, s *impl, version Version) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not stop listing on Windows")
	}
	dir := s.packagePath(version)
	if err := os.Chmod(dir, 0311); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("permissions are not enforced for this user")
	}
}

// An installation whose directory cannot be listed is still installed, so it is
// still given to PreInstall. Reading it used to fail the whole list, which left the
// table empty and installed the release a second time.
func TestPreInstallIsGivenAnInstallationItCannotList(t *testing.T) {
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	mustInstall(t, s, "28.0")
	mustInstall(t, s, "OTP-29.1.2")
	unreadable(t, s, "OTP-29.1.2")

	var seen map[string]*plugin.InstalledPackageItem
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		seen = ctx.InstalledSdks
		return renamedRelease(ctx)
	}
	got, err := s.InstallResolved("29.1.2", false)
	if err != nil {
		t.Fatalf("InstallResolved(29.1.2): %v", err)
	}
	if got != "OTP-29.1.2" {
		t.Errorf("InstallResolved(29.1.2) = %q, want the installation OTP-29.1.2", got)
	}
	if exists(s.packagePath("29.1.2")) {
		t.Error("29.1.2 was installed a second time")
	}
	var names []string
	for name := range seen {
		names = append(names, name)
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"28.0", "OTP-29.1.2"}) {
		t.Errorf("PreInstall was given %v, want 28.0 and OTP-29.1.2", names)
	}
}

// PreUse and the matching after it see the same list, so one installation that
// cannot be listed does not hide the others, or itself.
func TestUseMatchesAnInstallationItCannotList(t *testing.T) {
	s := newTestSDK(t, (&testPlugin{}).withCommand(t, "tool"))
	mustInstall(t, s, "1.2.3")
	unreadable(t, s, "1.2.3")
	got, err := s.ResolveInstalledVersion("1.2")
	if err != nil || got != "1.2.3" {
		t.Errorf("ResolveInstalledVersion(1.2) = %q, %v; want 1.2.3", got, err)
	}
}

// An installation of the release under its other name that is still running is
// seen: one installation of an SDK runs at a time, from the list PreInstall is given
// until the version is on disk. Another one is refused, as an installation of the
// same version is, and once the first is done the request maps to it.
func TestInstallIsRefusedWhileTheSDKIsBeingInstalled(t *testing.T) {
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	p.preInstall = renamedRelease

	var concurrent error
	postInstall := p.postInstall
	p.postInstall = func(ctx *plugin.PostInstallHookCtx) error {
		if ctx.SdkInfo["test-sdk"].Version == "OTP-29.1.2" {
			// OTP-29.1.2 is being installed: its marker is there and it is not done.
			// The lock is taken through a file of its own, as another vfox would take it.
			_, concurrent = s.InstallResolved("29.1.2", false)
		}
		return postInstall(ctx)
	}
	mustInstall(t, s, "OTP-29.1.2")

	if concurrent == nil || !strings.Contains(concurrent.Error(), "while another installation of test-sdk runs") {
		t.Errorf("InstallResolved(29.1.2) during the installation of OTP-29.1.2 = %v, want it refused", concurrent)
	}
	if exists(s.packagePath("29.1.2")) {
		t.Error("29.1.2 was installed beside the installation that was running")
	}
	got, err := s.InstallResolved("29.1.2", false)
	if err != nil || got != "OTP-29.1.2" {
		t.Errorf("InstallResolved(29.1.2) afterwards = %q, %v; want the installation OTP-29.1.2", got, err)
	}
}

// A reinstall replaces the version as typed, so the plugin resolves it as it would
// without the list: with both 29.1.2 and OTP-29.1.2 installed, 29.1.2 is reinstalled.
func TestReinstallIsNotGivenTheInstalledVersions(t *testing.T) {
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	mustInstall(t, s, "29.1.2")
	mustInstall(t, s, "OTP-29.1.2")
	main := s.mainRuntimePath("29.1.2")
	if err := os.WriteFile(filepath.Join(main, "changed-in-place"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	var seen map[string]*plugin.InstalledPackageItem
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		seen = ctx.InstalledSdks
		return renamedRelease(ctx)
	}
	if err := s.Reinstall("29.1.2"); err != nil {
		t.Fatalf("Reinstall(29.1.2): %v", err)
	}
	if seen == nil || len(seen) != 0 {
		t.Errorf("PreInstall was given %v, want an empty table on a reinstall", seen)
	}
	if exists(filepath.Join(main, "changed-in-place")) || !s.CheckRuntimeExist("29.1.2") {
		t.Error("29.1.2 was not reinstalled")
	}
	if !s.CheckRuntimeExist("OTP-29.1.2") {
		t.Error("the reinstall of 29.1.2 removed OTP-29.1.2")
	}
}

// A plugin can return any name, including the name of a copy. A copy is selected
// only by its own name, so it cannot be reported as the version asked for.
func TestPreInstallCannotResolveToACopy(t *testing.T) {
	s, p := newCopyTestSDK(t)
	mustInstall(t, s, "3.12")
	if _, err := s.InstallAs("3.12", "3.12-db"); err != nil {
		t.Fatalf("InstallAs: %v", err)
	}
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: "3.12-db"}}, nil
	}
	got, err := s.InstallResolved("3.13", false)
	if err == nil || !strings.Contains(err.Error(), "test-sdk@3.12-db is a copy") {
		t.Fatalf("InstallResolved(3.13) = %q, %v; want a refusal that names the copy", got, err)
	}
	if exists(s.packagePath("3.13")) {
		t.Error("3.13 was installed")
	}
	if !s.isCopy("3.12-db") || !exists(binCommand(s.mainRuntimePath("3.12-db"), "tool")) {
		t.Error("the refused installation changed the copy")
	}
}

// A copy of a mapped version is still made: --as keeps a second copy on purpose, so
// the installation the request maps to is recorded as what the copy resolved to.
func TestInstallAsOfAMappedVersionMakesTheCopy(t *testing.T) {
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	mustInstall(t, s, "OTP-29.1.2")
	p.preInstall = renamedRelease
	postInstall := p.postInstall
	postInstalls := 0
	p.postInstall = func(ctx *plugin.PostInstallHookCtx) error {
		postInstalls++
		return postInstall(ctx)
	}

	got, err := s.InstallAs("29.1.2", "erl-db")
	if err != nil {
		t.Fatalf("InstallAs(29.1.2, erl-db): %v", err)
	}
	if got != "erl-db" || !s.CheckRuntimeExist("erl-db") {
		t.Errorf("InstallAs returned %q, want the copy erl-db on disk", got)
	}
	if postInstalls != 1 {
		t.Errorf("PostInstall ran %d times, want once for the copy", postInstalls)
	}
	if receipt := s.copyReceipt("erl-db"); receipt == nil || receipt.Resolved != "OTP-29.1.2" || receipt.Requested != "29.1.2" {
		t.Errorf("receipt of the copy = %+v, want requested 29.1.2 resolved to OTP-29.1.2", receipt)
	}
	if exists(s.packagePath("29.1.2")) {
		t.Error("29.1.2 was installed as well as the copy")
	}
}
