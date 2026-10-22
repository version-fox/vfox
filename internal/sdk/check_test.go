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
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/plugin"
)

// installedSDK returns an SDK with version 1.0.0 installed, whose "tool" command
// is on PATH, and a PATH of system directories only. tweaks change the plugin
// before the installation.
func installedSDK(t *testing.T, tweaks ...func(*testPlugin)) *impl {
	t.Helper()
	p := (&testPlugin{}).withCommand(t, "tool")
	for _, tweak := range tweaks {
		tweak(p)
	}
	s := newTestSDK(t, p)
	t.Setenv("PATH", filepath.Join(t.TempDir(), "system-bin"))
	mustInstall(t, s, "1.0.0")
	return s
}

func TestDiagnoseHealthyInstallation(t *testing.T) {
	s := installedSDK(t)
	for _, opts := range []DiagnoseOptions{{}, {Version: "1.0.0"}, {Version: "1.0.0", Fast: true}} {
		if findings := s.Diagnose(opts); len(findings) != 0 {
			t.Errorf("Diagnose(%+v) = %+v, want nothing", opts, findings)
		}
	}
}

func TestDiagnoseBrokenLink(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	t.Setenv("PATH", t.TempDir())
	external := newExternalSDK(t, "tool")
	mustLink(t, s, "jdk", external)
	if err := os.RemoveAll(external); err != nil {
		t.Fatal(err)
	}

	findings := s.Diagnose(DiagnoseOptions{})
	broken := findingByID(findings, FindingLinkDangling)
	if len(findings) != 1 || broken == nil || broken.Severity != SeverityError ||
		!strings.Contains(broken.Message, external) || !reflect.DeepEqual(broken.Fix, []string{"vfox uninstall test-sdk@jdk"}) {
		t.Errorf("findings = %+v", findings)
	}
}

func TestDiagnoseEmptyPayload(t *testing.T) {
	t.Run("no directory", func(t *testing.T) {
		s := newTestSDK(t, &testPlugin{})
		t.Setenv("PATH", t.TempDir())
		mustInstall(t, s, "1.0.0")
		f := findingByID(s.Diagnose(DiagnoseOptions{Version: "1.0.0"}), FindingPayloadEmpty)
		if f == nil || f.Severity != SeverityError {
			t.Errorf("finding = %+v", f)
		}
	})
	t.Run("directory without commands", func(t *testing.T) {
		s := newTestSDK(t, &testPlugin{postInstall: func(ctx *plugin.PostInstallHookCtx) error {
			return os.MkdirAll(filepath.Join(ctx.SdkInfo["test-sdk"].Path, "bin"), 0755)
		}})
		t.Setenv("PATH", t.TempDir())
		mustInstall(t, s, "1.0.0")
		f := findingByID(s.Diagnose(DiagnoseOptions{Version: "1.0.0"}), FindingPayloadEmpty)
		if f == nil || f.Severity != SeverityWarn {
			t.Errorf("finding = %+v, want a warning: the directory is there, the commands are not", f)
		}
	})
	t.Run("a plugin that only sets variables has nothing to check", func(t *testing.T) {
		s := newTestSDK(t, &testPlugin{envKeys: func(ctx *plugin.EnvKeysHookCtx) ([]*plugin.EnvKeysHookResultItem, error) {
			return []*plugin.EnvKeysHookResultItem{{Key: "TEST_SDK_HOME", Value: ctx.Path}}, nil
		}})
		t.Setenv("PATH", t.TempDir())
		mustInstall(t, s, "1.0.0")
		if findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0"}); len(findings) != 0 {
			t.Errorf("findings = %+v", findings)
		}
	})
	t.Run("the plugin fails", func(t *testing.T) {
		boom := false
		s := newTestSDK(t, &testPlugin{envKeys: func(ctx *plugin.EnvKeysHookCtx) ([]*plugin.EnvKeysHookResultItem, error) {
			if boom {
				return nil, errors.New("EnvKeys exploded")
			}
			return []*plugin.EnvKeysHookResultItem{{Key: "X", Value: "1"}}, nil
		}})
		t.Setenv("PATH", t.TempDir())
		mustInstall(t, s, "1.0.0")
		boom = true
		f := findingByID(s.Diagnose(DiagnoseOptions{Version: "1.0.0"}), FindingEnvKeysFailed)
		if f == nil || f.Severity != SeverityError || !strings.Contains(f.Message, "EnvKeys exploded") {
			t.Errorf("finding = %+v", f)
		}
	})
}

func TestDiagnoseInterruptedAndIncompleteInstallations(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	t.Setenv("PATH", t.TempDir())
	// 2.0.0: killed during installation. 3.0.0: only the marker is left.
	// 4.0.0: a directory with nothing in it.
	if err := os.MkdirAll(filepath.Join(s.mainRuntimePath("2.0.0"), "partial"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, version := range []Version{"2.0.0", "3.0.0"} {
		if err := os.WriteFile(s.installMarkerPath(version), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(s.packagePath("4.0.0"), 0755); err != nil {
		t.Fatal(err)
	}

	if findings := s.Diagnose(DiagnoseOptions{Version: "2.0.0", Fast: true}); len(findings) != 0 {
		t.Errorf("use cannot select an unfinished installation, so it says nothing: %+v", findings)
	}
	findings := s.Diagnose(DiagnoseOptions{})
	got := map[string]string{}
	for _, f := range findings {
		got[f.Version] = f.ID
	}
	want := map[string]string{"2.0.0": FindingInstallInterrupted, "3.0.0": FindingInstallInterrupted, "4.0.0": FindingPayloadEmpty}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("findings by version = %v, want %v", got, want)
	}
	if f := findingByID(findings, FindingInstallInterrupted); f == nil || f.Severity != SeverityWarn || len(f.Fix) != 1 {
		t.Errorf("finding = %+v", f)
	}
}

func TestVersionsOnDisk(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	for _, dir := range []string{"v-1.0.0", "v-2.0.0", "not-a-version", "v-"} {
		if err := os.MkdirAll(filepath.Join(s.InstallPath, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{".v-3.0.0.installing", ".v-1.0.0.lock", "v-file", ".available.cache"} {
		if err := os.WriteFile(filepath.Join(s.InstallPath, file), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	got := s.versionsOnDisk()
	want := []Version{"", "1.0.0", "2.0.0", "3.0.0"} // "" is the odd 'v-' directory; it sorts first
	if !reflect.DeepEqual(got, want) {
		t.Errorf("versionsOnDisk = %q, want %q", got, want)
	}
	if (&impl{InstallPath: filepath.Join(t.TempDir(), "missing")}).versionsOnDisk() != nil {
		t.Error("an SDK that was never installed has no versions")
	}
}

// pathFixture arranges a PATH in which a command of the installed SDK is
// provided again by a directory in front of vfox's.
type pathFixture struct {
	root, shadow, vfox, system string
}

func newPathFixture(t *testing.T, shadowDir string) pathFixture {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX PATH lists")
	}
	f := pathFixture{root: t.TempDir()}
	f.shadow = filepath.Join(f.root, shadowDir, "bin")
	f.vfox = filepath.Join(f.root, ".vfox", "sdks", "test-sdk", "bin")
	f.system = filepath.Join(f.root, "system", "bin")
	writeTestCommand(t, f.shadow, "tool")
	writeTestCommand(t, f.system, "other")
	t.Setenv("PATH", strings.Join([]string{f.shadow, f.vfox, f.system}, string(os.PathListSeparator)))
	return f
}

func TestDiagnoseShadowedCommand(t *testing.T) {
	s := installedSDK(t)
	f := newPathFixture(t, "pnpm")

	findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0", Fast: true})
	shadowed := findingByID(findings, FindingPathShadowed)
	if len(findings) != 1 || shadowed == nil {
		t.Fatalf("findings = %+v, want one path.shadowed", findings)
	}
	if shadowed.Severity != SeverityWarn {
		t.Errorf("severity = %s", shadowed.Severity)
	}
	if want := "`tool` runs " + filepath.Join(f.shadow, "tool") + " instead of test-sdk@1.0.0"; !strings.Contains(shadowed.Message, want) {
		t.Errorf("message = %q, want it to contain %q", shadowed.Message, want)
	}
	if shadowed.Detail["directory"] != f.shadow || shadowed.Detail["commands"] != "tool" || shadowed.Detail["source"] != "unknown" {
		t.Errorf("detail = %v", shadowed.Detail)
	}
	if len(shadowed.Hints) != 0 {
		t.Errorf("hints = %v", shadowed.Hints)
	}

	t.Run("the plugin explains what it knows", func(t *testing.T) {
		s.plugin.Metadata.ShadowHints = []plugin.ShadowHint{
			{Match: "PNPM", Message: "pnpm installed its own tool.", URL: "https://example.com/pnpm.md"},
			{Match: "volta", Message: "never shown"},
		}
		t.Cleanup(func() { s.plugin.Metadata.ShadowHints = nil })
		hints := findingByID(s.Diagnose(DiagnoseOptions{Version: "1.0.0", Fast: true}), FindingPathShadowed).Hints
		if len(hints) != 1 || hints[0] != "pnpm installed its own tool. See https://example.com/pnpm.md" {
			t.Errorf("hints = %q", hints)
		}
	})

	t.Run("the caller can leave the PATH out", func(t *testing.T) {
		if findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0", NoPath: true}); len(findings) != 0 {
			t.Errorf("NoPath: %+v", findings)
		}
		if findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0", PathFor: "2.0.0"}); len(findings) != 0 {
			t.Errorf("only the version in effect matters: %+v", findings)
		}
		if findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0", PathFor: "1.0.0"}); len(findings) != 1 {
			t.Errorf("PathFor the version itself: %+v", findings)
		}
	})
}

func TestDiagnoseShadowingByAnActiveVirtualenvIsOnlyANotice(t *testing.T) {
	s := installedSDK(t)
	f := newPathFixture(t, filepath.Join("venv"))
	t.Setenv("VIRTUAL_ENV", filepath.Join(f.root, "venv"))

	shadowed := findingByID(s.Diagnose(DiagnoseOptions{Version: "1.0.0", Fast: true}), FindingPathShadowed)
	if shadowed == nil || shadowed.Severity != SeverityInfo || shadowed.Detail["source"] != "activated-env" {
		t.Fatalf("finding = %+v", shadowed)
	}
	var out bytes.Buffer
	printFindings(&out, []Finding{*shadowed}, SeverityWarn)
	if out.Len() != 0 {
		t.Errorf("a virtualenv that wins on purpose must not be reported to the user: %q", out.String())
	}
}

func TestDiagnoseSeveralShadowedCommandsAreOneFinding(t *testing.T) {
	p := &testPlugin{postInstall: func(ctx *plugin.PostInstallHookCtx) error {
		for _, name := range []string{"node", "npm", "npx", "corepack", "extra"} {
			if err := writeTestCommandAt(filepath.Join(ctx.SdkInfo["test-sdk"].Path, "bin"), name); err != nil {
				return err
			}
		}
		return nil
	}}
	s := newTestSDK(t, p)
	mustInstall(t, s, "1.0.0")
	f := newPathFixture(t, "pnpm")
	for _, name := range []string{"node", "npm", "npx", "corepack", "extra"} {
		writeTestCommand(t, f.shadow, name)
	}

	findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0", Fast: true})
	if len(findings) != 1 {
		t.Fatalf("findings = %+v, want one for the whole directory", findings)
	}
	if !strings.Contains(findings[0].Message, "`corepack`, `extra`, `node` and 2 more run files from "+f.shadow) {
		t.Errorf("message = %q", findings[0].Message)
	}
	if findings[0].Detail["commands"] != "corepack, extra, node, npm, npx" {
		t.Errorf("commands = %q", findings[0].Detail["commands"])
	}
}

func TestDiagnoseIgnore(t *testing.T) {
	s := installedSDK(t)
	newPathFixture(t, "pnpm")
	s.envContext.UserConfig = &config.Config{Diagnostics: &config.Diagnostics{OnUse: true, Ignore: []string{"path.shadowed:test-sdk@1.0.0"}}}

	findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0", Fast: true})
	if len(findings) != 1 || !findings[0].Ignored {
		t.Fatalf("findings = %+v, want the finding kept but marked ignored", findings)
	}
	var out bytes.Buffer
	printFindings(&out, findings, SeverityWarn)
	if out.Len() != 0 {
		t.Errorf("an ignored finding was printed: %q", out.String())
	}

	s.envContext.UserConfig = &config.Config{Diagnostics: &config.Diagnostics{OnUse: true, Ignore: []string{"path.shadowed:test-sdk@2.0.0"}}}
	if findings := s.Diagnose(DiagnoseOptions{Version: "1.0.0", Fast: true}); len(findings) != 1 || findings[0].Ignored {
		t.Errorf("an entry for another version must not silence this one: %+v", findings)
	}
}

func TestDiagnosticsOnUseSwitch(t *testing.T) {
	s := installedSDK(t)
	if !s.diagnosticsOnUse() {
		t.Error("the checks run after `use` by default")
	}
	s.envContext.UserConfig = &config.Config{Diagnostics: &config.Diagnostics{OnUse: false}}
	if s.diagnosticsOnUse() {
		t.Error("diagnostics.onUse: false was ignored")
	}
	s.envContext = nil
	if !s.diagnosticsOnUse() || s.ignores("sdk.drift", "x", "1") {
		t.Error("an SDK without an environment context must not crash and uses the defaults")
	}
}

func TestPrintFindings(t *testing.T) {
	findings := []Finding{
		{ID: "a", Severity: SeverityError, Message: "broken thing", Fix: []string{"vfox fix it"}},
		{ID: "b", Severity: SeverityWarn, Message: "odd thing", Hints: []string{"because of this", "and that"}},
		{ID: "c", Severity: SeverityInfo, Message: "fyi"},
		{ID: "d", Severity: SeverityWarn, Message: "silenced", Ignored: true},
	}
	var out bytes.Buffer
	printFindings(&out, findings, SeverityWarn)
	got := out.String()
	for _, want := range []string{"broken thing", "Fix: ", "vfox fix it", "odd thing", "  because of this", "  and that"} {
		if !strings.Contains(got, want) {
			t.Errorf("output lacks %q:\n%s", want, got)
		}
	}
	for _, unwanted := range []string{"fyi", "silenced"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("output must not contain %q:\n%s", unwanted, got)
		}
	}

	out.Reset()
	printFindings(&out, findings, SeverityInfo)
	if !strings.Contains(out.String(), "fyi") || strings.Contains(out.String(), "silenced") {
		t.Errorf("min = info:\n%s", out.String())
	}
}

func TestSortFindings(t *testing.T) {
	findings := []Finding{
		{ID: "z", Severity: SeverityInfo, SDK: "b", Version: "1"},
		{ID: "y", Severity: SeverityWarn, SDK: "a", Version: "2"},
		{ID: "x", Severity: SeverityWarn, SDK: "a", Version: "1"},
		{ID: "w", Severity: SeverityError, SDK: "a", Version: "1"},
	}
	SortFindings(findings)
	if got := findingIDs(findings); !reflect.DeepEqual(got, []string{"w", "x", "y", "z"}) {
		t.Errorf("order = %v", got)
	}
}

func TestSeverityAtLeast(t *testing.T) {
	if !SeverityError.AtLeast(SeverityWarn) || !SeverityWarn.AtLeast(SeverityWarn) || SeverityInfo.AtLeast(SeverityWarn) {
		t.Error("severity ordering is wrong")
	}
	if Severity("").AtLeast(SeverityWarn) {
		t.Error("an unknown severity ranks lowest")
	}
}

func TestCommandList(t *testing.T) {
	tests := map[string][]string{
		"`node`":                          {"node"},
		"`node` and `npm`":                {"node", "npm"},
		"`node`, `npm` and `npx`":         {"node", "npm", "npx"},
		"`node`, `npm`, `npx` and 1 more": {"node", "npm", "npx", "corepack"},
		"`node`, `npm`, `npx` and 3 more": {"node", "npm", "npx", "a", "b", "c"},
	}
	for want, names := range tests {
		if got := commandList(names); got != want {
			t.Errorf("commandList(%v) = %q, want %q", names, got, want)
		}
	}
}

func TestDiagnoseAVersionThatIsNotThere(t *testing.T) {
	s := installedSDK(t)
	if findings := s.Diagnose(DiagnoseOptions{Version: "9.9.9"}); len(findings) != 0 {
		t.Errorf("findings = %+v, want nothing about a version with no trace on disk", findings)
	}
}

// A link can be healthy and still lead to an SDK that lost its commands.
func TestDiagnoseALinkToAnSDKWithoutCommands(t *testing.T) {
	s := newTestSDK(t, &testPlugin{})
	t.Setenv("PATH", t.TempDir())
	external := newExternalSDK(t, "tool")
	mustLink(t, s, "jdk", external)
	if err := os.Remove(binCommand(external, "tool")); err != nil {
		t.Fatal(err)
	}

	f := findingByID(s.Diagnose(DiagnoseOptions{Version: "jdk"}), FindingPayloadEmpty)
	if f == nil || f.Severity != SeverityWarn || len(f.Hints) != 1 || !strings.Contains(f.Hints[0], external) {
		t.Fatalf("finding = %+v, want a warning that points at the linked directory", f)
	}
	if len(f.Fix) != 0 {
		t.Errorf("fix = %v, want none: vfox cannot repair what it does not own", f.Fix)
	}
}

func TestShadowChecksNeedWhatTheyLookAt(t *testing.T) {
	var s impl // no environment and no plugin
	if got := s.shadowFindings("1.0.0", []string{"/sdk/bin"}); got != nil {
		t.Errorf("shadowFindings = %+v, want none without an environment", got)
	}
	if got := s.shadowHints("/home/me/.local/share/pnpm/bin/node"); got != nil {
		t.Errorf("shadowHints = %v, want none without a plugin", got)
	}
}
