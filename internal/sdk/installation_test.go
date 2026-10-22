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
	"testing"

	"github.com/version-fox/vfox/internal/plugin"
)

func TestSdkNameIsTheOneOfThePlugin(t *testing.T) {
	if got := (&impl{Name: "Configured"}).sdkName(); got != "Configured" {
		t.Errorf("sdkName without a plugin = %q, want the configured name", got)
	}
	withoutMetadata := &impl{Name: "Configured", plugin: &plugin.Wrapper{}}
	if got := withoutMetadata.sdkName(); got != "Configured" {
		t.Errorf("sdkName without plugin metadata = %q, want the configured name", got)
	}
	s := newTestSDK(t, &testPlugin{})
	s.Name = "Configured"
	if got := s.sdkName(); got != "test-sdk" {
		t.Errorf("sdkName = %q, want the name the plugin lays its directories out under", got)
	}
}

// A receipt only informs, so one that cannot be read is no reason to fail.
func TestInstallationIgnoresAReceiptItCannotRead(t *testing.T) {
	s := newTestSDK(t, (&testPlugin{}).withCommand(t, "tool"))
	mustInstall(t, s, "1.0.0")
	if err := os.WriteFile(receiptPath(s.packagePath("1.0.0")), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

	inst, err := s.Installation("1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Receipt != nil || inst.Origin != OriginInstalled {
		t.Errorf("installation = %+v, want a version without a receipt that counts as installed", inst)
	}
	if target, linked := s.linkedTarget("1.0.0"); linked || target != "" {
		t.Errorf("linkedTarget = (%q, %v), want a plain installation", target, linked)
	}
}

// Everything a plugin has vfox download is a source, the additional packages
// too.
func TestInstallRecordsEveryPackageAsASource(t *testing.T) {
	p := (&testPlugin{preInstall: func(ctx *plugin.PreInstallHookCtx) (*plugin.PreInstallHookResult, error) {
		return &plugin.PreInstallHookResult{
			PreInstallPackageItem: &plugin.PreInstallPackageItem{Version: ctx.Version},
			Addition:              []*plugin.PreInstallPackageItem{{Name: "extra", Version: ctx.Version}},
		}, nil
	}}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0.0")

	inst, err := s.Installation("1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if inst.Receipt == nil || len(inst.Receipt.Sources) != 2 ||
		inst.Receipt.Sources[0].Name != "test-sdk" || inst.Receipt.Sources[1].Name != "extra" || inst.Receipt.Sources[1].Version != "1.0.0" {
		t.Errorf("receipt = %+v, want the main package and the additional one", inst.Receipt)
	}
	if len(inst.Additions) != 1 {
		t.Fatalf("additions = %+v, want the additional runtime", inst.Additions)
	}
	if _, err := os.Stat(filepath.Join(s.packagePath("1.0.0"), "add-extra-1.0.0")); err != nil {
		t.Errorf("the additional runtime has no directory: %v", err)
	}
}
