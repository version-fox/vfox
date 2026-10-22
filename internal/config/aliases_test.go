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
	"reflect"
	"testing"
)

func TestMergeAliases(t *testing.T) {
	shared := &Config{Aliases: map[string]map[string]string{
		"java": {"lts": "21.0.1", "j17": "17.0.9"},
	}}

	t.Run("entries of both are kept, the user wins on a clash", func(t *testing.T) {
		user := &Config{Aliases: map[string]map[string]string{
			"java": {"j17": "17.0.12"}, // overrides one entry
			"go":   {"stable": "1.25.6"},
		}}
		got := Merge(shared, user)
		want := map[string]map[string]string{
			"java": {"lts": "21.0.1", "j17": "17.0.12"},
			"go":   {"stable": "1.25.6"},
		}
		if !reflect.DeepEqual(got.Aliases, want) {
			t.Errorf("aliases = %v, want %v", got.Aliases, want)
		}
	})

	t.Run("merging does not modify its inputs", func(t *testing.T) {
		user := &Config{Aliases: map[string]map[string]string{"java": {"j17": "17.0.12"}}}
		Merge(shared, user)
		if shared.Aliases["java"]["j17"] != "17.0.9" {
			t.Error("Merge modified the shared aliases")
		}
	})

	t.Run("one side only", func(t *testing.T) {
		if got := Merge(shared, nil); !reflect.DeepEqual(got.Aliases, shared.Aliases) {
			t.Errorf("aliases = %v", got.Aliases)
		}
		if got := Merge(&Config{}, &Config{}); got.Aliases != nil {
			t.Errorf("empty configs must merge to no aliases: %v", got.Aliases)
		}
	})
}

func TestCloneIsIndependent(t *testing.T) {
	original := &Config{Aliases: map[string]map[string]string{"java": {"j17": "17.0.9"}}}
	clone := original.Clone()
	clone.Aliases["java"]["j17"] = "changed"
	clone.Aliases["go"] = map[string]string{"x": "1"}
	if original.Aliases["java"]["j17"] != "17.0.9" || len(original.Aliases) != 1 {
		t.Errorf("Clone shares its aliases with the original: %v", original.Aliases)
	}
	if (*Config)(nil).Clone() == nil {
		t.Error("Clone of a missing config must still be usable")
	}
}

func TestAliasesRoundTripThroughYAML(t *testing.T) {
	dir := t.TempDir()
	c := DefaultConfig.Clone()
	c.Aliases = map[string]map[string]string{"java": {"j17": "17.0.9-tem"}}
	if err := c.SaveConfig(dir); err != nil {
		t.Fatal(err)
	}
	loaded, err := NewConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.Aliases, c.Aliases) {
		t.Errorf("aliases = %v", loaded.Aliases)
	}
}

func TestMergeAliasesOfTheUserAlone(t *testing.T) {
	user := &Config{Aliases: map[string]map[string]string{"java": {"j17": "17.0.9"}}}
	got := Merge(&Config{}, user)
	if !reflect.DeepEqual(got.Aliases, user.Aliases) {
		t.Errorf("aliases = %v, want %v", got.Aliases, user.Aliases)
	}
	got.Aliases["java"]["j17"] = "changed"
	if user.Aliases["java"]["j17"] != "17.0.9" {
		t.Error("the merged aliases share their maps with the user's config")
	}
}
