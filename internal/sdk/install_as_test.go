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
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/plugin"
)

// resolves312 is a PreInstall hook that resolves 3.12 to 3.12.7, the way plugins
// resolve a partial version, and returns any other version as it was asked for.
func resolves312(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
	version := ctx.Version
	if version == "3.12" {
		version = "3.12.7"
	}
	return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: version}}, nil
}

// newCopyTestSDK returns an SDK whose plugin resolves 3.12 to 3.12.7. Installations
// get a command in their bin directory, so that they look like real ones.
func newCopyTestSDK(t *testing.T) (*impl, *testPlugin) {
	t.Helper()
	p := (&testPlugin{preInstall: resolves312}).withCommand(t, "tool")
	return newTestSDK(t, p), p
}

func TestInstallAsKeepsACopyBesideTheVersion(t *testing.T) {
	s, p := newCopyTestSDK(t)
	mustInstall(t, s, "3.12")

	var requested string
	var seen *plugin.PostInstallHookCtx
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		requested = ctx.Version
		return resolves312(ctx)
	}
	postInstall := p.postInstall
	p.postInstall = func(ctx *plugin.PostInstallHookCtx) error {
		seen = ctx
		return postInstall(ctx)
	}

	got, err := s.InstallAs("3.12", "3.12-db")
	if err != nil {
		t.Fatalf("InstallAs: %v", err)
	}
	if got != "3.12-db" {
		t.Errorf("InstallAs returned %q, want the name of the copy", got)
	}
	// The plugin is asked for the version the user asked for, not for the name.
	if requested != "3.12" {
		t.Errorf("PreInstall was asked for %q, want 3.12", requested)
	}
	// PostInstall sees the copy as it is named, which is also how EnvKeys sees it.
	main := s.mainRuntimePath("3.12-db")
	if seen == nil || seen.SdkInfo["test-sdk"] == nil ||
		seen.SdkInfo["test-sdk"].Version != "3.12-db" || seen.SdkInfo["test-sdk"].Path != main {
		t.Errorf("PostInstall saw %+v, want the copy under its own name and directory", seen)
	}

	// Both installations are on disk, and each has its own directory.
	if !s.CheckRuntimeExist("3.12.7") || !s.CheckRuntimeExist("3.12-db") {
		t.Fatalf("both must exist: plain=%v copy=%v", s.CheckRuntimeExist("3.12.7"), s.CheckRuntimeExist("3.12-db"))
	}
	if want := filepath.Join(s.InstallPath, "v-3.12-db", "test-sdk-3.12-db"); main != want {
		t.Errorf("mainRuntimePath = %q, want %q", main, want)
	}
	if !exists(binCommand(main, "tool")) {
		t.Error("the copy has no command in its own directory")
	}
	if list := s.InstalledList(); !slices.Contains(list, "3.12-db") || !slices.Contains(list, "3.12.7") {
		t.Errorf("InstalledList = %v, want both", list)
	}

	inst, err := s.Installation("3.12-db")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Runtime.Version != "3.12-db" || inst.Runtime.Path != main || inst.Origin != OriginInstalled {
		t.Errorf("installation = %+v, want the copy under its name", inst.Runtime)
	}
	r := inst.Receipt
	if r == nil || r.Version != "3.12-db" || r.Requested != "3.12" || r.Resolved != "3.12.7" ||
		len(r.Sources) != 1 || r.Sources[0].Version != "3.12.7" {
		t.Errorf("receipt = %+v, want the request and the version it resolved to", r)
	}
	envs, err := s.EnvKeys(inst.RuntimePackage)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(envs.Paths.Slice(), filepath.Join(main, "bin")) {
		t.Errorf("PATH = %v, want the copy's own bin directory", envs.Paths.Slice())
	}

	// The plain installation is unchanged, and records nothing about a name.
	plain, err := s.Installation("3.12.7")
	if err != nil {
		t.Fatal(err)
	}
	if plain.Receipt == nil || plain.Receipt.Resolved != "" || plain.Receipt.Requested != "3.12" {
		t.Errorf("the plain installation's receipt = %+v", plain.Receipt)
	}
}

// A copy's name starts like the version it copies, so the lookup of a partial
// version must not take it for an installation of that version.
func TestInstallAsIsNotAMatchForItsVersion(t *testing.T) {
	s, _ := newCopyTestSDK(t)
	mustInstall(t, s, "3.12")
	if _, err := s.InstallAs("3.12", "3.12-db"); err != nil {
		t.Fatal(err)
	}

	if got, err := s.ResolveInstalledVersion("3.12"); err != nil || got != "3.12.7" {
		t.Errorf("ResolveInstalledVersion(3.12) = %q, %v; want the plain installation", got, err)
	}
	if got, err := s.ResolveInstalledVersion("3.12-db"); err != nil || got != "3.12-db" {
		t.Errorf("ResolveInstalledVersion(3.12-db) = %q, %v; want the copy", got, err)
	}
}

func TestInstallAsRefusesANameInUse(t *testing.T) {
	t.Run("installed copy", func(t *testing.T) {
		s, p := newCopyTestSDK(t)
		if _, err := s.InstallAs("3.12", "3.12-db"); err != nil {
			t.Fatal(err)
		}
		before := p.preInstallCalls
		_, err := s.InstallAs("3.12", "3.12-db")
		if err == nil || !strings.Contains(err.Error(), "3.12-db is already installed") {
			t.Fatalf("err = %v, want the name to be reported as taken", err)
		}
		if p.preInstallCalls != before {
			t.Error("a refused name must not reach PreInstall")
		}
	})

	t.Run("installed version", func(t *testing.T) {
		s, _ := newCopyTestSDK(t)
		mustInstall(t, s, "3.12.6")
		if _, err := s.InstallAs("3.12", "3.12.6"); err == nil || !strings.Contains(err.Error(), "3.12.6 is already installed") {
			t.Fatalf("err = %v, want the installed version to hold its name", err)
		}
	})

	t.Run("link", func(t *testing.T) {
		s, p := newCopyTestSDK(t)
		mustLink(t, s, "3.12-db", newExternalSDK(t, "tool"))
		before := p.preInstallCalls
		_, err := s.InstallAs("3.12", "3.12-db")
		if err == nil || !strings.Contains(err.Error(), "3.12-db is already linked to") {
			t.Fatalf("err = %v, want the link to hold its name", err)
		}
		if p.preInstallCalls != before {
			t.Error("a refused name must not reach PreInstall")
		}
	})
}

// A copy under the name of the version it resolves to would be the plain
// installation, and would shadow it for every later lookup.
func TestInstallAsRefusesTheVersionItself(t *testing.T) {
	s, _ := newCopyTestSDK(t)
	_, err := s.InstallAs("3.12", "3.12.7")
	if err == nil || !strings.Contains(err.Error(), "cannot be named 3.12.7") {
		t.Fatalf("err = %v, want the copy refused", err)
	}
	if exists(s.packagePath("3.12.7")) {
		t.Error("a refused copy must leave nothing behind")
	}
}

func TestInstallAsRejectsAnUnusableName(t *testing.T) {
	s, p := newCopyTestSDK(t)
	for _, name := range []Version{"", "latest", "a/b", "-x", ".."} {
		if _, err := s.InstallAs("3.12", name); err == nil {
			t.Errorf("InstallAs with name %q: no error", name)
		}
	}
	if p.preInstallCalls != 0 {
		t.Errorf("PreInstall was called %d times for unusable names", p.preInstallCalls)
	}
}

func TestInstallAsFailureFreesTheName(t *testing.T) {
	s, p := newCopyTestSDK(t)
	boom := errors.New("post-install failed")
	postInstall := p.postInstall
	p.postInstall = func(*plugin.PostInstallHookCtx) error { return boom }
	if _, err := s.InstallAs("3.12", "3.12-db"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want the plugin's error", err)
	}
	if s.CheckRuntimeExist("3.12-db") || exists(s.packagePath("3.12-db")) {
		t.Error("a failed copy must leave nothing behind")
	}

	p.postInstall = postInstall
	if _, err := s.InstallAs("3.12", "3.12-db"); err != nil {
		t.Fatalf("the name must be free after a failure: %v", err)
	}
	if !s.CheckRuntimeExist("3.12-db") {
		t.Error("the retried copy is not installed")
	}
}

// What an interrupted installation left under the name does not hold it: the
// next copy under that name replaces it.
func TestInstallAsReplacesAnInterruptedLeftover(t *testing.T) {
	t.Run("directory without a marker", func(t *testing.T) {
		s, _ := newCopyTestSDK(t)
		leftover := s.packagePath("3.12-db")
		if err := os.MkdirAll(leftover, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(leftover, "junk"), nil, 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := s.InstallAs("3.12", "3.12-db"); err != nil {
			t.Fatalf("InstallAs over a leftover: %v", err)
		}
		if exists(filepath.Join(leftover, "junk")) {
			t.Error("the leftover was not cleaned up")
		}
		if !s.CheckRuntimeExist("3.12-db") {
			t.Error("the copy is not installed")
		}
	})

	t.Run("marker of a killed installation", func(t *testing.T) {
		s, _ := newCopyTestSDK(t)
		if err := os.MkdirAll(s.InstallPath, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(s.installMarkerPath("3.12-db"), nil, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.InstallAs("3.12", "3.12-db"); err != nil {
			t.Fatalf("InstallAs over a killed installation: %v", err)
		}
		if exists(s.installMarkerPath("3.12-db")) {
			t.Error("the marker was not removed")
		}
		if !s.CheckRuntimeExist("3.12-db") {
			t.Error("the copy is not installed")
		}
	})
}

func TestUninstallOfACopyKeepsTheVersion(t *testing.T) {
	s, _ := newCopyTestSDK(t)
	mustInstall(t, s, "3.12")
	if _, err := s.InstallAs("3.12", "3.12-db"); err != nil {
		t.Fatal(err)
	}

	if err := s.Uninstall("3.12-db"); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if s.CheckRuntimeExist("3.12-db") || exists(s.packagePath("3.12-db")) {
		t.Error("the copy is still there")
	}
	if !s.CheckRuntimeExist("3.12.7") || !exists(binCommand(s.mainRuntimePath("3.12.7"), "tool")) {
		t.Error("uninstalling the copy removed the version it copies")
	}
}

// A copy the SDK changed in place is reinstalled from the version it was asked for,
// under its own name, and only when that still resolves to the version it has.
func TestReinstallOfACopyKeepsItsName(t *testing.T) {
	s, p := newCopyTestSDK(t)
	if _, err := s.InstallAs("3.12", "3.12-db"); err != nil {
		t.Fatal(err)
	}
	main := s.mainRuntimePath("3.12-db")
	if err := os.WriteFile(filepath.Join(main, "changed-in-place"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	var requested string
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		requested = ctx.Version
		return resolves312(ctx)
	}
	if err := s.Reinstall("3.12-db"); err != nil {
		t.Fatalf("Reinstall: %v", err)
	}
	if requested != "3.12" {
		t.Errorf("PreInstall was asked for %q, want the request 3.12", requested)
	}
	if exists(filepath.Join(main, "changed-in-place")) {
		t.Error("the reinstall kept what the SDK changed in place")
	}
	inst, err := s.Installation("3.12-db")
	if err != nil || inst.Receipt == nil || inst.Receipt.Resolved != "3.12.7" {
		t.Errorf("after the reinstall: installation = %+v, err = %v", inst, err)
	}

	// Now the request resolves to another version: the copy must stay as it is.
	p.preInstall = func(*plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: "3.12.8"}}, nil
	}
	err = s.Reinstall("3.12-db")
	if err == nil || !strings.Contains(err.Error(), "test-sdk@3.12-db is a copy of 3.12") ||
		!strings.Contains(err.Error(), "not the 3.12.7 it was installed as") {
		t.Fatalf("Reinstall = %v, want a refusal that names the copy and the version it was installed as", err)
	}
	if !exists(binCommand(main, "tool")) {
		t.Error("a refused reinstall changed the copy")
	}
}

// Another process can finish the same copy while this one waits in PreInstall. The
// name is checked again once the installation lock is held, and the copy that is
// there stays as it is.
func TestInstallAsRefusesANameTakenWhileItWasResolved(t *testing.T) {
	s, p := newCopyTestSDK(t)
	other := s.mainRuntimePath("3.12-db")
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		if err := os.MkdirAll(other, 0755); err != nil {
			t.Fatal(err)
		}
		// A file the other process left behind: it is gone if this installation
		// removed the directory, even though PostInstall would recreate a command.
		if err := os.WriteFile(filepath.Join(other, "kept"), nil, 0644); err != nil {
			t.Fatal(err)
		}
		return resolves312(ctx)
	}

	_, err := s.InstallAs("3.12", "3.12-db")
	if err == nil || !strings.Contains(err.Error(), "3.12-db is already installed") {
		t.Fatalf("err = %v, want the name to be reported as taken", err)
	}
	if !exists(filepath.Join(other, "kept")) {
		t.Error("the refused copy changed the installation that holds its name")
	}
	if exists(s.installMarkerPath("3.12-db")) {
		t.Error("the refused copy left an installation marker behind")
	}
}

// The same recheck covers a plain installation: a version that appears while
// PreInstall runs is the one returned, and it is not installed again.
func TestInstallReturnsAVersionInstalledWhileItWasResolved(t *testing.T) {
	s, p := newCopyTestSDK(t)
	other := s.mainRuntimePath("3.12.7")
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		if err := os.MkdirAll(other, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(other, "kept"), nil, 0644); err != nil {
			t.Fatal(err)
		}
		return resolves312(ctx)
	}

	got, err := s.InstallResolved("3.12", false)
	if err != nil {
		t.Fatalf("InstallResolved: %v", err)
	}
	if got != "3.12.7" {
		t.Errorf("InstallResolved returned %q, want the version that is installed", got)
	}
	if !exists(filepath.Join(other, "kept")) || exists(s.installMarkerPath("3.12.7")) {
		t.Error("the installation that was already there was changed")
	}
}

// A receipt that cannot be read is advisory. It must not stop a version from being
// reinstalled, which is how a broken installation gets repaired.
func TestReinstallSurvivesAnUnreadableReceipt(t *testing.T) {
	s := newTestSDK(t, (&testPlugin{}).withCommand(t, "tool"))
	mustInstall(t, s, "1.0.0")
	main := s.mainRuntimePath("1.0.0")
	if err := os.WriteFile(filepath.Join(main, "changed-in-place"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath(s.packagePath("1.0.0")), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := s.Reinstall("1.0.0"); err != nil {
		t.Fatalf("Reinstall with an unreadable receipt: %v", err)
	}
	if exists(filepath.Join(main, "changed-in-place")) {
		t.Error("the reinstall kept what the SDK changed in place")
	}
	inst, err := s.Installation("1.0.0")
	if err != nil || inst.Receipt == nil || inst.Receipt.Version != "1.0.0" {
		t.Errorf("after the reinstall: installation = %+v, err = %v; want a readable receipt", inst, err)
	}
}

// An additional runtime the plugin does not version keeps the add-<name> directory
// that it had before installations were kept under their names.
func TestInstallKeepsTheDirectoryOfAnUnversionedAddition(t *testing.T) {
	p := (&testPlugin{preInstall: func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		return &plugin.PreInstallHookResult{
			PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: ctx.Version},
			Addition:              []*plugin.PreInstallPackageItem{{Name: "tools"}},
		}, nil
	}}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0.0")

	if !exists(filepath.Join(s.packagePath("1.0.0"), "add-tools")) {
		t.Error("the unversioned addition has no add-tools directory")
	}
}
