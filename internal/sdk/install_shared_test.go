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
