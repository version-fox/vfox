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
