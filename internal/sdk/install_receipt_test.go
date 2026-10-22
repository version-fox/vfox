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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox/internal/plugin"
)

func TestInstallWritesTheReceipt(t *testing.T) {
	p := (&testPlugin{preInstall: func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		version := ctx.Version
		if version == "latest" {
			version = "2.0.0" // the plugin resolves what the user asked for
		}
		return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: version}}, nil
	}}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	before := time.Now().Add(-time.Second)

	mustInstall(t, s, "latest")

	inst, err := s.Installation("2.0.0")
	if err != nil {
		t.Fatal(err)
	}
	r := inst.Receipt
	if r == nil {
		t.Fatal("a finished installation must have a receipt")
	}
	if r.Schema != receiptSchema || r.SDK != "test-sdk" || r.Version != "2.0.0" || r.Requested != "latest" ||
		r.Origin != OriginInstalled || r.Target != "" || r.Vfox != "9.9.9" ||
		r.Plugin != (ReceiptPlugin{Name: "test-sdk", Version: "0.0.1"}) {
		t.Errorf("receipt = %+v", r)
	}
	if r.InstalledAt.Before(before) || r.InstalledAt.After(time.Now().Add(time.Second)) {
		t.Errorf("installedAt = %v", r.InstalledAt)
	}
	if len(r.Sources) != 1 || r.Sources[0].Name != "test-sdk" || r.Sources[0].Version != "2.0.0" {
		t.Errorf("sources = %+v", r.Sources)
	}
	if inst.Origin != OriginInstalled {
		t.Errorf("origin = %q", inst.Origin)
	}

	// It sits next to the payload, not in it: the SDK's own directory stays as shipped.
	if !exists(filepath.Join(s.packagePath("2.0.0"), receiptFilename)) {
		t.Error("no receipt in the version directory")
	}
	if exists(filepath.Join(s.mainRuntimePath("2.0.0"), receiptFilename)) {
		t.Error("the receipt must not be written into the SDK's own directory")
	}
	entries, _ := os.ReadDir(s.packagePath("2.0.0"))
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Errorf("left a temporary file behind: %s", entry.Name())
		}
	}
}

func TestFailedInstallLeavesNoReceipt(t *testing.T) {
	boom := errors.New("post-install failed")
	p := &testPlugin{postInstall: func(*plugin.PostInstallHookCtx) error { return boom }}
	s := newTestSDK(t, p)
	if err := s.Install("1.0.0"); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if exists(s.packagePath("1.0.0")) {
		t.Error("a failed installation must not leave its directory, and so no receipt")
	}
}

func TestInstallRecordsTheFacts(t *testing.T) {
	var seen *plugin.FingerprintHookCtx
	p := (&testPlugin{fingerprint: func(ctx *plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error) {
		seen = ctx
		return &plugin.FingerprintHookResult{Facts: map[string]string{"head": "abc123"}}, nil
	}}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0.0")

	inst, _ := s.Installation("1.0.0")
	if want := map[string]string{"head": "abc123"}; !reflect.DeepEqual(inst.Receipt.Facts, want) {
		t.Errorf("facts = %v, want %v", inst.Receipt.Facts, want)
	}
	main := filepath.Join(s.packagePath("1.0.0"), "test-sdk-1.0.0")
	if seen == nil || seen.Version != "1.0.0" || seen.Path != main || seen.SdkInfo["test-sdk"].Path != main {
		t.Errorf("hook context = %+v, want the final SDK path", seen)
	}
	if !exists(filepath.Join(main, "bin")) {
		t.Error("the hook must run after PostInstall, so that it sees the finished SDK")
	}
}

func TestInstallSurvivesAFailingFingerprintHook(t *testing.T) {
	for name, hook := range map[string]func(*plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error){
		"error": func(*plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error) {
			return nil, errors.New("boom")
		},
		"no result": func(*plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error) {
			return nil, plugin.ErrNoResultProvide
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := newTestSDK(t, (&testPlugin{fingerprint: hook}).withCommand(t, "tool"))
			mustInstall(t, s, "1.0.0")
			inst, err := s.Installation("1.0.0")
			if err != nil || inst.Receipt == nil || len(inst.Receipt.Facts) != 0 {
				t.Errorf("installation = %+v, err = %v: want a receipt without facts", inst, err)
			}
		})
	}
}

// A receipt only informs later checks, so a failure to write one must not fail
// the installation or take the SDK away.
func TestInstallSucceedsWhenTheReceiptCannotBeWritten(t *testing.T) {
	p := &testPlugin{postInstall: func(ctx *plugin.PostInstallHookCtx) error {
		// A directory where the receipt belongs makes the final rename fail.
		return os.MkdirAll(filepath.Join(ctx.RootPath, receiptFilename), 0755)
	}}
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0.0")

	if !s.CheckRuntimeExist("1.0.0") {
		t.Fatal("the installation must still be usable")
	}
	inst, err := s.Installation("1.0.0")
	if err != nil || inst.Receipt != nil {
		t.Errorf("installation = %+v, err = %v: want no receipt", inst, err)
	}
}

func TestReinstallReplacesThePayload(t *testing.T) {
	round := 0
	p := &testPlugin{
		postInstall: func(ctx *plugin.PostInstallHookCtx) error {
			round++
			return os.WriteFile(filepath.Join(ctx.SdkInfo["test-sdk"].Path, "round"+strconv.Itoa(round)), nil, 0644)
		},
		fingerprint: func(*plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error) {
			return &plugin.FingerprintHookResult{Facts: map[string]string{"round": strconv.Itoa(round)}}, nil
		},
	}
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0.0")
	main := s.mainRuntimePath("1.0.0")
	if !exists(filepath.Join(main, "round1")) {
		t.Fatal("setup: first installation")
	}

	if err := s.Reinstall("1.0.0"); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(main, "round1")) || !exists(filepath.Join(main, "round2")) {
		t.Error("the old payload must be replaced by the new one, not merged with it")
	}
	inst, err := s.Installation("1.0.0")
	if err != nil || inst.Receipt == nil || inst.Receipt.Facts["round"] != "2" {
		t.Errorf("the receipt must describe the new installation: %+v, %v", inst, err)
	}
	if _, err := os.Lstat(s.installMarkerPath("1.0.0")); !os.IsNotExist(err) {
		t.Error("the installation marker was left behind")
	}
}

func TestPlainInstallStillSkipsWhatIsInstalled(t *testing.T) {
	rounds := 0
	p := &testPlugin{postInstall: func(*plugin.PostInstallHookCtx) error { rounds++; return nil }}
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0.0")
	mustInstall(t, s, "1.0.0")
	if rounds != 1 || p.preInstallCalls != 1 {
		t.Errorf("rounds = %d, PreInstall calls = %d, want the second Install to be a no-op", rounds, p.preInstallCalls)
	}
}

func TestReinstallOfAMissingVersionJustInstallsIt(t *testing.T) {
	s := newTestSDK(t, (&testPlugin{}).withCommand(t, "tool"))
	if err := s.Reinstall("1.0.0"); err != nil {
		t.Fatal(err)
	}
	if !s.CheckRuntimeExist("1.0.0") {
		t.Error("not installed")
	}
}

func TestReinstallRefusesLinkedVersions(t *testing.T) {
	s, p, external := newLinkTestSDK(t)
	mustLink(t, s, "jdk", external)

	err := s.Reinstall("jdk")
	if err == nil || !strings.Contains(err.Error(), "linked") {
		t.Fatalf("err = %v, want a refusal that says the version is linked", err)
	}
	if p.preInstallCalls != 0 || !exists(binCommand(external, "tool")) || !s.CheckRuntimeExist("jdk") {
		t.Error("a refused reinstall must change nothing")
	}
}

func TestReinstallNeedsTheExactVersion(t *testing.T) {
	p := (&testPlugin{preInstall: func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		version := ctx.Version
		if version == "21" {
			version = "21.0.3"
		}
		return &plugin.PreInstallHookResult{PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: version}}, nil
	}}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	mustInstall(t, s, "21.0.3")

	err := s.Reinstall("21")
	if err == nil || !strings.Contains(err.Error(), "21.0.3") {
		t.Fatalf("err = %v, want it to name the version 21 resolves to", err)
	}
	if !s.CheckRuntimeExist("21.0.3") || !exists(binCommand(s.mainRuntimePath("21.0.3"), "tool")) {
		t.Error("the installed version was disturbed")
	}
}

// Documented behavior: like any failed installation, a failed reinstall leaves
// nothing, so the next attempt starts clean.
func TestFailedReinstallLeavesNothingBehind(t *testing.T) {
	fail := false
	p := &testPlugin{postInstall: func(*plugin.PostInstallHookCtx) error {
		if fail {
			return errors.New("second attempt failed")
		}
		return nil
	}}
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0.0")
	fail = true
	if err := s.Reinstall("1.0.0"); err == nil {
		t.Fatal("expected the failure")
	}
	if s.CheckRuntimeExist("1.0.0") || exists(s.packagePath("1.0.0")) || exists(s.installMarkerPath("1.0.0")) {
		t.Error("a failed reinstall must clean up completely")
	}
	fail = false
	if err := s.Reinstall("1.0.0"); err != nil || !s.CheckRuntimeExist("1.0.0") {
		t.Errorf("retry: %v", err)
	}
}
