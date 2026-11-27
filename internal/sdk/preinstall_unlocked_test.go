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
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/plugin"
)

// PreInstall runs no lock, so a PreInstall can install another version of its
// SDK, as a plugin does through os.execute, and both end up installed.
func TestPreInstallCanInstallAnotherVersionOfItsSDK(t *testing.T) {
	shortLockWait(t)
	p := (&testPlugin{}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	var nested error
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		if ctx.Version == "1.0" {
			nested = s.Install("2.0")
		}
		return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: ctx.Version}}, nil
	}
	mustInstall(t, s, "1.0")
	if nested != nil {
		t.Errorf("Install(2.0) from PreInstall of 1.0: %v", nested)
	}
	for _, version := range []Version{"1.0", "2.0"} {
		if !s.CheckRuntimeExist(version) {
			t.Errorf("%s is not installed", version)
		}
	}
}

// A slow PreInstall of one version does not hold up the install of another.
func TestASlowPreInstallDoesNotHoldUpOtherVersions(t *testing.T) {
	shortLockWait(t)
	p := (&testPlugin{}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	resolving, finish := make(chan struct{}), make(chan struct{})
	var once sync.Once
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		// 2.0 is installed meanwhile, so PreInstall of 1.0 is asked a second time.
		if ctx.Version == "1.0" {
			once.Do(func() {
				close(resolving)
				<-finish
			})
		}
		return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: ctx.Version}}, nil
	}
	first := make(chan error, 1)
	go func() { first <- s.Install("1.0") }()
	<-resolving

	err := s.Install("2.0")
	close(finish)
	if err != nil {
		t.Errorf("Install(2.0) while PreInstall of 1.0 runs: %v", err)
	}
	if err := <-first; err != nil {
		t.Fatalf("Install(1.0): %v", err)
	}
}

// The list PreInstall was given is checked again under the SDK lock. If another
// installation changed it meanwhile, PreInstall is asked again with the new list,
// so a version installed under its old name is not installed a second time.
func TestPreInstallIsAskedAgainWhenTheListChangedMeanwhile(t *testing.T) {
	shortLockWait(t)
	p := (&testPlugin{}).withCommand(t, "erl")
	s := newTestSDK(t, p)
	calls := 0
	p.preInstall = func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		if ctx.Version == "29.1.2" {
			calls++
			if calls == 1 {
				// Another vfox installs the release under its old name while this
				// PreInstall runs.
				if err := s.Install("OTP-29.1.2"); err != nil {
					t.Errorf("Install(OTP-29.1.2): %v", err)
				}
			}
		}
		return renamedRelease(ctx)
	}
	got, err := s.InstallResolved("29.1.2", false)
	if err != nil || got != "OTP-29.1.2" {
		t.Errorf("InstallResolved(29.1.2) = %q, %v; want OTP-29.1.2, nil", got, err)
	}
	if calls != 2 {
		t.Errorf("PreInstall of 29.1.2 was called %d times, want 2", calls)
	}
	if exists(s.packagePath("29.1.2")) {
		t.Error("29.1.2 was installed beside OTP-29.1.2")
	}
}

// The helper is a separate process so abrupt termination of `install --as`
// skips all install defers.
func TestInstallAsInterruptedHelper(t *testing.T) {
	root := os.Getenv("VFOX_INSTALL_AS_INTERRUPTION_TEST")
	if root == "" {
		return
	}
	s := newInstallTestSDK(root, func(ctx *plugin.PostInstallHookCtx) error {
		if err := os.WriteFile(filepath.Join(root, "ready"), nil, 0600); err != nil {
			return err
		}
		time.Sleep(time.Hour)
		return nil
	})
	if _, err := s.InstallAs("1.0", "erl-db"); err != nil {
		t.Fatal(err)
	}
}

// A copy killed during PostInstall never finished, so it does not make its name
// refuse a plain install, which installs the name as on main.
func TestAnInterruptedCopyDoesNotRefuseItsName(t *testing.T) {
	root := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestInstallAsInterruptedHelper$")
	cmd.Env = append(os.Environ(), "VFOX_INSTALL_AS_INTERRUPTION_TEST="+root)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(root, "ready")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			t.Fatal("child did not reach PostInstall")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()

	s := newInstallTestSDK(root, nil)
	if !s.hasPendingInstall("erl-db") {
		t.Fatal("the interrupted copy left no installation marker")
	}
	if s.isCopy("erl-db") {
		t.Error("an interrupted copy counts as a copy")
	}
	if err := s.Install("erl-db"); err != nil {
		if strings.Contains(err.Error(), "is a copy") {
			t.Fatalf("Install(erl-db) after an interrupted copy: %v", err)
		}
		t.Fatalf("Install(erl-db): %v", err)
	}
	if !s.CheckRuntimeExist("erl-db") || s.isCopy("erl-db") {
		t.Error("erl-db is not installed as a plain version")
	}
}
