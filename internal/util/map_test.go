/*
 *    Copyright 2024 Han Li and contributors
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

package util

import (
	"fmt"
	"testing"
)

func TestSortedMap(t *testing.T) {
	sm := NewSortedMap[int, string]()

	// Test Set method
	sm.Set(1, "one")

	// Test Get method
	if _, ok := sm.Get(1); !ok {
		t.Errorf("Expected true, got false")
	}

	// Test Contains method
	if !sm.Contains(1) {
		t.Errorf("Expected true, got false")
	}

	// Test Len method
	if sm.Len() != 1 {
		t.Errorf("Expected 1, got %d", sm.Len())
	}

	// Test ForEach method
	err := sm.ForEach(func(k int, v string) error {
		if k != 1 || v != "one" {
			return fmt.Errorf("Expected key: 1, value: 'one', got key: %d, value: %s", k, v)
		}
		return nil
	})
	if err != nil {
		t.Errorf(err.Error())
	}

	// Test Remove method
	val := sm.Remove(1)
	if val != "one" {
		t.Errorf("Expected 'one', got %s", val)
	}
	if sm.Contains(1) {
		t.Errorf("Expected false, got true")
	}
	if sm.Len() != 0 {
		t.Errorf("Expected 0, got %d", sm.Len())
	}
}

func TestSortedMap_ForEach(t *testing.T) {
	sm := NewSortedMap[int, string]()
	sm.Set(1, "one")
	sm.Set(2, "two")
	sm.Set(3, "three")

	var keys []int
	var values []string
	_ = sm.ForEach(func(k int, v string) error {
		keys = append(keys, k)
		values = append(values, v)
		return nil
	})

	if len(keys) != 3 {
		t.Errorf("Expected 3, got %d", len(keys))
	}
	if len(values) != 3 {
		t.Errorf("Expected 3, got %d", len(values))
	}
	if keys[0] != 1 || keys[1] != 2 || keys[2] != 3 {
		t.Errorf("Expected [1 2 3], got %v", keys)
	}
	if values[0] != "one" || values[1] != "two" || values[2] != "three" {
		t.Errorf("Expected ['one' 'two' 'three'], got %v", values)
	}
}
