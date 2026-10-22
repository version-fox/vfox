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

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiagnosticsIgnores(t *testing.T) {
	d := &Diagnostics{Ignore: []string{
		"sdk.drift:flutter@3.47.0",
		"path.shadowed:nodejs",
		"sdk.no-receipt",
		"  ",
	}}
	tests := []struct {
		id, sdk, version string
		want             bool
	}{
		{"sdk.drift", "flutter", "3.47.0", true},
		{"sdk.drift", "flutter", "3.47.4", false}, // another version
		{"sdk.drift", "dart", "3.47.0", false},    // another SDK
		{"sdk.drift", "Flutter", "3.47.0", true},  // SDK names are not case sensitive
		{"path.shadowed", "nodejs", "24.21.0", true},
		{"path.shadowed", "nodejs", "", true},
		{"path.shadowed", "java", "21", false},
		{"sdk.no-receipt", "anything", "1", true}, // no scope: every SDK
		{"sdk.link-dangling", "java", "17", false},
	}
	for _, tt := range tests {
		if got := d.Ignores(tt.id, tt.sdk, tt.version); got != tt.want {
			t.Errorf("Ignores(%q, %q, %q) = %v, want %v", tt.id, tt.sdk, tt.version, got, tt.want)
		}
	}
	var none *Diagnostics
	if none.Ignores("sdk.drift", "flutter", "1") {
		t.Error("a missing section must ignore nothing")
	}
}

// A file that only lists "ignore" must not switch the checks off.
func TestDiagnosticsKeepDefaultsForUnsetKeys(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	content := "diagnostics:\n  ignore: [\"sdk.drift\"]\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := NewConfigWithPath(file)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Diagnostics.OnUse {
		t.Error("diagnostics.onUse must stay on when the file does not mention it")
	}
	if !reflect.DeepEqual(c.Diagnostics.Ignore, []string{"sdk.drift"}) {
		t.Errorf("ignore = %v", c.Diagnostics.Ignore)
	}
}

func TestDiagnosticsCanBeSwitchedOff(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(file, []byte("diagnostics:\n  onUse: false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := NewConfigWithPath(file)
	if err != nil {
		t.Fatal(err)
	}
	if c.Diagnostics.OnUse {
		t.Error("diagnostics.onUse: false was not honored")
	}
}

func TestMissingDiagnosticsGetDefaults(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(file, []byte("proxy:\n  enable: false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := NewConfigWithPath(file)
	if err != nil {
		t.Fatal(err)
	}
	if c.Diagnostics != EmptyDiagnostics {
		t.Error("a section the file does not declare must be the shared default")
	}
	if !c.Diagnostics.OnUse {
		t.Error("defaults changed")
	}
}

func TestMergeDiagnostics(t *testing.T) {
	shared := &Config{Diagnostics: &Diagnostics{OnUse: false, Ignore: []string{"sdk.drift"}}}

	t.Run("user declares nothing: shared wins", func(t *testing.T) {
		user := &Config{Diagnostics: EmptyDiagnostics}
		got := Merge(shared, user)
		if got.Diagnostics.OnUse || !reflect.DeepEqual(got.Diagnostics.Ignore, []string{"sdk.drift"}) {
			t.Errorf("shared settings were lost: %+v", got.Diagnostics)
		}
	})

	t.Run("user declares the section: user wins", func(t *testing.T) {
		user := &Config{Diagnostics: &Diagnostics{OnUse: true}}
		got := Merge(shared, user)
		if !got.Diagnostics.OnUse {
			t.Errorf("user settings were lost: %+v", got.Diagnostics)
		}
	})

	t.Run("one side only", func(t *testing.T) {
		if got := Merge(shared, nil); got.Diagnostics.OnUse {
			t.Errorf("shared settings were lost: %+v", got.Diagnostics)
		}
		if got := Merge(&Config{}, &Config{}); got.Diagnostics != EmptyDiagnostics {
			t.Errorf("empty configs must merge to the defaults: %+v", got)
		}
	})
}

func TestDiagnosticsRoundTripThroughYAML(t *testing.T) {
	dir := t.TempDir()
	c := *DefaultConfig
	c.Diagnostics = &Diagnostics{OnUse: false, Ignore: []string{"path.shadowed:nodejs"}}
	if err := c.SaveConfig(dir); err != nil {
		t.Fatal(err)
	}
	loaded, err := NewConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Diagnostics.OnUse || !reflect.DeepEqual(loaded.Diagnostics.Ignore, c.Diagnostics.Ignore) {
		t.Errorf("diagnostics = %+v", loaded.Diagnostics)
	}
}

func TestDiagnosticsMustBeAMapping(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(file, []byte("diagnostics: 5\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewConfigWithPath(file); err == nil {
		t.Error("a diagnostics section that is not a mapping must be reported, not replaced by the defaults")
	}
}
