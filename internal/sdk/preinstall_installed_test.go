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
	"time"

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

// unreadable takes permissions away from the version directory of version, as an
// installation owned by another user can, and restores them when the test ends.
// With mode 0311 the directory cannot be listed; with 0000 it cannot be entered.
func unreadable(t *testing.T, s *impl, version Version, mode os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not stop listing on Windows")
	}
	dir := s.packagePath(version)
	if err := os.Chmod(dir, mode); err != nil {
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
	unreadable(t, s, "OTP-29.1.2", 0311)

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

// An installation whose directory cannot be entered cannot be told apart from an
// empty directory, so it is not taken as absent: the plugin is given it, and an
// install that resolves to it is refused with the directory named, instead of
// installing the release a second time beside it.
func TestInstallRefusesAnInstallationItCannotEnter(t *testing.T) {
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	mustInstall(t, s, "28.0")
	mustInstall(t, s, "OTP-29.1.2")
	unreadable(t, s, "OTP-29.1.2", 0000)

	var seen map[string]*plugin.InstalledPackageItem
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		seen = ctx.InstalledSdks
		return renamedRelease(ctx)
	}
	got, err := s.InstallResolved("29.1.2", false)
	if err == nil || !strings.Contains(err.Error(), s.packagePath("OTP-29.1.2")) {
		t.Fatalf("InstallResolved(29.1.2) = %q, %v; want a refusal that names %s", got, err, s.packagePath("OTP-29.1.2"))
	}
	if exists(s.packagePath("29.1.2")) {
		t.Error("29.1.2 was installed a second time")
	}
	if _, ok := seen["OTP-29.1.2"]; !ok {
		t.Errorf("PreInstall was given %v, want OTP-29.1.2 among them", seen)
	}
	if err := os.Chmod(s.packagePath("OTP-29.1.2"), 0755); err != nil {
		t.Fatal(err)
	}
	if !s.CheckRuntimeExist("OTP-29.1.2") {
		t.Error("the refused installation changed OTP-29.1.2")
	}
}

// PreUse and the matching after it see the same list, so one installation that
// cannot be listed does not hide the others, or itself.
func TestUseMatchesAnInstallationItCannotList(t *testing.T) {
	s := newTestSDK(t, (&testPlugin{}).withCommand(t, "tool"))
	mustInstall(t, s, "1.2.3")
	unreadable(t, s, "1.2.3", 0311)
	got, err := s.ResolveInstalledVersion("1.2")
	if err != nil || got != "1.2.3" {
		t.Errorf("ResolveInstalledVersion(1.2) = %q, %v; want 1.2.3", got, err)
	}
}

// An installation of the release under its other name that is still running is
// seen: one installation of an SDK runs at a time, from the list PreInstall is given
// until the version is on disk. Another one waits for it, and then maps to it.
func TestInstallWaitsWhileTheSDKIsBeingInstalled(t *testing.T) {
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	p.preInstall = renamedRelease

	installing := make(chan struct{})
	finish := make(chan struct{})
	postInstall := p.postInstall
	p.postInstall = func(ctx *plugin.PostInstallHookCtx) error {
		if ctx.SdkInfo["test-sdk"].Version == "OTP-29.1.2" {
			asSeparateProcess()
			close(installing)
			<-finish
		}
		return postInstall(ctx)
	}
	first := make(chan error, 1)
	go func() { first <- s.Install("OTP-29.1.2") }()
	<-installing

	type result struct {
		version Version
		err     error
	}
	second := make(chan result, 1)
	go func() {
		// The lock is taken through a file of its own, as another vfox would take it.
		version, err := s.InstallResolved("29.1.2", false)
		second <- result{version, err}
	}()
	select {
	case r := <-second:
		t.Fatalf("InstallResolved(29.1.2) during the installation of OTP-29.1.2 = %q, %v; want it to wait", r.version, r.err)
	case <-time.After(20 * installLockPoll):
	}
	close(finish)
	if err := <-first; err != nil {
		t.Fatalf("Install(OTP-29.1.2): %v", err)
	}
	r := <-second
	if r.err != nil || r.version != "OTP-29.1.2" {
		t.Errorf("InstallResolved(29.1.2) after waiting = %q, %v; want the installation OTP-29.1.2", r.version, r.err)
	}
	if exists(s.packagePath("29.1.2")) {
		t.Error("29.1.2 was installed beside the installation that was running")
	}
}

// Waiting has a limit. After it the install fails and says why, and leaves nothing.
func TestInstallGivesUpWaitingForTheSDK(t *testing.T) {
	s := newTestSDK(t, (&testPlugin{}).withCommand(t, "tool"))
	wait := installLockWait
	installLockWait = 5 * installLockPoll
	t.Cleanup(func() { installLockWait = wait })
	held, err := acquireInstallLock(filepath.Join(s.InstallPath, sdkInstallLockName))
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()

	err = s.Install("2.0")
	if err == nil || !strings.Contains(err.Error(), "another installation of test-sdk is still running") {
		t.Errorf("Install(2.0) while the SDK lock is held = %v, want it to give up waiting", err)
	}
	if exists(s.packagePath("2.0")) {
		t.Error("2.0 was installed")
	}
}

// Only a lock another installation holds is reported as one. Any other failure to
// lock is returned as it is, as it was before the SDK had a lock.
func TestInstallReportsLockFailuresAsTheyAre(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not stop creating files on Windows")
	}
	s := newTestSDK(t, (&testPlugin{}).withCommand(t, "tool"))
	if err := os.MkdirAll(s.InstallPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(s.InstallPath, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(s.InstallPath, 0755) })
	if f, err := os.Create(filepath.Join(s.InstallPath, "probe")); err == nil {
		_ = f.Close()
		t.Skip("permissions are not enforced for this user")
	}

	err := s.Install("2.0")
	if err == nil || !strings.Contains(err.Error(), "cannot install test-sdk@2.0: open installation lock") || strings.Contains(err.Error(), "another installation") {
		t.Errorf("Install(2.0) into a read-only installation root = %v, want the error opening the lock, without another installation", err)
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

// fromSource is a PreInstall hook for a plugin without PostInstall whose payload
// is a file at a local path, and which maps a version to its installed OTP- name as
// renamedRelease does. A mapped answer has no path: there is nothing to fetch.
func fromSource(t *testing.T) func(*plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
	return func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		if _, ok := ctx.InstalledSdks["OTP-"+ctx.Version]; ok {
			return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: "OTP-" + ctx.Version}}, nil
		}
		source := filepath.Join(t.TempDir(), "erl")
		if err := os.WriteFile(source, []byte("erl "+ctx.Version), 0755); err != nil {
			return nil, err
		}
		return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: ctx.Version, Path: source}}, nil
	}
}

// --as keeps a second copy on purpose, so it is not given the installed versions: the
// plugin resolves the version as typed, and the copy gets its own payload instead of
// being made empty from a mapped answer that has nothing to fetch.
func TestInstallAsOfAMappedVersionGetsItsOwnPayload(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	p := s.plugin.Plugin.(*testPlugin)
	p.preInstall = fromSource(t)
	mustInstall(t, s, "OTP-29.1.2")

	var seen map[string]*plugin.InstalledPackageItem
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		seen = ctx.InstalledSdks
		return fromSource(t)(ctx)
	}
	got, err := s.InstallAs("29.1.2", "erl-db")
	if err != nil {
		t.Fatalf("InstallAs(29.1.2, erl-db): %v", err)
	}
	if got != "erl-db" || !exists(filepath.Join(s.mainRuntimePath("erl-db"), "erl")) {
		t.Errorf("InstallAs returned %q, want the copy erl-db with its payload", got)
	}
	if seen == nil || len(seen) != 0 {
		t.Errorf("PreInstall was given %v, want an empty table for --as", seen)
	}
	if receipt := s.copyReceipt("erl-db"); receipt == nil || receipt.Resolved != "29.1.2" || receipt.Requested != "29.1.2" {
		t.Errorf("receipt of the copy = %+v, want requested and resolved 29.1.2", receipt)
	}
	if exists(s.packagePath("29.1.2")) {
		t.Error("29.1.2 was installed as well as the copy")
	}
}

// A copy of a version a plugin maps is reinstalled under its name, as any copy is:
// the reinstall resolves the version as the copy did.
func TestReinstallOfACopyOfAMappedVersion(t *testing.T) {
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	mustInstall(t, s, "OTP-29.1.2")
	p.preInstall = renamedRelease
	if _, err := s.InstallAs("29.1.2", "erl-db"); err != nil {
		t.Fatalf("InstallAs(29.1.2, erl-db): %v", err)
	}
	main := s.mainRuntimePath("erl-db")
	if err := os.WriteFile(filepath.Join(main, "changed-in-place"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.Reinstall("erl-db"); err != nil {
		t.Fatalf("Reinstall(erl-db): %v", err)
	}
	if exists(filepath.Join(main, "changed-in-place")) || !s.isCopy("erl-db") {
		t.Error("erl-db was not reinstalled as a copy")
	}
	if !s.CheckRuntimeExist("OTP-29.1.2") || exists(s.packagePath("29.1.2")) {
		t.Error("the reinstall of erl-db changed the other installations")
	}
}

// blockPostInstall makes PostInstall of version wait until the returned finish is
// closed, and closes the returned installing when it starts waiting.
func blockPostInstall(p *testPlugin, version string) (installing, finish chan struct{}) {
	installing = make(chan struct{})
	finish = make(chan struct{})
	postInstall := p.postInstall
	p.postInstall = func(ctx *plugin.PostInstallHookCtx) error {
		if ctx.SdkInfo["test-sdk"].Version == version {
			asSeparateProcess()
			close(installing)
			<-finish
		}
		return postInstall(ctx)
	}
	return installing, finish
}

// shortLockWait makes waiting for another installation give up quickly.
func shortLockWait(t *testing.T) {
	wait := installLockWait
	installLockWait = 10 * installLockPoll
	t.Cleanup(func() { installLockWait = wait })
}

// The SDK lock covers deciding and claiming a name, not the download and
// PostInstall, so two versions of one SDK install side by side, as on main.
func TestInstallsOfTwoVersionsRunSideBySide(t *testing.T) {
	shortLockWait(t)
	p := (&testPlugin{}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	installing, finish := blockPostInstall(p, "1.0")
	first := make(chan error, 1)
	go func() { first <- s.Install("1.0") }()
	<-installing

	// 1.0 is still in PostInstall, and stays there until 2.0 is done.
	err := s.Install("2.0")
	close(finish)
	if err != nil {
		t.Errorf("Install(2.0) while 1.0 is being installed: %v", err)
	}
	if err := <-first; err != nil {
		t.Fatalf("Install(1.0): %v", err)
	}
	for _, version := range []Version{"1.0", "2.0"} {
		if !s.CheckRuntimeExist(version) {
			t.Errorf("%s is not installed", version)
		}
	}
}

// A PostInstall can install another version of its SDK, as a plugin does through
// os.execute. The nested install takes the locks through files of its own, as the
// child vfox would, so it must not wait on the installation that started it.
func TestPostInstallCanInstallAnotherVersionOfItsSDK(t *testing.T) {
	shortLockWait(t)
	p := (&testPlugin{}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	postInstall := p.postInstall
	var nested error
	p.postInstall = func(ctx *plugin.PostInstallHookCtx) error {
		if ctx.SdkInfo["test-sdk"].Version == "1.0" {
			nested = s.Install("2.0")
		}
		return postInstall(ctx)
	}
	mustInstall(t, s, "1.0")
	if nested != nil {
		t.Errorf("Install(2.0) from PostInstall of 1.0: %v", nested)
	}
	for _, version := range []Version{"1.0", "2.0"} {
		if !s.CheckRuntimeExist(version) {
			t.Errorf("%s is not installed", version)
		}
	}
}

// A request mapped to a version still being installed waits for that installation,
// with a limit. After it, the install fails, says why, and leaves nothing.
func TestInstallGivesUpWaitingForTheVersionItWasMappedTo(t *testing.T) {
	shortLockWait(t)
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	p.preInstall = renamedRelease
	installing, finish := blockPostInstall(p, "OTP-29.1.2")
	first := make(chan error, 1)
	go func() { first <- s.Install("OTP-29.1.2") }()
	<-installing

	got, err := s.InstallResolved("29.1.2", false)
	close(finish)
	if err == nil || !strings.Contains(err.Error(), "test-sdk@OTP-29.1.2 is still being installed") {
		t.Errorf("InstallResolved(29.1.2) while OTP-29.1.2 is being installed = %q, %v; want it to give up waiting", got, err)
	}
	if err := <-first; err != nil {
		t.Fatalf("Install(OTP-29.1.2): %v", err)
	}
	if exists(s.packagePath("29.1.2")) {
		t.Error("29.1.2 was installed beside the installation that was running")
	}
}

// A copy is left out of the list even when its directory cannot be entered, and
// so its receipt cannot be read.
func TestPreInstallIsNotGivenACopyItCannotEnter(t *testing.T) {
	s, p := newCopyTestSDK(t)
	mustInstall(t, s, "3.12")
	if _, err := s.InstallAs("3.12", "3.12-db"); err != nil {
		t.Fatalf("InstallAs: %v", err)
	}
	unreadable(t, s, "3.12-db", 0000)

	var seen map[string]*plugin.InstalledPackageItem
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		seen = ctx.InstalledSdks
		return resolves312(ctx)
	}
	mustInstall(t, s, "3.13")
	if _, ok := seen["3.12-db"]; ok {
		t.Errorf("PreInstall was given %v, want the copy 3.12-db left out", seen)
	}

	// A plugin that returns it anyway is told it is a copy.
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: "3.12-db"}}, nil
	}
	got, err := s.InstallResolved("3.14", false)
	if err == nil || !strings.Contains(err.Error(), "test-sdk@3.12-db is a copy") {
		t.Errorf("InstallResolved(3.14) = %q, %v; want a refusal that names the copy", got, err)
	}
}

// A copy still being installed is left out of the list too.
func TestPreInstallIsNotGivenACopyBeingInstalled(t *testing.T) {
	shortLockWait(t)
	s, p := newCopyTestSDK(t)
	installing, finish := blockPostInstall(p, "3.12-db")
	first := make(chan error, 1)
	go func() {
		_, err := s.InstallAs("3.12", "3.12-db")
		first <- err
	}()
	<-installing

	var seen map[string]*plugin.InstalledPackageItem
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		seen = ctx.InstalledSdks
		return resolves312(ctx)
	}
	err := s.Install("3.13")
	close(finish)
	if err != nil {
		t.Errorf("Install(3.13): %v", err)
	}
	if err := <-first; err != nil {
		t.Fatalf("InstallAs: %v", err)
	}
	if _, ok := seen["3.12-db"]; ok {
		t.Errorf("PreInstall was given %v, want the copy being installed left out", seen)
	}
}

// The mark of a copy goes with it, and a version installed under a name whose copy
// is gone is not taken for a copy because of a mark left behind.
func TestTheMarkOfACopyGoesWithIt(t *testing.T) {
	s, _ := newCopyTestSDK(t)
	if _, err := s.InstallAs("3.12", "3.12-db"); err != nil {
		t.Fatalf("InstallAs: %v", err)
	}
	if !exists(s.copyMarkerPath("3.12-db")) {
		t.Fatal("InstallAs left no mark beside the copy")
	}
	if err := s.Uninstall("3.12-db"); err != nil {
		t.Fatalf("Uninstall(3.12-db): %v", err)
	}
	if exists(s.copyMarkerPath("3.12-db")) {
		t.Error("Uninstall left the mark of the copy")
	}

	// A mark left behind, as by a removal that failed half way.
	if err := os.WriteFile(s.copyMarkerPath("3.13"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	mustInstall(t, s, "3.13")
	if s.isCopy("3.13") || exists(s.copyMarkerPath("3.13")) {
		t.Error("3.13 was installed as a version but is taken for a copy")
	}
}
