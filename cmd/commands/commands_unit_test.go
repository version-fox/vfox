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
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/version-fox/vfox/internal/sdk"
)

func TestParseLinkSpec(t *testing.T) {
	valid := map[string][2]string{
		"java@17-system":     {"java", "17-system"},
		"Java@21.0.3+9":      {"java", "21.0.3+9"}, // SDK names are lower-cased, versions are used as written
		"nodejs@v24.21.0":    {"nodejs", "v24.21.0"},
		"golang@brew":        {"golang", "brew"},
		"java@17.0.9-tem.v2": {"java", "17.0.9-tem.v2"},
	}
	for spec, want := range valid {
		name, version, err := parseLinkSpec(spec)
		if err != nil || name != want[0] || string(version) != want[1] {
			t.Errorf("parseLinkSpec(%q) = (%q, %q, %v), want (%q, %q)", spec, name, version, err, want[0], want[1])
		}
	}
	for _, spec := range []string{"", "java", "java@", "@17", "java@17@x", "@"} {
		if _, _, err := parseLinkSpec(spec); err == nil {
			t.Errorf("parseLinkSpec(%q) must fail", spec)
		}
	}
}

func TestSplitDoctorSpec(t *testing.T) {
	tests := []struct {
		spec, name, version string
	}{
		{"", "", ""},
		{"nodejs", "nodejs", ""},
		{"NodeJS", "nodejs", ""},
		{"nodejs@24.21.0", "nodejs", "24.21.0"},
		{"nodejs@v24.21.0", "nodejs", "v24.21.0"}, // the spelling `vfox current` prints; whether the v goes is for ResolveVersionName
		{"java@vendor", "java", "vendor"},
		{"nodejs@", "nodejs", ""},
	}
	for _, tt := range tests {
		name, version := splitDoctorSpec(tt.spec)
		if name != tt.name || string(version) != tt.version {
			t.Errorf("splitDoctorSpec(%q) = (%q, %q), want (%q, %q)", tt.spec, name, version, tt.name, tt.version)
		}
	}
}

// The commands that take <sdk>@<version> hand the version over as typed: a
// leading "v" may belong to the name of a linked version or an alias.
func TestSdkArgumentsKeepTheVersionAsTyped(t *testing.T) {
	for _, tt := range []struct{ arg, name, version string }{
		{"nodejs", "nodejs", ""},
		{"nodejs@24.21.0", "nodejs", "24.21.0"},
		{"nodejs@v24.21.0", "nodejs", "v24.21.0"},
		{"java@vendor", "java", "vendor"},
	} {
		name, version := parseSdkArg(tt.arg)
		if name != tt.name || string(version) != tt.version {
			t.Errorf("parseSdkArg(%q) = (%q, %q), want (%q, %q)", tt.arg, name, version, tt.name, tt.version)
		}
		spec, err := parseExecSDKSpec(tt.arg)
		if err != nil || spec.Name != tt.name || string(spec.Version) != tt.version {
			t.Errorf("parseExecSDKSpec(%q) = (%+v, %v), want (%q, %q)", tt.arg, spec, err, tt.name, tt.version)
		}
	}
}

func TestFormatVersionLabel(t *testing.T) {
	tests := []struct {
		name    string
		version sdk.Version
		origin  sdk.Origin
		target  string
		aliases []string
		want    string
	}{
		{"installed", "17.0.9", sdk.OriginInstalled, "", nil, "17.0.9"},
		{"linked", "17-sys", sdk.OriginLinked, "/usr/lib/jvm/x", nil, "17-sys (linked → /usr/lib/jvm/x)"},
		{"linked, target unknown", "17-sys", sdk.OriginLinked, "", nil, "17-sys (linked)"},
		{"aliased", "17.0.9", sdk.OriginInstalled, "", []string{"j17", "lts"}, "17.0.9 [alias: j17, lts]"},
		{"both", "17-sys", sdk.OriginLinked, "/jdk", []string{"sys"}, "17-sys (linked → /jdk) [alias: sys]"},
	}
	for _, tt := range tests {
		if got := formatVersionLabel(tt.version, tt.origin, tt.target, tt.aliases); got != tt.want {
			t.Errorf("%s: label = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestAliasesByVersion(t *testing.T) {
	got := aliasesByVersion(map[string]string{"z": "1", "a": "1", "m": "2"})
	want := map[string][]string{"1": {"a", "z"}, "2": {"m"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("aliasesByVersion = %v, want %v", got, want)
	}
	if got := aliasesByVersion(nil); len(got) != 0 {
		t.Errorf("no aliases: %v", got)
	}
}

// fakeSdk is the part of sdk.Sdk that detectCandidates looks at.
type fakeSdk struct {
	sdk.Sdk
	installed map[sdk.Version]string // version -> link target ("" when installed by vfox)
}

func (f fakeSdk) CheckRuntimeExist(v sdk.Version) bool { _, ok := f.installed[v]; return ok }

func (f fakeSdk) InstalledList() []sdk.Version {
	var versions []sdk.Version
	for v := range f.installed {
		versions = append(versions, v)
	}
	return versions
}

func (f fakeSdk) Installation(v sdk.Version) (*sdk.Installation, error) {
	target, ok := f.installed[v]
	if !ok {
		return nil, sdk.ErrRuntimeNotFound
	}
	return &sdk.Installation{Target: target}, nil
}

func TestDetectCandidates(t *testing.T) {
	s := fakeSdk{installed: map[sdk.Version]string{
		"17":     "",                        // installed by vfox
		"21-tem": "/usr/lib/jvm/temurin-21", // linked
	}}
	found := []sdk.DetectedRuntime{
		{Version: "17", Path: "/usr/lib/jvm/java-17"},            // name taken
		{Version: "21-other", Path: "/usr/lib/jvm/temurin-21/"},  // directory already linked (trailing slash)
		{Version: "11", Path: "/usr/lib/jvm/java-11", Note: "n"}, // new
		{Version: "8", Path: "/usr/lib/jvm/java-8"},              // new
	}
	candidates, skipped := detectCandidates(s, found)

	if want := []sdk.DetectedRuntime{found[2], found[3]}; !reflect.DeepEqual(candidates, want) {
		t.Errorf("candidates = %+v, want %+v", candidates, want)
	}
	if len(skipped) != 2 ||
		!strings.Contains(skipped[0], "17: a version of that name is already installed") ||
		!strings.Contains(skipped[1], "21-other: /usr/lib/jvm/temurin-21/ is already linked as 21-tem") {
		t.Errorf("skipped = %q", skipped)
	}
}

func TestRenderFindings(t *testing.T) {
	findings := []sdk.Finding{
		{ID: "sdk.link-dangling", Severity: sdk.SeverityError, SDK: "java", Version: "17", Message: "java@17 is broken.", Fix: []string{"vfox uninstall java@17"}},
		{ID: "path.shadowed", Severity: sdk.SeverityWarn, SDK: "nodejs", Version: "24", Message: "`node` is shadowed.", Hints: []string{"because of pnpm"}},
		{ID: "sdk.no-receipt", Severity: sdk.SeverityInfo, SDK: "go", Version: "1.25", Message: "old install."},
		{ID: "path.not-activated", Severity: sdk.SeverityError, Message: "not activated."},
		{ID: "sdk.drift", Severity: sdk.SeverityWarn, SDK: "flutter", Message: "silenced.", Ignored: true},
	}
	var out bytes.Buffer
	if err := renderFindings(&out, findings); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"Found 2 errors, 1 warning and 1 notice.",
		"1 finding ignored by diagnostics.ignore.",
		"error  sdk.link-dangling  java@17",
		"       java@17 is broken.",
		"       fix: vfox uninstall java@17",
		"warn   path.shadowed  nodejs@24",
		"       because of pnpm",
		"info   sdk.no-receipt  go@1.25",
		"error  path.not-activated\n", // no subject, no trailing spaces
		"sdk.drift  flutter  (ignored)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output lacks %q:\n%s", want, got)
		}
	}

	out.Reset()
	if err := renderFindings(&out, nil); err != nil || out.String() != "No problems found.\n" {
		t.Errorf("no findings: %q, %v", out.String(), err)
	}
}

type failingWriter struct{ after int }

func (w *failingWriter) Write(p []byte) (int, error) {
	if w.after <= 0 {
		return 0, errors.New("disk full")
	}
	w.after--
	return len(p), nil
}

func TestRenderFindingsReportsWriteErrors(t *testing.T) {
	findings := []sdk.Finding{{ID: "a", Severity: sdk.SeverityError, Message: "m", Hints: []string{"h"}, Fix: []string{"f"}}}
	// The summary, the header, the message, the hint and the fix are five writes.
	for after := 0; after < 5; after++ {
		if err := renderFindings(&failingWriter{after: after}, findings); err == nil {
			t.Errorf("a writer that fails after %d writes must make renderFindings fail", after)
		}
	}
	// So are the answer to "nothing found" and the note about what is ignored.
	if err := renderFindings(&failingWriter{}, nil); err == nil {
		t.Error("a writer that fails must make renderFindings fail when there is nothing to report")
	}
	ignored := []sdk.Finding{{ID: "a", Severity: sdk.SeverityError, Message: "m", Ignored: true}}
	for after := 0; after < 4; after++ {
		if err := renderFindings(&failingWriter{after: after}, ignored); err == nil {
			t.Errorf("a writer that fails after %d writes must make renderFindings fail for an ignored finding", after)
		}
	}
}

func TestHasErrorsIgnoresWhatIsSilenced(t *testing.T) {
	warn := sdk.Finding{Severity: sdk.SeverityWarn}
	bad := sdk.Finding{Severity: sdk.SeverityError}
	silenced := sdk.Finding{Severity: sdk.SeverityError, Ignored: true}
	tests := []struct {
		findings []sdk.Finding
		want     bool
	}{
		{nil, false},
		{[]sdk.Finding{warn}, false},
		{[]sdk.Finding{warn, bad}, true},
		{[]sdk.Finding{silenced}, false},
		{[]sdk.Finding{silenced, bad}, true},
	}
	for i, tt := range tests {
		if got := hasErrors(tt.findings); got != tt.want {
			t.Errorf("case %d: hasErrors = %v, want %v", i, got, tt.want)
		}
	}
	if nonNil(nil) == nil || len(nonNil(nil)) != 0 {
		t.Error("nonNil must turn nil into an empty list, so that JSON has [] and not null")
	}
}

func TestPlural(t *testing.T) {
	tests := map[int]string{0: "0 errors", 1: "1 error", 2: "2 errors", 11: "11 errors"}
	for n, want := range tests {
		if got := plural(n, "error"); got != want {
			t.Errorf("plural(%d) = %q, want %q", n, got, want)
		}
	}
}
