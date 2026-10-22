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
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/plugin"
)

// facts is a Fingerprint hook that reports whatever the test last put in it.
type facts struct{ current map[string]string }

func (f *facts) hook(*plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error) {
	if f.current == nil {
		return nil, plugin.ErrNoResultProvide
	}
	return &plugin.FingerprintHookResult{Facts: f.current}, nil
}

func withFacts(f *facts) func(*testPlugin) {
	return func(p *testPlugin) { p.fingerprint = f.hook }
}

func TestDiagnoseHealthyInstallationWithFacts(t *testing.T) {
	s := installedSDK(t, withFacts(&facts{current: map[string]string{"head": "aaa"}}))
	for _, opts := range []DiagnoseOptions{{}, {Version: "1.0.0"}, {Version: "1.0.0", Fast: true}} {
		if findings := s.Diagnose(opts); len(findings) != 0 {
			t.Errorf("Diagnose(%+v) = %+v, want nothing", opts, findings)
		}
	}
}

func TestDiagnoseDrift(t *testing.T) {
	f := &facts{current: map[string]string{"head": "9584c6713b324636289d067944a46fd6b49df14b"}}
	s := installedSDK(t, withFacts(f))

	f.current = map[string]string{"head": "abaf9c523780a608bd46686fd5e53740a07077f8"}
	findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0", Fast: true})
	drift := findingByID(findings, FindingDrift)
	if len(findings) != 1 || drift == nil {
		t.Fatalf("findings = %+v, want one sdk.drift", findings)
	}
	if drift.Severity != SeverityWarn || drift.SDK != "test-sdk" || drift.Version != "1.0.0" {
		t.Errorf("finding = %+v", drift)
	}
	for _, want := range []string{"test-sdk@1.0.0", "head: 9584c6713b32… -> abaf9c523780…"} {
		if !strings.Contains(drift.Message, want) {
			t.Errorf("message %q lacks %q", drift.Message, want)
		}
	}
	if !reflect.DeepEqual(drift.Fix, []string{"vfox install --reinstall test-sdk@1.0.0"}) {
		t.Errorf("fix = %v", drift.Fix)
	}
	if drift.Detail["head"] != "9584c6713b324636289d067944a46fd6b49df14b -> abaf9c523780a608bd46686fd5e53740a07077f8" {
		t.Errorf("detail = %v", drift.Detail)
	}

	// A fact the plugin can no longer read is a change too.
	f.current = map[string]string{"other": "x"}
	drift = findingByID(s.Diagnose(DiagnoseOptions{Version: "1.0.0"}), FindingDrift)
	if drift == nil || !strings.Contains(drift.Message, "unreadable") {
		t.Errorf("finding = %+v, want the fact reported as unreadable", drift)
	}

	// New facts alongside the recorded ones are not drift: plugins may learn more.
	f.current = map[string]string{"head": "9584c6713b324636289d067944a46fd6b49df14b", "extra": "new"}
	if findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0"}); len(findings) != 0 {
		t.Errorf("extra facts reported as drift: %+v", findings)
	}
}

func TestDiagnoseDriftOfAFailingHook(t *testing.T) {
	f := &facts{current: map[string]string{"head": "aaa"}}
	p := (&testPlugin{fingerprint: func(*plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error) {
		if f.current == nil {
			return nil, errors.New("cannot read the checkout")
		}
		return &plugin.FingerprintHookResult{Facts: f.current}, nil
	}}).withCommand(t, "tool")
	s := newTestSDK(t, p)
	t.Setenv("PATH", t.TempDir())
	mustInstall(t, s, "1.0.0")

	f.current = nil
	if findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0", Fast: true}); len(findings) != 0 {
		t.Errorf("use must stay quiet about a hook that fails: %+v", findings)
	}
	findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0"})
	if f := findingByID(findings, FindingFingerprintFailed); f == nil || !strings.Contains(f.Message, "cannot read the checkout") {
		t.Errorf("doctor must say so: %+v", findings)
	}
}

func TestDiagnoseDriftOfALinkedVersionIsOnlyANotice(t *testing.T) {
	f := &facts{current: map[string]string{"release": "17.0.9"}}
	p := &testPlugin{fingerprint: f.hook}
	s := newTestSDK(t, p)
	t.Setenv("PATH", t.TempDir())
	mustLink(t, s, "jdk", newExternalSDK(t, "tool"))

	f.current = map[string]string{"release": "17.0.10"} // the owner updated their JDK
	drift := findingByID(s.Diagnose(DiagnoseOptions{Version: "jdk"}), FindingDrift)
	if drift == nil || drift.Severity != SeverityInfo || len(drift.Fix) != 0 {
		t.Errorf("finding = %+v, want an info without a fix: vfox cannot repair what it does not own", drift)
	}
}

func TestDiagnoseVersionWithoutReceipt(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	t.Setenv("PATH", t.TempDir())
	writeTestCommand(t, filepath.Join(s.mainRuntimePath("1.0.0"), "bin"), "tool") // as an old vfox left it

	if findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0", Fast: true}); len(findings) != 0 {
		t.Errorf("use must not nag about this: %+v", findings)
	}
	finding := findingByID(s.Diagnose(DiagnoseOptions{}), FindingNoReceipt)
	if finding == nil || finding.Severity != SeverityInfo ||
		!reflect.DeepEqual(finding.Fix, []string{"vfox doctor --adopt test-sdk@1.0.0"}) {
		t.Errorf("finding = %+v", finding)
	}
}

func TestCompareFacts(t *testing.T) {
	recorded := map[string]string{"head": "a", "tag": "v1", "gone": "x"}
	current := map[string]string{"head": "b", "tag": "v1", "new": "y"}
	got := compareFacts(recorded, current)
	want := []factDiff{{Key: "gone", Was: "x", Missing: true}, {Key: "head", Was: "a", Now: "b"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("diffs = %+v, want %+v", got, want)
	}
	if len(compareFacts(recorded, recorded)) != 0 {
		t.Error("identical facts are not a change")
	}
}

func TestLinkRecordsTheFacts(t *testing.T) {
	var seen *plugin.FingerprintHookCtx
	p := &testPlugin{fingerprint: func(ctx *plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error) {
		seen = ctx
		return &plugin.FingerprintHookResult{Facts: map[string]string{"release": "17.0.9"}}, nil
	}}
	s := newTestSDK(t, p)
	external := newExternalSDK(t, "tool")
	mustLink(t, s, "jdk", external)

	inst, err := s.Installation("jdk")
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"release": "17.0.9"}; !reflect.DeepEqual(inst.Receipt.Facts, want) {
		t.Errorf("facts = %v, want %v", inst.Receipt.Facts, want)
	}
	if seen == nil || seen.Version != "jdk" || !sameDir(t, seen.Path, external) {
		t.Errorf("hook context = %+v, want the linked SDK", seen)
	}
}

func TestLinkSurvivesAFailingFingerprintHook(t *testing.T) {
	p := &testPlugin{fingerprint: func(*plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error) {
		return nil, errors.New("boom")
	}}
	s := newTestSDK(t, p)
	mustLink(t, s, "jdk", newExternalSDK(t, "tool"))

	inst, err := s.Installation("jdk")
	if err != nil || inst.Receipt == nil || len(inst.Receipt.Facts) != 0 {
		t.Errorf("installation = %+v, err = %v: want a link with a receipt without facts", inst, err)
	}
}

func TestAdoptReportsAReceiptItCannotWrite(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	writeTestCommand(t, filepath.Join(s.mainRuntimePath("1.0.0"), "bin"), "tool") // as an old vfox left it
	// A directory where the receipt belongs.
	if err := os.MkdirAll(receiptPath(s.packagePath("1.0.0")), 0755); err != nil {
		t.Fatal(err)
	}

	adopted, err := s.Adopt("1.0.0")
	if adopted || err == nil || !strings.Contains(err.Error(), "write receipt") {
		t.Errorf("Adopt = (%v, %v), want the error of the write", adopted, err)
	}
}

func TestSdkInfoOfListsTheAdditionalRuntimes(t *testing.T) {
	pkg := &RuntimePackage{
		Runtime:   &Runtime{Name: "main", Version: "1.0.0", Path: "/sdk/main"},
		Additions: []*Runtime{{Name: "extra", Version: "2.0.0", Path: "/sdk/extra"}},
	}
	info := sdkInfoOf(pkg)
	if len(info) != 2 || info["main"].Path != "/sdk/main" || info["extra"].Version != "2.0.0" || info["extra"].Path != "/sdk/extra" {
		t.Errorf("sdkInfoOf = %+v, want the main runtime and the additional one, by name", info)
	}
}

func TestAdoptSurvivesAFailingFingerprintHook(t *testing.T) {
	p := &testPlugin{fingerprint: func(*plugin.FingerprintHookCtx) (*plugin.FingerprintHookResult, error) {
		return nil, errors.New("boom")
	}}
	s := newTestSDK(t, p)
	writeTestCommand(t, filepath.Join(s.mainRuntimePath("1.0.0"), "bin"), "tool") // as an old vfox left it

	adopted, err := s.Adopt("1.0.0")
	if err != nil || !adopted {
		t.Fatalf("Adopt = (%v, %v), want a receipt even though the plugin has nothing to say", adopted, err)
	}
	inst, _ := s.Installation("1.0.0")
	if inst.Receipt == nil || len(inst.Receipt.Facts) != 0 {
		t.Errorf("receipt = %+v, want one without facts", inst.Receipt)
	}
}
