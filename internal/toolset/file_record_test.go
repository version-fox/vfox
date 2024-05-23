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

package toolset

import (
	"os"
	"testing"
)

func TestFileRecord(t *testing.T) {
	// Create a temporary file for testing
	file, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	fm, _ := NewFileRecord(file.Name())

	// Test Set method
	fm.Set("key", "value")

	// Test Get method
	if v, ok := fm.Get("key"); !ok || v != "value" {
		t.Errorf("Expected 'value', got %s", v)
	}

	// Test Contains method
	if !fm.Contains("key") {
		t.Errorf("Expected true, got false")
	}

	// Test Len method
	if fm.Len() != 1 {
		t.Errorf("Expected 1, got %d", fm.Len())
	}

	// Test ForEach method
	err = fm.ForEach(func(k string, v string) error {
		if k != "key" || v != "value" {
			t.Errorf("Expected key: 'key', value: 'value', got key: %s, value: %s", k, v)
		}
		return nil
	})
	if err != nil {
		t.Errorf("ForEach method failed with error: %v", err)
	}

	// Test Save method
	err = fm.Save()
	if err != nil {
		t.Errorf("Save method failed with error: %v", err)
	}

	// Test Remove method
	v := fm.Remove("key")
	if v != "value" {
		t.Errorf("Expected 'value', got %s", v)
	}
	if fm.Contains("key") {
		t.Errorf("Expected false, got true")
	}
	if fm.Len() != 0 {
		t.Errorf("Expected 0, got %d", fm.Len())
	}
}
