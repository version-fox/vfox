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
		{"nodejs@v24.21.0", "nodejs", "24.21.0"}, // the spelling `vfox current` prints
		{"nodejs@", "nodejs", ""},
	}
	for _, tt := range tests {
		name, version := splitDoctorSpec(tt.spec)
		if name != tt.name || string(version) != tt.version {
			t.Errorf("splitDoctorSpec(%q) = (%q, %q), want (%q, %q)", tt.spec, name, version, tt.name, tt.version)
		}
	}
}

func TestFormatVersionLabel(t *testing.T) {
	tests := []struct {
		name    string
		version sdk.Version
		origin  sdk.Origin
		target  string
		want    string
	}{
		{"installed", "17.0.9", sdk.OriginInstalled, "", "17.0.9"},
		{"linked", "17-sys", sdk.OriginLinked, "/usr/lib/jvm/x", "17-sys (linked → /usr/lib/jvm/x)"},
		{"linked, target unknown", "17-sys", sdk.OriginLinked, "", "17-sys (linked)"},
	}
	for _, tt := range tests {
		if got := formatVersionLabel(tt.version, tt.origin, tt.target); got != tt.want {
			t.Errorf("%s: label = %q, want %q", tt.name, got, tt.want)
		}
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
