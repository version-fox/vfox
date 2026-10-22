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
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/plugin"
)

func newLinkTestSDK(t *testing.T) (*impl, *testPlugin, string) {
	t.Helper()
	p := &testPlugin{}
	return newTestSDK(t, p), p, newExternalSDK(t, "tool")
}

// sameDir reports whether two paths lead to the same directory, however they
// are spelled (Windows junctions, symlinked temp directories).
func sameDir(t *testing.T, a, b string) bool {
	t.Helper()
	infoA, errA := os.Stat(a)
	infoB, errB := os.Stat(b)
	if errA != nil || errB != nil {
		t.Fatalf("stat %s: %v, stat %s: %v", a, errA, b, errB)
	}
	return os.SameFile(infoA, infoB)
}

func TestLinkRegistersAnExistingDirectory(t *testing.T) {
	s, _, external := newLinkTestSDK(t)
	mustLink(t, s, "17-system", external)

	main := s.mainRuntimePath("17-system")
	if _, isLink := linkTarget(main); !isLink {
		t.Fatalf("%s is not a link", main)
	}
	if !s.CheckRuntimeExist("17-system") {
		t.Error("a linked version must count as installed")
	}
	if got := s.InstalledList(); len(got) != 1 || got[0] != "17-system" {
		t.Errorf("InstalledList = %v", got)
	}
	pkg, err := s.GetRuntimePackage("17-system")
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Runtime.Path != main || pkg.Runtime.Name != "test-sdk" || pkg.PackagePath != s.packagePath("17-system") {
		t.Errorf("runtime package = %+v", pkg.Runtime)
	}

	inst, err := s.Installation("17-system")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Origin != OriginLinked || !sameDir(t, inst.Target, external) {
		t.Errorf("origin = %q, target = %q", inst.Origin, inst.Target)
	}
	r := inst.Receipt
	if r == nil || r.Origin != OriginLinked || r.Target != external || r.Requested != "17-system" ||
		r.SDK != "test-sdk" || r.Version != "17-system" || r.Vfox != "9.9.9" || r.Schema != receiptSchema {
		t.Errorf("receipt = %+v", r)
	}
	if !exists(binCommand(external, "tool")) {
		t.Error("linking must leave the SDK as it was")
	}
}

func TestLinkMakesTheTargetAbsolute(t *testing.T) {
	s, _, external := newLinkTestSDK(t)
	t.Chdir(filepath.Dir(external))
	mustLink(t, s, "rel", filepath.Base(external))

	inst, err := s.Installation("rel")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(inst.Receipt.Target) || !sameDir(t, inst.Receipt.Target, external) {
		t.Errorf("receipt target = %q, want the absolute path of %s", inst.Receipt.Target, external)
	}
}

func TestLinkedVersionWorksLikeAnyOther(t *testing.T) {
	s, _, external := newLinkTestSDK(t)
	mustLink(t, s, "17-system", external)

	// Scope links point at the version directory, which holds a link of its own.
	if err := s.CreateSymlinksForScope("17-system", env.Global); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(s.envContext.PathMeta.Working.GlobalSdkDir, "test-sdk", "bin")) {
		t.Error("the global link does not reach the linked SDK")
	}
	envs, err := s.EnvKeysForScope("17-system", env.Global)
	if err != nil {
		t.Fatal(err)
	}
	if commands := env.ListExecutables(envs.Paths.Slice()); len(commands) != 1 || commands[0].Name != "tool" {
		t.Errorf("commands on PATH = %+v, want tool", commands)
	}
}

func TestLinkRejectsADirectoryThatIsNotTheSDK(t *testing.T) {
	s, _, _ := newLinkTestSDK(t)
	empty := newExternalSDK(t) // no bin directory at all

	err := s.Link("nothing", empty, LinkOptions{})
	if err == nil || !strings.Contains(err.Error(), "does not look like") {
		t.Fatalf("err = %v, want the plugin's verdict", err)
	}
	if exists(s.packagePath("nothing")) || len(s.InstalledList()) != 0 {
		t.Error("a refused link must leave nothing behind")
	}
}

func TestLinkAcceptsAPluginThatOnlySetsVariables(t *testing.T) {
	p := &testPlugin{envKeys: func(ctx *plugin.EnvKeysHookCtx) ([]*plugin.EnvKeysHookResultItem, error) {
		return []*plugin.EnvKeysHookResultItem{{Key: "TEST_SDK_HOME", Value: ctx.Path}}, nil
	}}
	s := newTestSDK(t, p)
	mustLink(t, s, "vars", newExternalSDK(t))
}

func TestLinkShowsThePluginsError(t *testing.T) {
	p := &testPlugin{envKeys: func(ctx *plugin.EnvKeysHookCtx) ([]*plugin.EnvKeysHookResultItem, error) {
		return nil, errors.New("not my layout")
	}}
	s := newTestSDK(t, p)
	err := s.Link("x", newExternalSDK(t, "tool"), LinkOptions{})
	if err == nil || !strings.Contains(err.Error(), "not my layout") {
		t.Errorf("err = %v", err)
	}
}

func TestLinkValidatesTheVersionName(t *testing.T) {
	valid := []string{"17", "17.0.9", "21.0.3+9-tem", "system", "brew_x", "A1"}
	invalid := []string{"", "latest", "a/b", "a b", "../x", ".hidden", "-x", "a@b", `a\b`, "x:y"}
	for _, name := range valid {
		if !ValidVersionName(name) {
			t.Errorf("%q should be a usable name", name)
		}
	}
	for _, name := range invalid {
		if ValidVersionName(name) {
			t.Errorf("%q must not be a usable name", name)
		}
	}

	s, _, external := newLinkTestSDK(t)
	if err := s.Link("../escape", external, LinkOptions{}); err == nil {
		t.Error("a name that is a path must be refused")
	}
	if entries, _ := os.ReadDir(filepath.Dir(s.InstallPath)); len(entries) > 0 {
		t.Errorf("a refused name created %v", entries)
	}
}

func TestLinkNeedsAnExistingDirectory(t *testing.T) {
	s, _, external := newLinkTestSDK(t)
	if err := s.Link("gone", filepath.Join(external, "missing"), LinkOptions{}); err == nil {
		t.Error("a path that does not exist must be refused")
	}
	file := binCommand(external, "tool")
	if err := s.Link("file", file, LinkOptions{}); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("err = %v, want 'not a directory'", err)
	}
}

func TestLinkNeverReplacesAnInstalledVersion(t *testing.T) {
	s := newTestSDK(t, (&testPlugin{}).withCommand(t, "tool"))
	mustInstall(t, s, "1.0.0")

	for _, opts := range []LinkOptions{{}, {Force: true}} {
		err := s.Link("1.0.0", newExternalSDK(t, "tool"), opts)
		if err == nil || !strings.Contains(err.Error(), "already installed") {
			t.Errorf("Link(%+v) err = %v, want 'already installed'", opts, err)
		}
	}
	inst, err := s.Installation("1.0.0")
	if err != nil || inst.Origin != OriginInstalled {
		t.Errorf("the installed version was touched: %+v, %v", inst, err)
	}
	if !exists(binCommand(s.mainRuntimePath("1.0.0"), "tool")) {
		t.Error("the installed payload is gone")
	}
}

func TestLinkAgainNeedsForce(t *testing.T) {
	s, _, first := newLinkTestSDK(t)
	second := newExternalSDK(t, "tool")
	mustLink(t, s, "jdk", first)

	err := s.Link("jdk", second, LinkOptions{})
	if err == nil || !strings.Contains(err.Error(), "already linked") {
		t.Fatalf("err = %v, want 'already linked'", err)
	}
	inst, _ := s.Installation("jdk")
	if !sameDir(t, inst.Target, first) {
		t.Error("a refused link changed the target")
	}

	if err := s.Link("jdk", second, LinkOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	inst, err = s.Installation("jdk")
	if err != nil || !sameDir(t, inst.Target, second) || inst.Receipt.Target != second {
		t.Errorf("after --force: %+v, %v", inst, err)
	}
	if !exists(binCommand(first, "tool")) {
		t.Error("re-pointing a link must not touch the old target")
	}
}

func TestInstallOfALinkedVersionDoesNothing(t *testing.T) {
	s, p, external := newLinkTestSDK(t)
	mustLink(t, s, "jdk", external)
	if err := s.Install("jdk"); err != nil {
		t.Fatal(err)
	}
	if p.preInstallCalls != 0 {
		t.Errorf("PreInstall ran %d time(s); nothing may be downloaded for a linked version", p.preInstallCalls)
	}
}

func TestUninstallLeavesTheLinkedSDKAlone(t *testing.T) {
	p := &testPlugin{preUninstall: func(*plugin.PreUninstallHookCtx) error {
		t.Error("PreUninstall must not run for a version vfox does not own")
		return nil
	}}
	s := newTestSDK(t, p)
	external := newExternalSDK(t, "tool")
	data := filepath.Join(external, "lib", "data")
	if err := os.MkdirAll(filepath.Dir(data), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(data, []byte("precious"), 0644); err != nil {
		t.Fatal(err)
	}
	mustLink(t, s, "jdk", external)

	if err := s.Uninstall("jdk"); err != nil {
		t.Fatal(err)
	}
	if exists(s.packagePath("jdk")) || s.CheckRuntimeExist("jdk") {
		t.Error("the version is still there")
	}
	for _, path := range []string{binCommand(external, "tool"), data} {
		if !exists(path) {
			t.Errorf("uninstalling a linked version deleted %s", path)
		}
	}
	if content, _ := os.ReadFile(data); string(content) != "precious" {
		t.Error("the linked SDK's files were modified")
	}
}

func TestUninstallRemovesALinkWhoseTargetIsGone(t *testing.T) {
	s, _, external := newLinkTestSDK(t)
	mustLink(t, s, "jdk", external)
	if err := os.RemoveAll(external); err != nil {
		t.Fatal(err)
	}
	if s.CheckRuntimeExist("jdk") || !s.brokenLink("jdk") {
		t.Fatal("setup: the link should be broken")
	}

	if err := s.Uninstall("jdk"); err != nil {
		t.Fatalf("a broken link must still be removable: %v", err)
	}
	if exists(s.packagePath("jdk")) {
		t.Error("the broken link is still there")
	}
	if err := s.Uninstall("jdk"); err == nil {
		t.Error("uninstalling a version that is not there must fail")
	}
}

func TestUninstallTreatsAHandMadeLinkAsLinked(t *testing.T) {
	s, _, external := newLinkTestSDK(t)
	// What #669 describes: `ln -s` into the cache by hand, no receipt.
	if err := os.MkdirAll(s.packagePath("hand"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := env.CreateDirSymlink(external, s.mainRuntimePath("hand")); err != nil {
		t.Fatal(err)
	}
	inst, err := s.Installation("hand")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Origin != OriginLinked || inst.Receipt != nil || !sameDir(t, inst.Target, external) {
		t.Fatalf("installation = %+v", inst)
	}
	if err := s.Uninstall("hand"); err != nil {
		t.Fatal(err)
	}
	if !exists(binCommand(external, "tool")) || exists(s.packagePath("hand")) {
		t.Error("uninstalling a hand-made link must remove the link and keep the SDK")
	}
}

func TestUninstallNeverDeletesARealDirectoryItTakesForALink(t *testing.T) {
	s, _, external := newLinkTestSDK(t)
	mustLink(t, s, "jdk", external)

	// Somebody swapped the link for a directory with real content.
	main := s.mainRuntimePath("jdk")
	if err := env.RemoveDirSymlink(main); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(main, "bin"), 0755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(main, "bin", "keep")
	if err := os.WriteFile(keep, []byte("mine"), 0644); err != nil {
		t.Fatal(err)
	}

	err := s.Uninstall("jdk")
	if err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("err = %v, want a refusal", err)
	}
	if !exists(keep) {
		t.Error("the directory's content was deleted")
	}
}

func TestUninstallStillCleansInstalledVersions(t *testing.T) {
	p := (&testPlugin{preUninstall: func(*plugin.PreUninstallHookCtx) error { return nil }}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0.0")

	if err := s.Uninstall("1.0.0"); err != nil {
		t.Fatal(err)
	}
	if p.preUninstallCall != 1 {
		t.Errorf("PreUninstall ran %d time(s), want once", p.preUninstallCall)
	}
	if exists(s.packagePath("1.0.0")) {
		t.Error("the installation is still there")
	}
}

// A plugin may lay its own links out inside what it installs. The receipt says
// vfox owns that installation, so it is uninstalled like any other.
func TestALinkInsideAnInstalledVersionIsNotALinkedVersion(t *testing.T) {
	elsewhere := newExternalSDK(t, "tool")
	p := &testPlugin{
		preUninstall: func(*plugin.PreUninstallHookCtx) error { return nil },
		postInstall: func(ctx *plugin.PostInstallHookCtx) error {
			main := ctx.SdkInfo["test-sdk"].Path
			if err := os.Remove(main); err != nil {
				return err
			}
			return env.CreateDirSymlink(elsewhere, main)
		},
	}
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0.0")

	inst, err := s.Installation("1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Origin != OriginInstalled {
		t.Errorf("origin = %q, want installed", inst.Origin)
	}
	if err := s.Uninstall("1.0.0"); err != nil {
		t.Fatal(err)
	}
	if p.preUninstallCall != 1 {
		t.Error("PreUninstall must run for a version vfox installed")
	}
	if !exists(binCommand(elsewhere, "tool")) {
		t.Error("removing the version followed the link")
	}
}

func TestDetect(t *testing.T) {
	t.Run("plugin without the hook", func(t *testing.T) {
		s := newTestSDK(t, &testPlugin{})
		if _, err := s.Detect(); !errors.Is(err, ErrDetectNotSupported) {
			t.Errorf("err = %v, want ErrDetectNotSupported", err)
		}
	})
	t.Run("nothing found", func(t *testing.T) {
		s := newTestSDK(t, &testPlugin{detect: func(*plugin.DetectHookCtx) ([]*plugin.DetectHookResultItem, error) {
			return nil, plugin.ErrNoResultProvide
		}})
		found, err := s.Detect()
		if err != nil || len(found) != 0 {
			t.Errorf("found = %v, err = %v", found, err)
		}
	})
	t.Run("unusable entries are dropped", func(t *testing.T) {
		s := newTestSDK(t, &testPlugin{detect: func(*plugin.DetectHookCtx) ([]*plugin.DetectHookResultItem, error) {
			return []*plugin.DetectHookResultItem{
				{Version: "21.0.3-tem", Path: "/jdk/21", Note: "found"},
				{Version: "17", Path: ""},           // no path
				{Version: "latest", Path: "/jdk"},   // not a usable name
				{Version: "a/b", Path: "/jdk"},      // not a usable name
				{Version: "21.0.3-tem", Path: "/x"}, // name already taken by an earlier entry
				nil,
				{Version: "11", Path: "/jdk/11"},
			}, nil
		}})
		found, err := s.Detect()
		if err != nil {
			t.Fatal(err)
		}
		want := []DetectedRuntime{{Version: "21.0.3-tem", Path: "/jdk/21", Note: "found"}, {Version: "11", Path: "/jdk/11"}}
		if len(found) != len(want) || found[0] != want[0] || found[1] != want[1] {
			t.Errorf("found = %+v, want %+v", found, want)
		}
	})
	t.Run("hook error", func(t *testing.T) {
		s := newTestSDK(t, &testPlugin{detect: func(*plugin.DetectHookCtx) ([]*plugin.DetectHookResultItem, error) {
			return nil, errors.New("boom")
		}})
		if _, err := s.Detect(); err == nil || !strings.Contains(err.Error(), "boom") {
			t.Errorf("err = %v", err)
		}
	})
}

func TestAdoptWritesAReceiptForALegacyInstallation(t *testing.T) {
	p := &testPlugin{fingerprint: func(*plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error) {
		return &plugin.FingerprintHookResult{Facts: map[string]string{"head": "abc"}}, nil
	}}
	s := newTestSDK(t, p)
	// An installation from before receipts: just the payload directory.
	writeTestCommand(t, filepath.Join(s.mainRuntimePath("1.0.0"), "bin"), "tool")

	inst, err := s.Installation("1.0.0")
	if err != nil || inst.Receipt != nil {
		t.Fatalf("setup: %+v, %v", inst, err)
	}
	adopted, err := s.Adopt("1.0.0")
	if err != nil || !adopted {
		t.Fatalf("Adopt = (%v, %v)", adopted, err)
	}
	inst, _ = s.Installation("1.0.0")
	if inst.Receipt == nil || inst.Receipt.Origin != OriginInstalled || inst.Receipt.Facts["head"] != "abc" {
		t.Errorf("receipt = %+v", inst.Receipt)
	}
	if adopted, err := s.Adopt("1.0.0"); err != nil || adopted {
		t.Errorf("a second Adopt = (%v, %v), want (false, nil): a receipt is never overwritten", adopted, err)
	}
	if _, err := s.Adopt("9.9.9"); err == nil {
		t.Error("adopting a version that is not installed must fail")
	}
}

func TestAdoptKeepsALinkedVersionLinked(t *testing.T) {
	s, _, external := newLinkTestSDK(t)
	if err := os.MkdirAll(s.packagePath("hand"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := env.CreateDirSymlink(external, s.mainRuntimePath("hand")); err != nil {
		t.Fatal(err)
	}
	if adopted, err := s.Adopt("hand"); err != nil || !adopted {
		t.Fatalf("Adopt = (%v, %v)", adopted, err)
	}
	inst, _ := s.Installation("hand")
	if inst.Receipt.Origin != OriginLinked || inst.Receipt.Target == "" {
		t.Errorf("receipt = %+v", inst.Receipt)
	}
}

func TestLinkRefusesAVersionThatIsBeingInstalled(t *testing.T) {
	s, _, external := newLinkTestSDK(t)
	if err := os.MkdirAll(s.InstallPath, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.installMarkerPath("jdk"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	err := s.Link("jdk", external, LinkOptions{})
	if err == nil || !strings.Contains(err.Error(), "being installed") {
		t.Fatalf("err = %v, want a refusal that says the version is being installed", err)
	}
	if exists(s.packagePath("jdk")) {
		t.Error("a refused link must leave nothing behind")
	}
}

func TestLinkNeedsAnInstallationRoot(t *testing.T) {
	s, _, external := newLinkTestSDK(t)
	// Something that is not a directory where the installation root belongs.
	blocker := filepath.Join(t.TempDir(), "installs")
	if err := os.WriteFile(blocker, nil, 0644); err != nil {
		t.Fatal(err)
	}
	s.InstallPath = filepath.Join(blocker, "test-sdk")

	err := s.Link("jdk", external, LinkOptions{})
	if err == nil || !strings.Contains(err.Error(), "cannot link") {
		t.Errorf("err = %v, want the reason why nothing could be linked", err)
	}
}

func TestStripVersionPrefix(t *testing.T) {
	s, p, external := newLinkTestSDK(t)
	p.withCommand(t, "tool")
	mustInstall(t, s, "1.2.3")
	mustLink(t, s, "vendor", external)
	mustLink(t, s, "v7", newExternalSDK(t, "tool"))

	// Versions that cannot be used are still versions of that name: `doctor`
	// and `uninstall` have to find them.
	gone := newExternalSDK(t, "tool")
	mustLink(t, s, "vanished", gone)
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.installMarkerPath("vpending"), nil, 0600); err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name        string
		sdk         Sdk
		typed, want Version
	}{
		{"the usual v", s, "v1.2.3", "1.2.3"},
		{"no v", s, "1.2.3", "1.2.3"},
		{"a v that belongs to a linked version", s, "vendor", "vendor"},
		{"a linked version that is called like a prefixed one", s, "v7", "v7"},
		{"its spelling without the v is not that version", s, "7", "7"},
		{"a link whose target is gone", s, "vanished", "vanished"},
		{"an interrupted installation", s, "vpending", "vpending"},
		{"a v on something that is not installed", s, "v9.9.9", "9.9.9"},
		{"only one v", s, "vv1", "v1"},
		{"a lone v", s, "v", ""},
		{"nothing typed", s, "", ""},
		{"an SDK that is not known", nil, "vendor", "endor"},
	} {
		if got := StripVersionPrefix(tt.sdk, tt.typed); got != tt.want {
			t.Errorf("%s: StripVersionPrefix(%q) = %q, want %q", tt.name, tt.typed, got, tt.want)
		}
	}
}

func TestHasVersion(t *testing.T) {
	s, p, external := newLinkTestSDK(t)
	p.withCommand(t, "tool")
	mustInstall(t, s, "1.2.3")
	mustLink(t, s, "sys", external)
	if err := os.WriteFile(s.installMarkerPath("half"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(external); err != nil {
		t.Fatal(err)
	}

	for version, want := range map[Version]bool{
		"1.2.3": true,  // installed
		"sys":   true,  // linked, and its target is gone: it can no longer be used, but it is there
		"half":  true,  // an interrupted installation
		"9.9.9": false, // nothing
		"":      false,
	} {
		if got := s.HasVersion(version); got != want {
			t.Errorf("HasVersion(%q) = %v, want %v", version, got, want)
		}
	}
	if s.CheckRuntimeExist("sys") || s.CheckRuntimeExist("half") {
		t.Error("CheckRuntimeExist must stay the stricter question")
	}
}
