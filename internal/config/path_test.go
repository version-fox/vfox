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
	// `vfox config path.mode` prints the stored value, so it has to be the default itself, not just resolve to it.
	if c.Path.Mode != PathModeBaseline || c.Path.EffectiveMode() != PathModeBaseline {
		t.Errorf("path.mode = %q, want baseline when the file only sets preserve", c.Path.Mode)
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
	if c.Path.Mode != PathModeBaseline || c.Path.EffectiveMode() != PathModeBaseline {
		t.Errorf("defaults changed: path.mode = %q", c.Path.Mode)
	}
}

func TestPathEffectiveMode(t *testing.T) {
	tests := map[string]string{
		"":           PathModeBaseline, // not configured: the default
		"legacy":     PathModeLegacy,
		" Legacy ":   PathModeLegacy,
		"baseline":   PathModeBaseline,
		" Baseline ": PathModeBaseline,
		"AGGRESSIVE": PathModeAggressive,
		"whatever":   PathModeBaseline, // unknown falls back to the default instead of failing
	}
	for in, want := range tests {
		if got := (&Path{Mode: in}).EffectiveMode(); got != want {
			t.Errorf("EffectiveMode(%q) = %q, want %q", in, got, want)
		}
	}
	var none *Path
	if none.EffectiveMode() != PathModeBaseline {
		t.Error("a missing section must mean the default, baseline")
	}
}

// The default is the one setting that every fallback has to agree on.
func TestDefaultPathModeIsBaseline(t *testing.T) {
	if DefaultPathMode != PathModeBaseline {
		t.Errorf("DefaultPathMode = %q, want baseline", DefaultPathMode)
	}
	if EmptyPath.Mode != DefaultPathMode || EmptyPath.EffectiveMode() != DefaultPathMode {
		t.Errorf("EmptyPath = %+v, want the default mode", EmptyPath)
	}
	if DefaultConfig.Path != EmptyPath {
		t.Error("the default configuration must use EmptyPath")
	}
}

func TestMergePath(t *testing.T) {
	// Not the default, so that a setting that gets lost shows.
	shared := &Config{Path: &Path{Mode: PathModeLegacy}}

	t.Run("user declares nothing: shared wins", func(t *testing.T) {
		user := &Config{Path: EmptyPath}
		got := Merge(shared, user)
		if got.Path.EffectiveMode() != PathModeLegacy {
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
		if got := Merge(shared, nil); got.Path.EffectiveMode() != PathModeLegacy {
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
	// legacy is what people write down to keep the behavior of vfox 1.0.x, and it is not the default.
	c.Path = &Path{Mode: PathModeLegacy, Preserve: []string{"/opt/tools/bin"}}
	if err := c.SaveConfig(dir); err != nil {
		t.Fatal(err)
	}
	loaded, err := NewConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Path.Mode != PathModeLegacy || !reflect.DeepEqual(loaded.Path.Preserve, c.Path.Preserve) {
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
