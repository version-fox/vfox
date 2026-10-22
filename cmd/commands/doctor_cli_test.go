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

package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type doctorJSON struct {
	OK       bool
	Findings []struct {
		ID       string
		Severity string
		SDK      string
		Version  string
		Message  string
		Hints    []string
		Fix      []string
		Ignored  bool
	}
}

func (d doctorJSON) find(id string) (severity string, found bool) {
	for _, f := range d.Findings {
		if f.ID == id {
			return f.Severity, true
		}
	}
	return "", false
}

func parseDoctor(t *testing.T, r cliResult) doctorJSON {
	t.Helper()
	var report doctorJSON
	if err := json.Unmarshal([]byte(r.stdout), &report); err != nil {
		t.Fatalf("doctor --json printed %q: %v", r.stdout, err)
	}
	return report
}

func TestDoctorOnAnEmptyHome(t *testing.T) {
	f := newCLIFixture(t, false)
	out := f.run(t, "doctor").mustSucceed(t, "doctor").output()
	mustContain(t, "doctor", out, "No problems found.")

	report := parseDoctor(t, f.run(t, "doctor", "--json").mustSucceed(t, "doctor --json"))
	if !report.OK || report.Findings == nil || len(report.Findings) != 0 {
		t.Errorf("report = %+v, want ok with an empty list (not null)", report)
	}
}

func TestDoctorNeedsAnActivatedShell(t *testing.T) {
	f := newCLIFixture(t, false)
	f.env["__VFOX_SHELL"] = "-"
	r := f.run(t, "doctor").mustFail(t, "doctor in a shell that did not activate vfox")
	mustContain(t, "doctor", r.output(), "error", "path.not-activated", "vfox activate")

	report := parseDoctor(t, f.run(t, "doctor", "--json").mustFail(t, "doctor --json"))
	if severity, ok := report.find("path.not-activated"); !ok || severity != "error" || report.OK {
		t.Errorf("report = %+v", report)
	}
}

func TestDoctorReportsABrokenLink(t *testing.T) {
	f := newCLIFixture(t, false)
	external := f.external(t, "alphatool")
	f.run(t, "link", "alpha@sys", external).mustSucceed(t, "link")
	must(t, os.RemoveAll(external))

	r := f.run(t, "doctor").mustFail(t, "doctor with a broken link")
	mustContain(t, "doctor", r.stdout,
		"Found 1 error, 0 warnings and 0 notices.",
		"error  sdk.link-dangling  alpha@sys",
		external,
		"fix: vfox uninstall alpha@sys")

	report := parseDoctor(t, f.run(t, "doctor", "--json").mustFail(t, "doctor --json"))
	if severity, ok := report.find("sdk.link-dangling"); !ok || severity != "error" || report.OK {
		t.Errorf("report = %+v", report)
	}

	// Narrowed to what is broken, or to something else.
	f.run(t, "doctor", "alpha@sys").mustFail(t, "doctor alpha@sys")
	f.run(t, "doctor", "alpha").mustFail(t, "doctor alpha")
	f.run(t, "doctor", "plain").mustSucceed(t, "doctor plain")
	f.run(t, "doctor", "nosuch").mustFail(t, "doctor of an SDK without a plugin")

	// The fix it names works, and leaves the SDK's directory (gone as it is) alone.
	f.run(t, "uninstall", "alpha@sys").mustSucceed(t, "uninstall of a broken link")
	f.run(t, "doctor").mustSucceed(t, "doctor after the fix")
}

func TestDoctorAdoptsOldInstallationsAndNoticesWhenTheyChange(t *testing.T) {
	f := newCLIFixture(t, false)
	main := f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	must(t, os.WriteFile(filepath.Join(main, "VERSION"), []byte("1.0.0"), 0644))

	// Installed before vfox kept receipts: nothing to compare against.
	r := f.run(t, "doctor").mustSucceed(t, "doctor")
	mustContain(t, "doctor", r.stdout, "Found 0 errors, 0 warnings and 1 notice.", "info   sdk.no-receipt  alpha@1.0.0", "fix: vfox doctor --adopt alpha@1.0.0")

	out := f.run(t, "doctor", "--adopt").mustSucceed(t, "doctor --adopt").output()
	mustContain(t, "adopt", out, "Adopted alpha@1.0.0", "1 receipt written.")
	mustContain(t, "doctor", f.run(t, "doctor").mustSucceed(t, "doctor").output(), "No problems found.")
	mustContain(t, "second adopt", f.run(t, "doctor", "--adopt").mustSucceed(t, "doctor --adopt").output(), "0 receipts written.")
	mustNotContain(t, "second adopt", f.run(t, "doctor", "--adopt").output(), "Adopted")

	// The SDK modifies itself.
	must(t, os.WriteFile(filepath.Join(main, "VERSION"), []byte("1.0.1"), 0644))
	r = f.run(t, "doctor").mustSucceed(t, "a warning does not fail doctor")
	mustContain(t, "doctor", r.stdout,
		"Found 0 errors, 1 warning and 0 notices.",
		"warn   sdk.drift  alpha@1.0.0",
		"has changed since vfox installed it",
		"fix: vfox install --reinstall alpha@1.0.0")
	report := parseDoctor(t, f.run(t, "doctor", "--json", "alpha").mustSucceed(t, "doctor --json"))
	if severity, ok := report.find("sdk.drift"); !ok || severity != "warn" || !report.OK {
		t.Errorf("report = %+v", report)
	}
}

func TestDoctorAdoptNarrowsToTheArgument(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	f.installOnDisk(t, "alpha", "2.0.0", "alphatool")
	f.installOnDisk(t, "plain", "1.0.0", "plaintool")

	out := f.run(t, "doctor", "--adopt", "alpha@2.0.0").mustSucceed(t, "doctor --adopt alpha@2.0.0").output()
	mustContain(t, "adopt", out, "Adopted alpha@2.0.0", "1 receipt written.")
	mustNotContain(t, "adopt", out, "alpha@1.0.0", "plain@1.0.0")

	out = f.run(t, "doctor", "--adopt", "alpha").mustSucceed(t, "doctor --adopt alpha").output()
	mustContain(t, "adopt", out, "Adopted alpha@1.0.0", "1 receipt written.")

	f.run(t, "doctor", "--adopt", "alpha@9.9.9").mustFail(t, "adopting a version that is not installed")
	mustContain(t, "adopt", f.run(t, "doctor", "--adopt", "nosuch").mustFail(t, "adopting for an SDK without a plugin").output(), "not supported")
}

func TestDoctorReportsInstalledVersionsWithoutPlugin(t *testing.T) {
	f := newCLIFixture(t, false)
	must(t, os.MkdirAll(filepath.Join(f.shared, "cache", "orphan", "v-1.0.0", "orphan-1.0.0"), 0755))

	r := f.run(t, "doctor").mustSucceed(t, "doctor")
	mustContain(t, "doctor", r.stdout, "warn   sdk.plugin-missing  orphan", "fix: vfox add orphan")
}

func TestDoctorReportsAPluginThatNeedsANewerVfox(t *testing.T) {
	f := newCLIFixture(t, false)
	f.writePlugin(t, "future", strings.Replace(plainPlugin, `name="plain"`, `name="future"`, 1)+"PLUGIN.minRuntimeVersion = \"99.0.0\"\n")

	r := f.run(t, "doctor").mustSucceed(t, "doctor")
	mustContain(t, "doctor", r.stdout, "plugin.incompatible  future", "needs vfox 99.0.0", "fix: vfox upgrade")
}

func TestDoctorHonorsTheIgnoreList(t *testing.T) {
	f := newCLIFixture(t, false)
	external := f.external(t, "alphatool")
	f.run(t, "link", "alpha@sys", external).mustSucceed(t, "link")
	must(t, os.RemoveAll(external))
	f.run(t, "doctor").mustFail(t, "doctor with a broken link")

	// An ignored error does not fail the check either.
	f.writeUserConfig(t, "diagnostics:\n  ignore:\n    - sdk.link-dangling\n")
	out := f.run(t, "doctor").mustSucceed(t, "doctor with the broken link ignored").stdout
	mustContain(t, "doctor", out, "No problems found.", "1 finding ignored by diagnostics.ignore.", "(ignored)")

	report := parseDoctor(t, f.run(t, "doctor", "--json").mustSucceed(t, "doctor --json"))
	if len(report.Findings) != 1 || !report.Findings[0].Ignored || !report.OK {
		t.Errorf("report = %+v, want the one finding kept and marked", report)
	}
}

func TestDoctorHonorsTheIgnoreListForNotices(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool") // no receipt: an info finding
	f.writeUserConfig(t, "diagnostics:\n  ignore:\n    - sdk.no-receipt:alpha\n")

	out := f.run(t, "doctor").mustSucceed(t, "doctor").stdout
	mustContain(t, "doctor", out, "No problems found.", "1 finding ignored by diagnostics.ignore.", "(ignored)")

	report := parseDoctor(t, f.run(t, "doctor", "--json").mustSucceed(t, "doctor --json"))
	if len(report.Findings) != 1 || !report.Findings[0].Ignored {
		t.Errorf("findings = %+v, want the one finding kept and marked", report.Findings)
	}
}

// shadowFixture is the situation behind pnpm.md: a command of the SDK exists a
// second time in a directory that comes before vfox's in PATH.
type shadowFixture struct {
	*cliFixture
	shadowDir string
}

func newShadowFixture(t *testing.T, dirName string) shadowFixture {
	t.Helper()
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	shadowDir := filepath.Join(f.root, dirName, "bin")
	must(t, os.MkdirAll(shadowDir, 0755))
	must(t, os.WriteFile(filepath.Join(shadowDir, "alphatool"), []byte("#!/bin/sh\n"), 0755))
	f.env["PATH"] = strings.Join([]string{shadowDir, filepath.Join(f.userHome, "sdks", "alpha", "bin"), f.system}, ":")
	return shadowFixture{cliFixture: f, shadowDir: shadowDir}
}

func TestUseSaysWhenACommandIsShadowed(t *testing.T) {
	f := newShadowFixture(t, "pnpm")

	r := f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use must not fail because of a warning")
	out := r.output()
	mustContain(t, "use", out,
		"Now using", "alpha@1.0.0",
		"`alphatool` runs "+filepath.Join(f.shadowDir, "alphatool")+" instead of alpha@1.0.0",
		"comes before vfox in PATH",
		"pnpm put its own alphatool first. See https://example.com/pnpm")
	// The same diagnosis is available on demand, with the same words.
	doctor := f.run(t, "doctor", "alpha").mustSucceed(t, "doctor alpha").stdout
	mustContain(t, "doctor", doctor, "warn   path.shadowed  alpha@1.0.0", "pnpm put its own alphatool first.")
}

func TestUseStaysQuietWhenNothingIsShadowed(t *testing.T) {
	f := newCLIFixture(t, false)
	f.installOnDisk(t, "alpha", "1.0.0", "alphatool")
	f.env["PATH"] = strings.Join([]string{filepath.Join(f.userHome, "sdks", "alpha", "bin"), f.system}, ":")
	out := f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use").output()
	mustContain(t, "use", out, "Now using")
	mustNotContain(t, "use", out, "comes before vfox", "WARNING")
}

func TestUseWarningCanBeSwitchedOffOrIgnored(t *testing.T) {
	t.Run("diagnostics.onUse", func(t *testing.T) {
		f := newShadowFixture(t, "pnpm")
		f.writeUserConfig(t, "diagnostics:\n  onUse: false\n")
		mustNotContain(t, "use", f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use").output(), "comes before vfox")
		// Switching it off for `use` does not switch off `doctor`.
		mustContain(t, "doctor", f.run(t, "doctor", "alpha").mustSucceed(t, "doctor").stdout, "path.shadowed")
	})
	t.Run("diagnostics.ignore", func(t *testing.T) {
		f := newShadowFixture(t, "pnpm")
		f.writeUserConfig(t, "diagnostics:\n  ignore:\n    - path.shadowed:alpha\n")
		mustNotContain(t, "use", f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use").output(), "comes before vfox")
	})
	t.Run("an entry for another SDK", func(t *testing.T) {
		f := newShadowFixture(t, "pnpm")
		f.writeUserConfig(t, "diagnostics:\n  ignore:\n    - path.shadowed:plain\n")
		mustContain(t, "use", f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use").output(), "comes before vfox")
	})
}

func TestUseStaysQuietAboutAnActiveVirtualenv(t *testing.T) {
	f := newShadowFixture(t, "venv")
	f.env["VIRTUAL_ENV"] = filepath.Join(f.root, "venv")
	out := f.run(t, "use", "-g", "alpha@1.0.0").mustSucceed(t, "use").output()
	mustContain(t, "use", out, "Now using")
	mustNotContain(t, "use", out, "comes before vfox")
	// It is still on record for whoever asks.
	mustContain(t, "doctor", f.run(t, "doctor", "alpha").mustSucceed(t, "doctor").stdout, "info   path.shadowed")
}
