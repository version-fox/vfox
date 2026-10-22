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
