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

package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/env"
	"github.com/version-fox/vfox/internal/pathmeta"
	"github.com/version-fox/vfox/internal/plugin"
	"github.com/version-fox/vfox/internal/sdk"
)

// fixturePlugin is a minimal SDK plugin: <path>/bin goes on PATH.
const fixturePlugin = `PLUGIN = {name="%s", version="0.0.1", description="test", author="test", minRuntimeVersion="%s"}
function PLUGIN:Available(ctx) return {{version="1.0.0"}} end
function PLUGIN:PreInstall(ctx) return {version=ctx.version} end
function PLUGIN:EnvKeys(ctx) return {{key="PATH", value=ctx.path .. "/bin"}} end
`

// managerFixture is a Manager over a temporary vfox home, with no plugin added.
type managerFixture struct {
	*Manager
	meta *pathmeta.PathMeta
}

func newManagerFixture(t *testing.T) *managerFixture {
	t.Helper()
	root := t.TempDir()
	work := filepath.Join(root, "project")
	shared := filepath.Join(root, "shared")
	for _, dir := range []string{work, shared} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	meta, err := pathmeta.NewPathMeta(filepath.Join(root, "home"), shared, work, 4242)
	if err != nil {
		t.Fatal(err)
	}
	m := &Manager{
		RuntimeEnvContext: &env.RuntimeEnvContext{
			UserConfig:        &config.Config{},
			CurrentWorkingDir: work,
			PathMeta:          meta,
			RuntimeVersion:    RuntimeVersion,
		},
		openSdks: make(map[string]sdk.Sdk),
	}
	t.Cleanup(m.Close)
	// Whatever shell the tests run in must not leak into the checks. An
	// activated shell is the default, unless a test says otherwise.
	t.Setenv(env.HookFlag, "bash")
	t.Setenv(env.OrigPathFlag, "")
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("CONDA_PREFIX", "")
	return &managerFixture{Manager: m, meta: meta}
}

func (f *managerFixture) writePlugin(t *testing.T, name, minRuntimeVersion string) {
	t.Helper()
	dir := filepath.Join(f.meta.Shared.Plugins, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	source := fmt.Sprintf(fixturePlugin, name, minRuntimeVersion)
	if err := os.WriteFile(filepath.Join(dir, "main.lua"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
}

// addSDK adds the plugin of an SDK and returns the SDK.
func (f *managerFixture) addSDK(t *testing.T, name string) sdk.Sdk {
	t.Helper()
	f.writePlugin(t, name, "0.0.0")
	s, err := f.LookupSdk(name)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// install lays out an installed version the way vfox did before it kept
// receipts, with one command in its bin directory. It returns the main runtime.
func (f *managerFixture) install(t *testing.T, name, version string) string {
	t.Helper()
	main := filepath.Join(f.meta.Shared.Installs, name, "v-"+version, name+"-"+version)
	bin := filepath.Join(main, "bin")
	if err := os.MkdirAll(bin, 0755); err != nil {
		t.Fatal(err)
	}
	command := name + "tool"
	if runtime.GOOS == "windows" {
		command += ".cmd"
	}
	if err := os.WriteFile(filepath.Join(bin, command), []byte("echo"), 0755); err != nil {
		t.Fatal(err)
	}
	return main
}

func findFinding(findings []sdk.Finding, id, sdkName string) *sdk.Finding {
	for i := range findings {
		if findings[i].ID == id && findings[i].SDK == sdkName {
			return &findings[i]
		}
	}
	return nil
}

func TestDiagnoseWholeHome(t *testing.T) {
	f := newManagerFixture(t)
	f.addSDK(t, "alpha")
	f.install(t, "alpha", "1.0.0")
	// Installed versions whose plugin is not added.
	if err := os.MkdirAll(filepath.Join(f.meta.Shared.Installs, "orphan", "v-1.0.0", "orphan-1.0.0"), 0755); err != nil {
		t.Fatal(err)
	}
	// A plugin written for a newer vfox.
	f.writePlugin(t, "future", "99.0.0")
	t.Setenv(env.HookFlag, "") // this shell was not activated

	findings, err := f.Diagnose("", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if findFinding(findings, sdk.FindingPathNotActivated, "") == nil {
		t.Errorf("a shell without the hook must be reported: %+v", findings)
	}
	if got := findFinding(findings, sdk.FindingPluginMissing, "orphan"); got == nil || len(got.Fix) != 1 || got.Fix[0] != "vfox add orphan" {
		t.Errorf("finding = %+v, want the orphaned installation with the way to add its plugin", got)
	}
	if got := findFinding(findings, sdk.FindingPluginIncompatible, "future"); got == nil || len(got.Fix) != 1 || got.Fix[0] != "vfox upgrade" {
		t.Errorf("finding = %+v, want the plugin that needs a newer vfox", got)
	}
	for _, finding := range findings {
		if finding.SDK == "alpha" && finding.Severity != sdk.SeverityInfo {
			t.Errorf("a healthy installation has a finding: %+v", finding)
		}
	}
	// What is not about one SDK comes first; the rest is ordered by SDK.
	if findings[0].ID != sdk.FindingPathNotActivated {
		t.Errorf("findings are not sorted: %+v", findings)
	}
}

func TestDiagnoseOneSDK(t *testing.T) {
	f := newManagerFixture(t)
	f.addSDK(t, "alpha")
	f.install(t, "alpha", "1.0.0")
	if err := os.MkdirAll(filepath.Join(f.meta.Shared.Installs, "orphan", "v-1.0.0", "orphan-1.0.0"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(env.HookFlag, "")

	findings, err := f.Diagnose("alpha", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if findFinding(findings, sdk.FindingPathNotActivated, "") == nil {
		t.Error("whether the shell is activated matters for every SDK")
	}
	if findFinding(findings, sdk.FindingPluginMissing, "orphan") != nil {
		t.Error("the scan for orphaned installations belongs to the check of the whole home")
	}

	if _, err := f.Diagnose("nosuch", "", nil); err == nil || !strings.Contains(err.Error(), "not supported") {
		t.Errorf("err = %v, want it to say that the SDK is not supported", err)
	}
}

// Only the version in effect can be shadowed in a way that matters, so the PATH
// is looked at for that version only.
func TestDiagnoseLooksAtThePATHOfTheVersionInEffect(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX PATH lists")
	}
	f := newManagerFixture(t)
	f.addSDK(t, "alpha")
	f.install(t, "alpha", "1.0.0")
	f.install(t, "alpha", "2.0.0")
	shadow := filepath.Join(t.TempDir(), "pnpm", "bin")
	if err := os.MkdirAll(shadow, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(shadow, "alphatool"), []byte("echo"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", strings.Join([]string{shadow, filepath.Join(f.meta.Working.GlobalSdkDir, "alpha", "bin"), t.TempDir()}, string(os.PathListSeparator)))

	shadowedVersions := func(version sdk.Version, inEffect VersionInEffect) []string {
		findings, err := f.Diagnose("alpha", version, inEffect)
		if err != nil {
			t.Fatal(err)
		}
		var versions []string
		for _, finding := range findings {
			if finding.ID == sdk.FindingPathShadowed {
				versions = append(versions, finding.Version)
			}
		}
		return versions
	}

	if got := shadowedVersions("", nil); len(got) != 0 {
		t.Errorf("no version is in effect, so there is no PATH to look at: %v", got)
	}
	if err := os.WriteFile(filepath.Join(f.meta.Working.Directory, ".vfox.toml"), []byte("[tools]\nalpha = \"1.0.0\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := shadowedVersions("", nil); len(got) != 1 || got[0] != "1.0.0" {
		t.Errorf("shadowed = %v, want only the version that is in effect", got)
	}
	if got := shadowedVersions("2.0.0", nil); len(got) != 1 || got[0] != "2.0.0" {
		t.Errorf("shadowed = %v, want the version that was asked for", got)
	}

	// The caller can say better than Sdk.Current which version the shell uses,
	// for instance when a version that is not installed is asked for first.
	versionIs := func(version sdk.Version) VersionInEffect { return func(sdk.Sdk) sdk.Version { return version } }
	if got := shadowedVersions("", versionIs("2.0.0")); len(got) != 1 || got[0] != "2.0.0" {
		t.Errorf("shadowed = %v, want the version the caller names", got)
	}
	if got := shadowedVersions("", versionIs("")); len(got) != 0 {
		t.Errorf("shadowed = %v, want no PATH check when the caller names no version", got)
	}
	if got := shadowedVersions("1.0.0", versionIs("2.0.0")); len(got) != 1 || got[0] != "1.0.0" {
		t.Errorf("shadowed = %v, want the version that was asked for to win", got)
	}
}

func TestDiagnoseMarksWhatTheUserIgnores(t *testing.T) {
	f := newManagerFixture(t)
	f.addSDK(t, "alpha")
	t.Setenv(env.HookFlag, "")
	f.RuntimeEnvContext.UserConfig = &config.Config{Diagnostics: &config.Diagnostics{OnUse: true, Ignore: []string{sdk.FindingPathNotActivated}}}

	findings, err := f.Diagnose("", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	got := findFinding(findings, sdk.FindingPathNotActivated, "")
	if got == nil || !got.Ignored {
		t.Errorf("finding = %+v, want it kept and marked as ignored", got)
	}
}

func TestManagerIgnores(t *testing.T) {
	f := newManagerFixture(t)
	f.RuntimeEnvContext.UserConfig = &config.Config{Diagnostics: &config.Diagnostics{Ignore: []string{"sdk.drift:alpha@1.0.0", "path.shadowed"}}}
	for _, tt := range []struct {
		id, sdkName, version string
		want                 bool
	}{
		{"sdk.drift", "alpha", "1.0.0", true},
		{"sdk.drift", "alpha", "2.0.0", false},
		{"path.shadowed", "beta", "3", true},
		{"sdk.no-receipt", "alpha", "1.0.0", false},
	} {
		if got := f.ignores(tt.id, tt.sdkName, tt.version); got != tt.want {
			t.Errorf("ignores(%q, %q, %q) = %v, want %v", tt.id, tt.sdkName, tt.version, got, tt.want)
		}
	}

	f.RuntimeEnvContext.UserConfig = nil
	if f.ignores("sdk.drift", "alpha", "1.0.0") {
		t.Error("without a config nothing is ignored")
	}
	f.RuntimeEnvContext = nil
	if f.ignores("sdk.drift", "alpha", "1.0.0") {
		t.Error("without a context nothing is ignored")
	}
}

func TestActivationFindings(t *testing.T) {
	f := newManagerFixture(t)
	if got := f.activationFindings(); got != nil {
		t.Errorf("an activated shell has nothing to report: %+v", got)
	}

	t.Setenv(env.HookFlag, "")
	got := f.activationFindings()
	if len(got) != 1 || got[0].ID != sdk.FindingPathNotActivated || len(got[0].Hints) == 0 {
		t.Fatalf("findings = %+v, want path.not-activated with hints on how to activate", got)
	}
	// Without a hook only the global scope works on Windows, which is less than broken.
	want := sdk.SeverityError
	if runtime.GOOS == "windows" {
		want = sdk.SeverityWarn
	}
	if got[0].Severity != want {
		t.Errorf("severity = %s, want %s", got[0].Severity, want)
	}
}

func TestPluginFindings(t *testing.T) {
	metadata := func(minRuntimeVersion string) *sdk.Metadata {
		return &sdk.Metadata{Name: "Alpha", PluginMetadata: &plugin.Metadata{Name: "Alpha", MinRuntimeVersion: minRuntimeVersion}}
	}
	for name, meta := range map[string]*sdk.Metadata{
		"no metadata":        nil,
		"no plugin metadata": {Name: "alpha"},
		"no requirement":     metadata(""),
		"an older vfox":      metadata("0.0.1"),
		"this very vfox":     metadata(RuntimeVersion),
	} {
		if got := pluginFindings(meta); got != nil {
			t.Errorf("%s: findings = %+v, want none", name, got)
		}
	}

	got := pluginFindings(metadata("99.0.0"))
	if len(got) != 1 || got[0].ID != sdk.FindingPluginIncompatible || got[0].SDK != "alpha" || got[0].Severity != sdk.SeverityWarn ||
		!strings.Contains(got[0].Message, "99.0.0") || !strings.Contains(got[0].Message, RuntimeVersion) {
		t.Errorf("findings = %+v, want a warning that names both versions", got)
	}
}

func TestOrphanedInstallFindings(t *testing.T) {
	f := newManagerFixture(t)
	installs := f.meta.Shared.Installs
	mkdir := func(parts ...string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(append([]string{installs}, parts...)...), 0755); err != nil {
			t.Fatal(err)
		}
	}
	f.writePlugin(t, "added", "0.0.0")
	mkdir("added", "v-1.0.0")      // has its plugin
	mkdir("empty")                 // nothing was installed
	mkdir("junk", "not-a-version") // not an installation
	mkdir("orphan", "v-1.0.0")     // installed, no plugin
	mkdir("orphan", "v-2.0.0")     // a second version is no second finding
	mkdir("Mixed", "v-3.0.0")      // the name keeps its case in the message only
	if err := os.WriteFile(filepath.Join(installs, ".lock"), nil, 0644); err != nil {
		t.Fatal(err) // a file among the SDKs
	}

	findings := f.orphanedInstallFindings()
	var names []string
	for _, finding := range findings {
		names = append(names, finding.SDK)
		if finding.ID != sdk.FindingPluginMissing || finding.Severity != sdk.SeverityWarn {
			t.Errorf("finding = %+v", finding)
		}
	}
	if got := strings.Join(names, ","); got != "mixed,orphan" {
		t.Errorf("SDKs = %q, want mixed,orphan", got)
	}

	// Nothing is installed at all.
	if err := os.RemoveAll(installs); err != nil {
		t.Fatal(err)
	}
	if got := f.orphanedInstallFindings(); got != nil {
		t.Errorf("findings = %+v, want none without an installation directory", got)
	}
}
