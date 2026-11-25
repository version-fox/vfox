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
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/plugin"
)

// stubShellAndRegistry keeps a test from opening a shell and from writing the
// Windows registry of whoever runs it. It returns how many shells were opened.
func stubShellAndRegistry(t *testing.T) *int {
	t.Helper()
	opened := 0
	oldOpen, oldApply, oldRemove := openShell, env.ApplyEnvsToRegistry, env.RemoveEnvsFromRegistry
	openShell = func(int) error { opened++; return nil }
	env.ApplyEnvsToRegistry = func(*env.Envs) error { return nil }
	env.RemoveEnvsFromRegistry = func(*env.Envs) error { return nil }
	t.Cleanup(func() {
		openShell, env.ApplyEnvsToRegistry, env.RemoveEnvsFromRegistry = oldOpen, oldApply, oldRemove
	})
	return &opened
}

// Uninstall moves the global version with or without the hook, and never hands
// off to a new shell.
func TestUninstallOpensNoShell(t *testing.T) {
	for _, hook := range []string{"", "bash"} {
		t.Run("hook="+hook, func(t *testing.T) {
			s := installedSDK(t)
			opened := stubShellAndRegistry(t)
			capturePterm(t)
			mustInstall(t, s, "2.0.0")
			mustMakeGlobal(t, s, "2.0.0")
			t.Setenv(env.HookFlag, hook)

			if err := s.Uninstall("2.0.0", "1.0.0"); err != nil {
				t.Fatalf("Uninstall(2.0.0): %v", err)
			}
			if got := s.globalVersion(); got != "1.0.0" {
				t.Errorf("global version = %q, want 1.0.0", got)
			}
			if *opened != 0 {
				t.Errorf("opened %d shells, want none", *opened)
			}
		})
	}
}

// mustMakeGlobal makes version the global one, as vfox use -g does in a shell with the hook.
func mustMakeGlobal(t *testing.T, s *impl, version Version) {
	t.Helper()
	t.Setenv(env.HookFlag, "bash")
	if err := s.UseWithConfig(version, env.Global, false); err != nil {
		t.Fatalf("UseWithConfig(%s, global): %v", version, err)
	}
}

// use -g in a Windows shell without the hook still hands off to a new shell, so
// that the user gets the new version right away.
func TestUseWithoutTheHookOpensAShellOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("only Windows falls back to the global scope without the hook")
	}
	s := installedSDK(t)
	opened := stubShellAndRegistry(t)
	t.Setenv(env.HookFlag, "")
	capturePterm(t)

	if err := s.UseWithConfig("1.0.0", env.Global, false); err != nil {
		t.Fatalf("UseWithConfig(1.0.0, global): %v", err)
	}
	if *opened != 1 {
		t.Errorf("opened %d shells, want 1", *opened)
	}
}

// An error from the plugin's EnvKeys hook stops use, and the global config keeps
// the version it had.
func TestUseReturnsTheEnvKeysError(t *testing.T) {
	boom := false
	s := installedSDK(t, func(p *testPlugin) {
		p.envKeys = func(ctx *plugin.EnvKeysHookCtx) ([]*plugin.EnvKeysHookResultItem, error) {
			if boom {
				return nil, errors.New("EnvKeys exploded")
			}
			return []*plugin.EnvKeysHookResultItem{{Key: "PATH", Value: filepath.Join(ctx.Path, "bin")}}, nil
		}
	})
	opened := stubShellAndRegistry(t)
	t.Setenv(env.HookFlag, "bash")
	capturePterm(t)
	mustInstall(t, s, "2.0.0")
	if err := s.UseWithConfig("1.0.0", env.Global, false); err != nil {
		t.Fatalf("UseWithConfig(1.0.0, global): %v", err)
	}

	boom = true
	err := s.UseWithConfig("2.0.0", env.Global, false)
	if err == nil || !strings.Contains(err.Error(), "EnvKeys exploded") {
		t.Fatalf("UseWithConfig(2.0.0, global) = %v, want the EnvKeys error", err)
	}
	if got := s.globalVersion(); got != "1.0.0" {
		t.Errorf("global version = %q, want 1.0.0 unchanged", got)
	}
	if *opened != 0 {
		t.Errorf("opened %d shells, want none", *opened)
	}
}

// A version that is not installed is refused, and the global config keeps the
// version it had.
func TestUseRefusesAVersionThatIsNotInstalled(t *testing.T) {
	s := installedSDK(t)
	stubShellAndRegistry(t)
	capturePterm(t)

	mustMakeGlobal(t, s, "1.0.0")
	err := s.UseWithConfig("2.0.0", env.Global, false)
	var notExists *VersionNotExistsError
	if !errors.As(err, &notExists) {
		t.Fatalf("UseWithConfig(2.0.0, global) = %v, want a VersionNotExistsError", err)
	}
	if got := s.globalVersion(); got != "1.0.0" {
		t.Errorf("global version = %q, want 1.0.0 unchanged", got)
	}
}

const failingPreUsePlugin = `PLUGIN = {name="delta", version="0.0.1", description="test", author="test", minRuntimeVersion="0.0.0"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
function PLUGIN:PreUse(ctx) error("no version for you") end
`

// An error from the plugin's PreUse hook is returned, by use and by an uninstall
// that needs a version to take over, and the global config is not written.
func TestPreUseErrorLeavesTheGlobalVersionAlone(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	dir := filepath.Join(t.TempDir(), "delta")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.lua"), []byte(failingPreUsePlugin), 0644); err != nil {
		t.Fatal(err)
	}
	plug, err := plugin.CreatePlugin(dir, s.envContext)
	if err != nil {
		t.Fatalf("CreatePlugin: %v", err)
	}
	defer plug.Close()
	s.plugin = plug
	s.Name = "delta"
	stubShellAndRegistry(t)
	t.Setenv(env.HookFlag, "bash")
	capturePterm(t)
	mustInstall(t, s, "1.0.0")
	mustInstall(t, s, "2.0.0")

	err = s.UseWithConfig("1.0.0", env.Global, false)
	if err == nil || !strings.Contains(err.Error(), "no version for you") {
		t.Fatalf("UseWithConfig(1.0.0, global) = %v, want the PreUse error", err)
	}
	if got := s.globalVersion(); got != "" {
		t.Errorf("global version = %q, want none", got)
	}

	// apply does not call PreUse.
	if _, err := s.apply("2.0.0", env.Global, false); err != nil {
		t.Fatal(err)
	}
	err = s.Uninstall("2.0.0", "1.0.0")
	if err == nil || !strings.Contains(err.Error(), "no version for you") {
		t.Fatalf("Uninstall(2.0.0) = %v, want the PreUse error", err)
	}
	if got := s.globalVersion(); got != "2.0.0" || !s.CheckRuntimeExist("2.0.0") {
		t.Errorf("global version = %q, installed = %v, want 2.0.0 untouched", got, s.CheckRuntimeExist("2.0.0"))
	}
}

// When the config cannot be written, the links go back to the version it asks for.
func TestUseRevertsTheLinksWhenTheConfigCannotBeWritten(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only file does not stop the write")
	}
	s := installedSDK(t)
	stubShellAndRegistry(t)
	capturePterm(t)
	mustInstall(t, s, "2.0.0")
	mustMakeGlobal(t, s, "1.0.0")
	home := s.envContext.PathMeta.User.Home
	config := filepath.Join(home, ".vfox.toml")
	for _, path := range []string{config, home} {
		if err := os.Chmod(path, 0555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0755) })
	}

	if err := s.UseWithConfig("2.0.0", env.Global, false); err == nil {
		t.Fatal("UseWithConfig(2.0.0, global) succeeded, want an error from the config")
	}
	if got := s.globalVersion(); got != "1.0.0" {
		t.Errorf("global version = %q, want 1.0.0 unchanged", got)
	}
	target, err := os.Readlink(filepath.Join(s.envContext.PathMeta.Working.GlobalSdkDir, "test-sdk"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(target, "1.0.0") {
		t.Errorf("the link points at %s, want 1.0.0", target)
	}
}

// The PreUninstall hook runs only for a version that is going to be removed: not
// when no fallback can take over.
func TestUninstallChecksTheFallbacksBeforePreUninstall(t *testing.T) {
	boom := false
	p := &testPlugin{
		preUninstall: func(*plugin.PreUninstallHookCtx) error { return nil },
		envKeys: func(ctx *plugin.EnvKeysHookCtx) ([]*plugin.EnvKeysHookResultItem, error) {
			if boom && ctx.Main.Version == "1.0.0" {
				return nil, errors.New("EnvKeys exploded")
			}
			return []*plugin.EnvKeysHookResultItem{{Key: "PATH", Value: filepath.Join(ctx.Path, "bin")}}, nil
		},
	}
	s := newTestSDK(t, p)
	stubShellAndRegistry(t)
	capturePterm(t)
	mustInstall(t, s, "1.0.0")
	mustInstall(t, s, "2.0.0")
	mustMakeGlobal(t, s, "2.0.0")

	boom = true
	if err := s.Uninstall("2.0.0", "1.0.0"); err == nil || !strings.Contains(err.Error(), "EnvKeys exploded") {
		t.Fatalf("Uninstall(2.0.0) = %v, want the EnvKeys error", err)
	}
	if p.preUninstallCall != 0 {
		t.Errorf("PreUninstall ran %d times for a version that stays installed", p.preUninstallCall)
	}
	if !s.CheckRuntimeExist("2.0.0") || s.globalVersion() != "2.0.0" {
		t.Errorf("2.0.0 installed = %v, global version = %q, want it untouched", s.CheckRuntimeExist("2.0.0"), s.globalVersion())
	}

	boom = false
	if err := s.Uninstall("2.0.0", "1.0.0"); err != nil {
		t.Fatalf("Uninstall(2.0.0): %v", err)
	}
	if p.preUninstallCall != 1 {
		t.Errorf("PreUninstall ran %d times, want 1", p.preUninstallCall)
	}
	if got := s.globalVersion(); got != "1.0.0" {
		t.Errorf("global version = %q, want 1.0.0", got)
	}
}

// PreUninstall does not run when the global setting cannot be changed, and it runs
// once when a retry can change it. The registry is not touched before the config.
func TestUninstallWithAnUnwritableConfig(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only file does not stop the write")
	}
	for _, tc := range []struct {
		name      string
		fallbacks []Version
		want      Version
	}{
		{"with a fallback", []Version{"1.0.0"}, "1.0.0"},
		{"without a fallback", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &testPlugin{preUninstall: func(*plugin.PreUninstallHookCtx) error { return nil }}
			s := newTestSDK(t, p)
			stubShellAndRegistry(t)
			capturePterm(t)
			mustInstall(t, s, "1.0.0")
			mustInstall(t, s, "2.0.0")
			mustMakeGlobal(t, s, "2.0.0")
			registry := 0
			env.ApplyEnvsToRegistry = func(*env.Envs) error { registry++; return nil }
			env.RemoveEnvsFromRegistry = func(*env.Envs) error { registry++; return nil }
			home := s.envContext.PathMeta.User.Home
			config := filepath.Join(home, ".vfox.toml")
			for _, path := range []string{config, home} {
				if err := os.Chmod(path, 0555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(path, 0755) })
			}

			if err := s.Uninstall("2.0.0", tc.fallbacks...); err == nil || !strings.Contains(err.Error(), "nothing was removed") {
				t.Fatalf("Uninstall(2.0.0) = %v, want an error from the config", err)
			}
			if p.preUninstallCall != 0 {
				t.Errorf("PreUninstall ran %d times for a version that stays installed", p.preUninstallCall)
			}
			if registry != 0 {
				t.Errorf("the registry was changed %d times, want none before the config is saved", registry)
			}
			if !s.CheckRuntimeExist("2.0.0") || s.globalVersion() != "2.0.0" {
				t.Errorf("2.0.0 installed = %v, global version = %q, want it untouched", s.CheckRuntimeExist("2.0.0"), s.globalVersion())
			}

			_ = os.Chmod(home, 0755)
			_ = os.Chmod(config, 0644)
			if err := s.Uninstall("2.0.0", tc.fallbacks...); err != nil {
				t.Fatalf("Uninstall(2.0.0) again: %v", err)
			}
			if p.preUninstallCall != 1 {
				t.Errorf("PreUninstall ran %d times, want 1", p.preUninstallCall)
			}
			if got := s.globalVersion(); got != tc.want {
				t.Errorf("global version = %q, want %q", got, tc.want)
			}
		})
	}
}

// When PreUninstall fails, the global setting and the links go back to the version.
func TestUninstallRestoresTheGlobalVersionWhenPreUninstallFails(t *testing.T) {
	for _, tc := range []struct {
		name      string
		fallbacks []Version
	}{
		{"with a fallback", []Version{"1.0.0"}},
		{"without a fallback", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			boom := true
			p := &testPlugin{preUninstall: func(*plugin.PreUninstallHookCtx) error {
				if boom {
					return errors.New("PreUninstall refused")
				}
				return nil
			}}
			s := newTestSDK(t, p)
			stubShellAndRegistry(t)
			capturePterm(t)
			mustInstall(t, s, "1.0.0")
			mustInstall(t, s, "2.0.0")
			mustMakeGlobal(t, s, "2.0.0")
			link := filepath.Join(s.envContext.PathMeta.Working.GlobalSdkDir, "test-sdk")

			if err := s.Uninstall("2.0.0", tc.fallbacks...); err == nil || !strings.Contains(err.Error(), "PreUninstall refused") {
				t.Fatalf("Uninstall(2.0.0) = %v, want the PreUninstall error", err)
			}
			if got := s.globalVersion(); got != "2.0.0" {
				t.Errorf("global version = %q, want 2.0.0 again", got)
			}
			if target, err := env.ReadDirSymlink(link); err != nil || !strings.Contains(target, "2.0.0") {
				t.Errorf("the link points at %q (%v), want 2.0.0", target, err)
			}

			boom = false
			if err := s.Uninstall("2.0.0", tc.fallbacks...); err != nil {
				t.Fatalf("Uninstall(2.0.0) again: %v", err)
			}
			if got := s.globalVersion(); got == "2.0.0" {
				t.Error("the global version still names the removed version")
			}
		})
	}
}

// When the global setting cannot be set back either, the error says what it is
// and how to fix it, and the links agree with the config.
func TestUninstallSaysWhatTheGlobalVersionIsWhenItCannotBeRestored(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only directory does not stop the write")
	}
	for _, tc := range []struct {
		name      string
		fallbacks []Version
		lockLinks bool
		global    Version
		says      string
		link      string
	}{
		{"the config, with a fallback", []Version{"1.0.0"}, false, "1.0.0", "the global version is test-sdk@1.0.0 now", "1.0.0"},
		{"the config, without a fallback", nil, false, "", "there is no global version now", ""},
		{"the links, with a fallback", []Version{"1.0.0"}, true, "1.0.0", "the global version is test-sdk@1.0.0 now", "1.0.0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var s *impl
			p := &testPlugin{preUninstall: func(*plugin.PreUninstallHookCtx) error {
				home := s.envContext.PathMeta.User.Home
				paths := []string{filepath.Join(home, ".vfox.toml"), home, s.InstallPath}
				if tc.lockLinks {
					paths = []string{s.envContext.PathMeta.Working.GlobalSdkDir, s.InstallPath}
				}
				for _, path := range paths {
					if err := os.Chmod(path, 0555); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = os.Chmod(path, 0755) })
				}
				return nil
			}}
			s = newTestSDK(t, p)
			stubShellAndRegistry(t)
			capturePterm(t)
			mustInstall(t, s, "1.0.0")
			mustInstall(t, s, "2.0.0")
			mustMakeGlobal(t, s, "2.0.0")

			err := s.Uninstall("2.0.0", tc.fallbacks...)
			if err == nil {
				t.Fatal("Uninstall(2.0.0) succeeded, want the error of the rename")
			}
			for _, want := range []string{"its PreUninstall hook has already run", tc.says, "vfox use -g test-sdk@2.0.0"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Uninstall(2.0.0) = %v, want it to say %q", err, want)
				}
			}
			if got := s.globalVersion(); got != tc.global {
				t.Errorf("global version = %q, want %q", got, tc.global)
			}
			if !s.CheckRuntimeExist("2.0.0") {
				t.Error("2.0.0 is gone")
			}
			target, err := env.ReadDirSymlink(filepath.Join(s.envContext.PathMeta.Working.GlobalSdkDir, "test-sdk"))
			if tc.link == "" {
				if !os.IsNotExist(err) {
					t.Errorf("the link points at %q (%v), want no link", target, err)
				}
			} else if err != nil || !strings.Contains(target, tc.link) {
				t.Errorf("the link points at %q (%v), want %s", target, err, tc.link)
			}
		})
	}
}

// The global entry comes back with its attributes when the version stays
// installed, and the links of the scope are those from before: the ones the
// fallback replaced, and none of the ones only the fallback has. What another SDK
// does to its links stays.
func TestUninstallRestoresTheAttributesAndTheLinks(t *testing.T) {
	for _, tc := range []struct {
		name      string
		fallbacks []Version
	}{
		{"with a fallback", []Version{"1.0.0"}},
		{"without a fallback", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			moved := t.TempDir()
			var s *impl
			p := &testPlugin{preUninstall: func(*plugin.PreUninstallHookCtx) error {
				// Another SDK moves its link in the meantime.
				if err := env.CreateDirSymlink(moved, filepath.Join(s.envContext.PathMeta.Working.GlobalSdkDir, "another")); err != nil {
					t.Error(err)
				}
				return errors.New("PreUninstall refused")
			}}
			s = newTestSDK(t, p)
			stubShellAndRegistry(t)
			capturePterm(t)
			mustInstall(t, s, "1.0.0")
			mustInstall(t, s, "2.0.0")
			for _, addition := range []string{"v-1.0.0/only-old-1.0.0", "v-1.0.0/both-1.0.0", "v-1.0.0/other-1.0.0", "v-2.0.0/both-2.0.0", "v-2.0.0/only-new-2.0.0"} {
				if err := os.MkdirAll(filepath.Join(s.InstallPath, filepath.FromSlash(addition)), 0755); err != nil {
					t.Fatal(err)
				}
			}
			mustMakeGlobal(t, s, "2.0.0")
			linkDir := s.envContext.PathMeta.Working.GlobalSdkDir
			elsewhere := t.TempDir()
			for _, name := range []string{"other", "another"} { // "other" is also an addition of the fallback
				if err := env.CreateDirSymlink(elsewhere, filepath.Join(linkDir, name)); err != nil {
					t.Fatal(err)
				}
			}
			config := filepath.Join(s.envContext.PathMeta.User.Home, ".vfox.toml")
			if err := os.WriteFile(config, []byte("[tools]\ntest-sdk = { version = \"2.0.0\", vendor = \"openjdk\" }\n"), 0644); err != nil {
				t.Fatal(err)
			}
			links, err := s.linksOfScope(env.Global)
			if err != nil {
				t.Fatal(err)
			}
			if len(links) != 5 {
				t.Fatalf("the links are %v, want test-sdk, both, only-new, other and another", links)
			}
			links["another"] = moved

			if err := s.Uninstall("2.0.0", tc.fallbacks...); err == nil || !strings.Contains(err.Error(), "PreUninstall refused") {
				t.Fatalf("Uninstall(2.0.0) = %v, want the PreUninstall error", err)
			}
			global, err := s.envContext.LoadVfoxTomlByScope(env.Global)
			if err != nil {
				t.Fatal(err)
			}
			if entry, ok := global.GetToolConfig("test-sdk"); !ok || entry.Version != "2.0.0" || entry.Attr["vendor"] != "openjdk" {
				t.Errorf("the global entry is %+v, want 2.0.0 with its vendor", entry)
			}
			after, err := s.linksOfScope(env.Global)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(after, links) {
				t.Errorf("the links are\n%v\nwant\n%v", after, links)
			}
		})
	}
}

// The global version is linked and its target is gone, so it has no runtime
// package to make links from. Its global link comes back all the same when the
// version cannot be removed, when the fallback cannot be written to the config, and
// when a use cannot be.
func TestTheLinkOfABrokenVersionIsRestored(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a read-only directory does not stop the write")
	}
	for _, tc := range []struct {
		name string
		lock func(s *impl) []string
		run  func(s *impl) error
	}{
		{
			"the removal fails",
			func(s *impl) []string { return []string{s.InstallPath} },
			func(s *impl) error { return s.Uninstall("sys", "1.0.0") },
		},
		{
			"the switch fails",
			func(s *impl) []string {
				return []string{filepath.Join(s.envContext.PathMeta.User.Home, ".vfox.toml"), s.envContext.PathMeta.User.Home}
			},
			func(s *impl) error { return s.Uninstall("sys", "1.0.0") },
		},
		{
			"a use fails",
			func(s *impl) []string {
				return []string{filepath.Join(s.envContext.PathMeta.User.Home, ".vfox.toml"), s.envContext.PathMeta.User.Home}
			},
			func(s *impl) error { return s.UseWithConfig("1.0.0", env.Global, false) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, external := newLinkTestSDK(t)
			stubShellAndRegistry(t)
			capturePterm(t)
			mustInstall(t, s, "1.0.0")
			mustLink(t, s, "sys", external)
			mustMakeGlobal(t, s, "sys")
			link := filepath.Join(s.envContext.PathMeta.Working.GlobalSdkDir, "test-sdk")
			before, err := env.ReadDirSymlink(link)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.RemoveAll(external); err != nil {
				t.Fatal(err)
			}
			for _, path := range tc.lock(s) {
				if err := os.Chmod(path, 0555); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(path, 0755) })
			}

			err = tc.run(s)
			if err == nil {
				t.Fatal("it succeeded, want an error")
			}
			if strings.Contains(err.Error(), "restore") || strings.Contains(err.Error(), "could not be set back") {
				t.Errorf("the error says the old state did not come back: %v", err)
			}
			if got := s.globalVersion(); got != "sys" {
				t.Errorf("global version = %q, want sys", got)
			}
			if after, err := env.ReadDirSymlink(link); err != nil || after != before {
				t.Errorf("the link points at %q (%v), want %q", after, err, before)
			}
		})
	}
}

// A fallback that PreUse resolves to the version being removed cannot take over,
// and PreUninstall does not run.
func TestUninstallRefusesAFallbackThatResolvesToTheRemovedVersion(t *testing.T) {
	p := &testPlugin{
		preUninstall: func(*plugin.PreUninstallHookCtx) error { return nil },
		preUse: func(*plugin.PreUseHookCtx) (*plugin.PreUseHookResult, error) {
			return &plugin.PreUseHookResult{Version: "2.0.0"}, nil
		},
	}
	s := newTestSDK(t, p)
	stubShellAndRegistry(t)
	capturePterm(t)
	mustInstall(t, s, "1.0.0")
	mustInstall(t, s, "2.0.0")
	mustMakeGlobal(t, s, "2.0.0")

	err := s.Uninstall("2.0.0", "1.0.0")
	if err == nil || !strings.Contains(err.Error(), "resolves to the version that is being removed") {
		t.Fatalf("Uninstall(2.0.0) = %v, want the fallback to be refused", err)
	}
	if p.preUninstallCall != 0 {
		t.Errorf("PreUninstall ran %d times for a version that stays installed", p.preUninstallCall)
	}
	if !s.CheckRuntimeExist("2.0.0") || s.globalVersion() != "2.0.0" {
		t.Errorf("2.0.0 installed = %v, global version = %q, want it untouched", s.CheckRuntimeExist("2.0.0"), s.globalVersion())
	}
}

// PreUninstall is given the main runtime and the additions.
func TestUninstallGivesPreUninstallTheAdditions(t *testing.T) {
	var got map[string]*plugin.InstalledPackageItem
	p := &testPlugin{preUninstall: func(ctx *plugin.PreUninstallHookCtx) error {
		got = ctx.SdkInfo
		return nil
	}}
	s := newTestSDK(t, p)
	stubShellAndRegistry(t)
	capturePterm(t)
	mustInstall(t, s, "1.0.0")
	extra := filepath.Join(s.packagePath("1.0.0"), "extra-1.0.0")
	if err := os.MkdirAll(extra, 0755); err != nil {
		t.Fatal(err)
	}

	if err := s.Uninstall("1.0.0"); err != nil {
		t.Fatalf("Uninstall(1.0.0): %v", err)
	}
	if len(got) != 2 || got["test-sdk"] == nil || got["extra"] == nil || got["extra"].Path != extra {
		t.Errorf("PreUninstall got %v, want test-sdk and extra", got)
	}
}

// dependentSetups make x depend on the directory of 2.0.0 in different ways.
var dependentSetups = []struct {
	name  string
	setup func(t *testing.T, s *impl)
}{
	{"linked to the main runtime", func(t *testing.T, s *impl) {
		writeTestCommand(t, filepath.Join(s.mainRuntimePath("2.0.0"), "bin"), "tool")
		mustLink(t, s, "x", s.mainRuntimePath("2.0.0"))
	}},
	{"linked through a link that leads out again", func(t *testing.T, s *impl) {
		out := newExternalSDK(t, "tool")
		through := filepath.Join(s.packagePath("2.0.0"), "out")
		if err := env.CreateDirSymlink(out, through); err != nil {
			t.Fatal(err)
		}
		mustLink(t, s, "x", through)
	}},
	{"linked over another link", func(t *testing.T, s *impl) {
		writeTestCommand(t, filepath.Join(s.mainRuntimePath("2.0.0"), "bin"), "tool")
		first := filepath.Join(t.TempDir(), "first")
		if err := env.CreateDirSymlink(s.mainRuntimePath("2.0.0"), first); err != nil {
			t.Fatal(err)
		}
		mustLink(t, s, "x", first)
	}},
	{"linked below another link", func(t *testing.T, s *impl) {
		writeTestCommand(t, filepath.Join(s.mainRuntimePath("2.0.0"), "bin"), "tool")
		first := filepath.Join(t.TempDir(), "first")
		if err := env.CreateDirSymlink(s.InstallPath, first); err != nil {
			t.Fatal(err)
		}
		mustLink(t, s, "x", filepath.Join(first, "v-2.0.0", "test-sdk-2.0.0"))
	}},
	{"linked over a relative link", func(t *testing.T, s *impl) {
		if runtime.GOOS == "windows" {
			t.Skip("a junction cannot be relative")
		}
		writeTestCommand(t, filepath.Join(s.mainRuntimePath("2.0.0"), "bin"), "tool")
		first := filepath.Join(filepath.Dir(s.InstallPath), "first")
		if err := os.Symlink(filepath.Join("test-sdk", "v-2.0.0", "test-sdk-2.0.0"), first); err != nil {
			t.Fatal(err)
		}
		mustLink(t, s, "x", first)
	}},
	{"an addition is in there", func(t *testing.T, s *impl) {
		mustInstall(t, s, "x")
		if err := env.CreateDirSymlink(s.mainRuntimePath("2.0.0"), filepath.Join(s.packagePath("x"), "extra-x")); err != nil {
			t.Fatal(err)
		}
	}},
}

// A fallback that is reached through the directory of the removed version breaks
// when that directory is moved, so it cannot take over. It does not matter whether
// it ends in there or only passes through.
func TestUninstallRefusesAFallbackInTheRemovedDirectory(t *testing.T) {
	for _, tc := range dependentSetups {
		for _, next := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{true: ", another one is left"}[next], func(t *testing.T) {
				p := &testPlugin{preUninstall: func(*plugin.PreUninstallHookCtx) error { return nil }}
				s := newTestSDK(t, p)
				stubShellAndRegistry(t)
				capturePterm(t)
				mustInstall(t, s, "2.0.0")
				tc.setup(t, s)
				fallbacks := []Version{"x"}
				if next {
					mustInstall(t, s, "1.0.0")
					fallbacks = append(fallbacks, "1.0.0")
				}
				mustMakeGlobal(t, s, "2.0.0")

				err := s.Uninstall("2.0.0", fallbacks...)
				if next {
					if err != nil {
						t.Fatalf("Uninstall(2.0.0): %v", err)
					}
					if got := s.globalVersion(); got != "1.0.0" || s.CheckRuntimeExist("2.0.0") {
						t.Errorf("global version = %q, 2.0.0 installed = %v, want 1.0.0 and false", got, s.CheckRuntimeExist("2.0.0"))
					}
					return
				}
				if err == nil || !strings.Contains(err.Error(), "linked into the directory of") {
					t.Fatalf("Uninstall(2.0.0) = %v, want the fallback to be refused", err)
				}
				if p.preUninstallCall != 0 {
					t.Errorf("PreUninstall ran %d times for a version that stays installed", p.preUninstallCall)
				}
				if !s.CheckRuntimeExist("2.0.0") || s.globalVersion() != "2.0.0" {
					t.Errorf("2.0.0 installed = %v, global version = %q, want it untouched", s.CheckRuntimeExist("2.0.0"), s.globalVersion())
				}
			})
		}
	}
}

// The global version must not break because another version is uninstalled.
func TestUninstallRefusesAVersionTheGlobalOneDependsOn(t *testing.T) {
	for _, tc := range dependentSetups {
		t.Run(tc.name, func(t *testing.T) {
			p := &testPlugin{preUninstall: func(*plugin.PreUninstallHookCtx) error { return nil }}
			s := newTestSDK(t, p)
			stubShellAndRegistry(t)
			capturePterm(t)
			mustInstall(t, s, "2.0.0")
			mustInstall(t, s, "1.0.0")
			tc.setup(t, s)
			mustMakeGlobal(t, s, "x")
			links, err := s.linksOfScope(env.Global)
			if err != nil || len(links) == 0 {
				t.Fatalf("links = %v, %v", links, err)
			}

			err = s.Uninstall("2.0.0", "x", "1.0.0")
			if err == nil || !strings.Contains(err.Error(), "the global version test-sdk@x is linked into the directory of") ||
				!strings.Contains(err.Error(), "nothing was removed") {
				t.Fatalf("Uninstall(2.0.0) = %v, want it refused for the global version", err)
			}
			if p.preUninstallCall != 0 {
				t.Errorf("PreUninstall ran %d times for a version that stays installed", p.preUninstallCall)
			}
			if !s.CheckRuntimeExist("2.0.0") || !s.CheckRuntimeExist("x") || s.globalVersion() != "x" {
				t.Errorf("2.0.0 installed = %v, x installed = %v, global version = %q, want it untouched",
					s.CheckRuntimeExist("2.0.0"), s.CheckRuntimeExist("x"), s.globalVersion())
			}
			if got, err := s.linksOfScope(env.Global); err != nil || !reflect.DeepEqual(got, links) {
				t.Errorf("global links = %v, %v, want %v", got, err, links)
			}

			// It can go once the global version is another one.
			mustMakeGlobal(t, s, "1.0.0")
			if err := s.Uninstall("2.0.0", "1.0.0"); err != nil || s.CheckRuntimeExist("2.0.0") {
				t.Errorf("Uninstall(2.0.0) after the switch: %v, installed = %v", err, s.CheckRuntimeExist("2.0.0"))
			}
		})
	}
}

// A version that is not global does not stop the uninstall, but it does not break
// silently either.
func TestUninstallWarnsOfTheVersionsThatBreak(t *testing.T) {
	for _, tc := range dependentSetups {
		for _, global := range []Version{"", "1.0.0", "2.0.0"} {
			t.Run(tc.name+", global version "+string(global), func(t *testing.T) {
				s := newTestSDK(t, &testPlugin{})
				stubShellAndRegistry(t)
				out := capturePterm(t)
				mustInstall(t, s, "2.0.0")
				mustInstall(t, s, "1.0.0")
				mustLink(t, s, "y", newExternalSDK(t, "tool"))
				tc.setup(t, s)
				if global != "" {
					mustMakeGlobal(t, s, global)
				}

				if err := s.Uninstall("2.0.0", "1.0.0"); err != nil {
					t.Fatalf("Uninstall(2.0.0): %v", err)
				}
				if got := strings.Count(out.String(), "no longer works"); got != 1 || !strings.Contains(out.String(), "test-sdk@x no longer works") {
					t.Errorf("output = %q, want one warning, for x", out.String())
				}
			})
		}
	}
}

// Two versions linked to the same directory do not depend on each other.
func TestUninstallTakesAFallbackLinkedToTheSameDirectory(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	stubShellAndRegistry(t)
	capturePterm(t)
	external := newExternalSDK(t, "tool")
	mustLink(t, s, "a", external)
	mustLink(t, s, "b", external)
	mustMakeGlobal(t, s, "a")

	if err := s.Uninstall("a", "b"); err != nil {
		t.Fatalf("Uninstall(a): %v", err)
	}
	if got := s.globalVersion(); got != "b" {
		t.Errorf("global version = %q, want b", got)
	}
}

// Links into a version that is gone are what an uninstall left, unless the global
// config still names the version or its directory is there.
func TestUninstallLeavesTheLinksOfAVersionItDidNotRemove(t *testing.T) {
	for _, tc := range []struct {
		name   string
		damage func(t *testing.T, s *impl)
	}{
		{"the config names it", func(t *testing.T, s *impl) {
			if err := os.RemoveAll(s.packagePath("2.0.0")); err != nil {
				t.Fatal(err)
			}
		}},
		{"its directory is there", func(t *testing.T, s *impl) {
			if err := os.RemoveAll(s.mainRuntimePath("2.0.0")); err != nil {
				t.Fatal(err)
			}
			vfoxToml, err := s.envContext.LoadVfoxTomlByScope(env.Global)
			if err != nil {
				t.Fatal(err)
			}
			vfoxToml.RemoveTool(s.Name)
			if err := vfoxToml.Save(); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestSDK(t, &testPlugin{})
			stubShellAndRegistry(t)
			capturePterm(t)
			mustInstall(t, s, "2.0.0")
			mustMakeGlobal(t, s, "2.0.0")
			tc.damage(t, s)

			if s.HasLeftovers("2.0.0") {
				t.Error("HasLeftovers(2.0.0) = true, want false")
			}
			err := s.Uninstall("2.0.0")
			if err == nil || !strings.Contains(err.Error(), "is not installed") {
				t.Errorf("Uninstall(2.0.0) = %v, want it to be not installed", err)
			}
			if link := filepath.Join(s.envContext.GetLinkDirPathByScope(env.Global), "test-sdk"); !env.IsDirSymlink(link) {
				t.Error("the global link is gone")
			}
		})
	}
}
