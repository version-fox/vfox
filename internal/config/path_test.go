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

// A file that only lists "preserve" must keep the default mode.
func TestPathKeepsDefaultsForUnsetKeys(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	content := "path:\n  preserve: [\"~/.cargo/bin\"]\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := NewConfigWithPath(file)
	if err != nil {
		t.Fatal(err)
	}
	if c.Path.EffectiveMode() != PathModeLegacy {
		t.Errorf("path.mode = %q, want legacy when the file only sets preserve", c.Path.Mode)
	}
	if !reflect.DeepEqual(c.Path.Preserve, []string{"~/.cargo/bin"}) {
		t.Errorf("preserve = %v", c.Path.Preserve)
	}
}

func TestMissingPathGetsDefaults(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(file, []byte("proxy:\n  enable: false\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c, err := NewConfigWithPath(file)
	if err != nil {
		t.Fatal(err)
	}
	if c.Path != EmptyPath {
		t.Error("a section the file does not declare must be the shared default")
	}
	if c.Path.EffectiveMode() != PathModeLegacy {
		t.Error("defaults changed")
	}
}

func TestPathEffectiveMode(t *testing.T) {
	tests := map[string]string{
		"":           PathModeLegacy,
		"legacy":     PathModeLegacy,
		"baseline":   PathModeBaseline,
		" Baseline ": PathModeBaseline,
		"AGGRESSIVE": PathModeAggressive,
		"whatever":   PathModeLegacy, // unknown falls back instead of failing
	}
	for in, want := range tests {
		if got := (&Path{Mode: in}).EffectiveMode(); got != want {
			t.Errorf("EffectiveMode(%q) = %q, want %q", in, got, want)
		}
	}
	var none *Path
	if none.EffectiveMode() != PathModeLegacy {
		t.Error("a missing section must mean legacy")
	}
}

func TestMergePath(t *testing.T) {
	shared := &Config{Path: &Path{Mode: PathModeBaseline}}

	t.Run("user declares nothing: shared wins", func(t *testing.T) {
		user := &Config{Path: EmptyPath}
		got := Merge(shared, user)
		if got.Path.EffectiveMode() != PathModeBaseline {
			t.Errorf("shared settings were lost: %+v", got.Path)
		}
	})

	t.Run("user declares the section: user wins", func(t *testing.T) {
		user := &Config{Path: &Path{Mode: PathModeAggressive}}
		got := Merge(shared, user)
		if got.Path.EffectiveMode() != PathModeAggressive {
			t.Errorf("user settings were lost: %+v", got.Path)
		}
	})

	t.Run("one side only", func(t *testing.T) {
		if got := Merge(shared, nil); got.Path.EffectiveMode() != PathModeBaseline {
			t.Errorf("shared settings were lost: %+v", got.Path)
		}
		if got := Merge(&Config{}, &Config{}); got.Path != EmptyPath {
			t.Errorf("empty configs must merge to the defaults: %+v", got)
		}
	})
}

func TestPathRoundTripThroughYAML(t *testing.T) {
	dir := t.TempDir()
	c := *DefaultConfig
	c.Path = &Path{Mode: PathModeBaseline, Preserve: []string{"/opt/tools/bin"}}
	if err := c.SaveConfig(dir); err != nil {
		t.Fatal(err)
	}
	loaded, err := NewConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Path.Mode != PathModeBaseline || !reflect.DeepEqual(loaded.Path.Preserve, c.Path.Preserve) {
		t.Errorf("path = %+v", loaded.Path)
	}
}

func TestPathMustBeAMapping(t *testing.T) {
	file := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(file, []byte("path: 5\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewConfigWithPath(file); err == nil {
		t.Error("a path section that is not a mapping must be reported, not replaced by the defaults")
	}
}
