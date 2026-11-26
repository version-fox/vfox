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

// waitUntil fails the test if cond is not true within a few seconds.
func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting until %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// In a VFOX_HOME shared by a group, the lock files another member created may not
// be writable. They are still opened for reading and locked, as flock and
// LockFileEx allow, so every version of the SDK can still be installed.
func TestLockFilesThatCannotBeWrittenStillLock(t *testing.T) {
	if runtime.GOOS != "windows" && os.Geteuid() == 0 {
		t.Skip("root can write any file")
	}
	p := (&testPlugin{}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0")
	locks := []string{
		filepath.Join(s.InstallPath, sdkInstallLockName),
		filepath.Join(s.InstallPath, ".v-1.0.lock"),
	}
	for _, lock := range locks {
		if err := os.Chmod(lock, 0444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(lock, 0644) })
	}
	if err := s.Install("2.0"); err != nil {
		t.Fatalf("Install(2.0) with a read-only SDK lock: %v", err)
	}
	if _, err := s.InstallResolved("1.0", true); err != nil {
		t.Fatalf("reinstall of 1.0 with read-only locks: %v", err)
	}
	// A read-only lock still keeps a second installation out.
	held, err := acquireInstallLock(locks[0])
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	if second, err := acquireInstallLock(locks[0]); err == nil {
		_ = second.Close()
		t.Fatal("a read-only lock was taken twice")
	}
}

// asSeparateProcess is called by a PostInstall that blocks to stand for another
// vfox. The test runs that vfox in a goroutine, which shares the environment, so
// this drops what its installation passes on to the commands PostInstall runs: a
// separate vfox would not inherit it.
func asSeparateProcess() {
	_ = os.Unsetenv(postInstallLocksEnv)
}

// A plain install mapped to a version whose reinstall starts during its PreInstall
// waits for the reinstall, and then finds the version installed.
func TestAMappedInstallWaitsForAReinstallThatStartedDuringPreInstall(t *testing.T) {
	shortLockWait(t)
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	mustInstall(t, s, "OTP-29.1.2")

	postInstall := p.postInstall
	inPostInstall, release := make(chan struct{}), make(chan struct{})
	p.postInstall = func(ctx *plugin.PostInstallHookCtx) error {
		if filepath.Base(ctx.RootPath) == "v-OTP-29.1.2" {
			asSeparateProcess()
			close(inPostInstall)
			<-release
		}
		return postInstall(ctx)
	}
	reinstalled := make(chan error, 1)
	calls := 0
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		if ctx.Version == "29.1.2" {
			calls++
			if calls == 1 {
				// Another vfox reinstalls the version this request maps to.
				go func() {
					_, err := s.InstallResolved("OTP-29.1.2", true)
					reinstalled <- err
				}()
				<-inPostInstall
				time.AfterFunc(200*time.Millisecond, func() { close(release) })
			}
		}
		return renamedRelease(ctx)
	}
	got, err := s.InstallResolved("29.1.2", false)
	if err != nil || got != "OTP-29.1.2" {
		t.Errorf("InstallResolved(29.1.2) = %q, %v; want OTP-29.1.2, nil", got, err)
	}
	if err := <-reinstalled; err != nil {
		t.Fatalf("reinstall of OTP-29.1.2: %v", err)
	}
	if exists(s.packagePath("29.1.2")) {
		t.Error("29.1.2 was installed beside OTP-29.1.2")
	}
}

// A plain install that a plugin maps to the name of a copy that `install --as` is
// still making is refused at once, with the same reason it gets once the copy is
// made.
func TestAnInstallMappedToACopyBeingMadeIsRefusedAsACopy(t *testing.T) {
	shortLockWait(t)
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	postInstall := p.postInstall
	inPostInstall, release := make(chan struct{}), make(chan struct{})
	p.postInstall = func(ctx *plugin.PostInstallHookCtx) error {
		if filepath.Base(ctx.RootPath) == "v-erl-db" {
			close(inPostInstall)
			<-release
		}
		return postInstall(ctx)
	}
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		version := ctx.Version
		if version == "31.0" {
			version = "erl-db"
		}
		return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: version}}, nil
	}
	copied := make(chan error, 1)
	go func() {
		_, err := s.InstallAs("30.0", "erl-db")
		copied <- err
	}()
	<-inPostInstall
	during := s.Install("31.0")
	close(release)
	if err := <-copied; err != nil {
		t.Fatalf("InstallAs(30.0, erl-db): %v", err)
	}
	after := s.Install("31.0")
	for when, err := range map[string]error{"while the copy is made": during, "after": after} {
		if err == nil || !strings.Contains(err.Error(), "is a copy made with `vfox install --as`") {
			t.Errorf("Install(31.0) %s: %v; want the copy refusal", when, err)
		}
	}
}

// Reinstalling a copy whose directory cannot be entered says so, and names the
// directory, instead of taking the copy for a plain version.
func TestReinstallOfACopyThatCannotBeEnteredNamesTheDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions do not keep the owner out on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can enter any directory")
	}
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	if _, err := s.InstallAs("29.1.2", "erl-db"); err != nil {
		t.Fatal(err)
	}
	dir := s.packagePath("erl-db")
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })
	called := false
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		called = true
		return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: ctx.Version}}, nil
	}
	_, err := s.InstallResolved("erl-db", true)
	if err == nil || !strings.Contains(err.Error(), dir) || strings.Contains(err.Error(), "was resolved to") {
		t.Fatalf("reinstall of an unreadable copy: %v; want an error that names %s", err, dir)
	}
	if called {
		t.Error("PreInstall was called for a copy whose receipt cannot be read")
	}
}

// A PostInstall that installs a version its plugin's PreInstall maps onto the
// version being installed fails at once instead of waiting for the installation
// that runs it, and the commands PostInstall starts are told which one that is.
// That holds when the installation root contains the path list separator.
func TestAnInstallInPostInstallMappedOntoItsOwnVersionFailsAtOnce(t *testing.T) {
	for _, tc := range []struct{ name, dir string }{
		{"plain root", "installs"},
		{"root with the path list separator", "in" + string(os.PathListSeparator) + "stalls"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wait := installLockWait
			installLockWait = 5 * time.Second
			t.Cleanup(func() { installLockWait = wait })
			p := (&testPlugin{}).withCommand(t, "erl")
			s := newTestSDK(t, p)
			s.InstallPath = filepath.Join(t.TempDir(), tc.dir, "test-sdk")
			p.preInstall = renamedRelease
			postInstall := p.postInstall
			var nested error
			var took time.Duration
			var inherited []string
			p.postInstall = func(ctx *plugin.PostInstallHookCtx) error {
				if filepath.Base(ctx.RootPath) == "v-OTP-29.1.2" {
					inherited = postInstallLocks()
					start := time.Now()
					nested = s.Install("29.1.2")
					took = time.Since(start)
				}
				return postInstall(ctx)
			}
			if err := s.Install("OTP-29.1.2"); err != nil {
				t.Fatalf("Install(OTP-29.1.2): %v", err)
			}
			if nested == nil || !strings.Contains(nested.Error(), "runs in the PostInstall of") {
				t.Errorf("Install(29.1.2) in the PostInstall of OTP-29.1.2: %v; want the PostInstall error", nested)
			}
			if took >= installLockWait {
				t.Errorf("Install(29.1.2) in the PostInstall of OTP-29.1.2 took %s; want it to fail at once", took)
			}
			if lock := s.versionLockPath("OTP-29.1.2"); !slices.Equal(inherited, []string{lock}) {
				t.Errorf("%s in PostInstall lists %q; want %q", postInstallLocksEnv, inherited, lock)
			}
			if value, set := os.LookupEnv(postInstallLocksEnv); set {
				t.Errorf("%s = %q after the installation; want it unset again", postInstallLocksEnv, value)
			}
			if exists(s.packagePath("29.1.2")) {
				t.Error("29.1.2 was installed beside OTP-29.1.2")
			}
		})
	}
}

// The locks of nested PostInstalls add up, and a value vfox did not set lists none.
func TestPostInstallLocksAddUp(t *testing.T) {
	t.Setenv(postInstallLocksEnv, "not a list")
	if locks := postInstallLocks(); locks != nil {
		t.Errorf("postInstallLocks() = %q for a value vfox did not set; want none", locks)
	}
	outer := filepath.Join(t.TempDir(), "a"+string(os.PathListSeparator)+"b", ".v-1.lock")
	inner := filepath.Join(t.TempDir(), ".v-2.lock")
	t.Setenv(postInstallLocksEnv, "")
	_ = withPostInstallLock(outer, func() error {
		return withPostInstallLock(inner, func() error {
			if locks := postInstallLocks(); !slices.Equal(locks, []string{outer, inner}) {
				t.Errorf("postInstallLocks() = %q; want %q", locks, []string{outer, inner})
			}
			if !inPostInstallOf(outer) || !inPostInstallOf(inner) {
				t.Error("inPostInstallOf does not find both locks")
			}
			return nil
		})
	})
	if value := os.Getenv(postInstallLocksEnv); value != "" {
		t.Errorf("%s = %q after both PostInstalls; want it as it was", postInstallLocksEnv, value)
	}
}

// A plain install mapped onto a version that is uninstalled after the list given
// to PreInstall was checked again asks PreInstall again, instead of installing the
// answer, which names the version it was mapped onto and nothing else.
func TestAnInstallMappedOntoAVersionUninstalledMeanwhileAsksAgain(t *testing.T) {
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	mustInstall(t, s, "OTP-29.1.2")
	p.preInstall = renamedRelease
	t.Cleanup(func() { afterInstallListRecheck = nil })
	afterInstallListRecheck = func() {
		afterInstallListRecheck = nil
		if err := s.Uninstall("OTP-29.1.2"); err != nil {
			t.Errorf("Uninstall(OTP-29.1.2): %v", err)
		}
	}
	p.preInstallCalls = 0
	if err := s.Install("29.1.2"); err != nil {
		t.Fatalf("Install(29.1.2): %v", err)
	}
	if p.preInstallCalls != 2 {
		t.Errorf("PreInstall was called %d times; want it asked again after the uninstall", p.preInstallCalls)
	}
	if !s.CheckRuntimeExist("29.1.2") {
		t.Error("29.1.2 is not installed")
	}
	if exists(s.packagePath("OTP-29.1.2")) {
		t.Error("the uninstalled OTP-29.1.2 was installed again from the answer given for it")
	}
}

// Lists of the same size are told apart by their versions.
func TestInstallListsOfTheSameSizeDiffer(t *testing.T) {
	item := &plugin.InstalledPackageItem{}
	if sameVersions(map[string]*plugin.InstalledPackageItem{"1.0": item}, map[string]*plugin.InstalledPackageItem{"2.0": item}) {
		t.Error("sameVersions takes 1.0 and 2.0 for the same list")
	}
	if !sameVersions(map[string]*plugin.InstalledPackageItem{"1.0": item}, map[string]*plugin.InstalledPackageItem{"1.0": {}}) {
		t.Error("sameVersions tells two lists of 1.0 apart")
	}
	if sameInstalling(map[Version]bool{"1.0": true}, map[Version]bool{"2.0": true}) {
		t.Error("sameInstalling takes 1.0 and 2.0 for the same set")
	}
	if !sameInstalling(map[Version]bool{"1.0": true}, map[Version]bool{"1.0": true}) {
		t.Error("sameInstalling tells two sets of 1.0 apart")
	}
}

// A version whose lock cannot be opened is not taken for one being installed, and
// waiting for it reports why the lock could not be opened.
func TestALockThatCannotBeOpened(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	// The installation root is under a file, so its lock files cannot be created.
	s.InstallPath = filepath.Join(file, "test-sdk")
	if s.installationRunning("1.0") {
		t.Error("installationRunning(1.0) = true for a lock that cannot be opened")
	}
	err := s.waitForInstallation("1", "1.0")
	if err == nil || !strings.Contains(err.Error(), "cannot install 1: create installation root") {
		t.Errorf("waitForInstallation = %v; want the error that kept the lock from opening", err)
	}
}
