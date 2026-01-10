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

package env

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigState_Load_NotExist(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	state := NewConfigState(stateFile)

	// Loading non-existent file should not error
	err := state.Load()
	if err != nil {
		t.Errorf("Load() should not error for non-existent file, got: %v", err)
	}

	// State should be empty
	if state.LastCheck != 0 {
		t.Error("LastCheck should be zero for non-existent file")
	}
}

func TestConfigState_SaveAndLoad(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create state and save
	state := NewConfigState(stateFile)
	state.LastCheck = time.Now().Unix()
	state.GlobalMtime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()
	state.CachedOutput = "export PATH=/test"

	err := state.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load into new state
	state2 := NewConfigState(stateFile)
	err = state2.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify loaded data
	if state2.LastCheck == 0 {
		t.Error("LastCheck should not be zero")
	}
	if state2.GlobalMtime != state.GlobalMtime {
		t.Errorf("GlobalMtime = %d, want %d", state2.GlobalMtime, state.GlobalMtime)
	}
	if state2.CachedOutput != state.CachedOutput {
		t.Errorf("CachedOutput = %q, want %q", state2.CachedOutput, state.CachedOutput)
	}
}

func TestConfigState_HasChanged_NoChange(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	configFile := filepath.Join(tmpDir, ".vfox.toml")

	// Create config file
	err := os.WriteFile(configFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create state and update with current mtime
	state := NewConfigState(stateFile)
	configPaths := map[UseScope]string{
		Global: configFile,
	}

	err = state.Update(configPaths, "export PATH=/test")
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Check if changed (should be false)
	changed, err := state.HasChanged(configPaths)
	if err != nil {
		t.Fatalf("HasChanged() failed: %v", err)
	}
	if changed {
		t.Error("HasChanged() should return false when config hasn't changed")
	}
}

func TestConfigState_HasChanged_FileModified(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	configFile := filepath.Join(tmpDir, ".vfox.toml")

	// Create config file
	err := os.WriteFile(configFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create state and update with current mtime
	state := NewConfigState(stateFile)
	configPaths := map[UseScope]string{
		Global: configFile,
	}

	err = state.Update(configPaths, "export PATH=/test")
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Modify config file
	time.Sleep(time.Second + 100*time.Millisecond) // Ensure different mtime (Unix timestamp is second precision)
	err = os.WriteFile(configFile, []byte("modified"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify config file: %v", err)
	}

	// Check if changed (should be true)
	changed, err := state.HasChanged(configPaths)
	if err != nil {
		t.Fatalf("HasChanged() failed: %v", err)
	}
	if !changed {
		t.Error("HasChanged() should return true when config has been modified")
	}
}

func TestConfigState_HasChanged_FileDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	configFile := filepath.Join(tmpDir, ".vfox.toml")

	// Create config file
	err := os.WriteFile(configFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Create state and update with current mtime
	state := NewConfigState(stateFile)
	configPaths := map[UseScope]string{
		Global: configFile,
	}

	err = state.Update(configPaths, "export PATH=/test")
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Delete config file
	err = os.Remove(configFile)
	if err != nil {
		t.Fatalf("Failed to delete config file: %v", err)
	}

	// Check if changed (should be true)
	changed, err := state.HasChanged(configPaths)
	if err != nil {
		t.Fatalf("HasChanged() failed: %v", err)
	}
	if !changed {
		t.Error("HasChanged() should return true when config has been deleted")
	}
}

func TestConfigState_HasChanged_EmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create state
	state := NewConfigState(stateFile)
	configPaths := map[UseScope]string{
		Global: "",
	}

	// Update with empty path
	err := state.Update(configPaths, "export PATH=/test")
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Check if changed (should be false)
	changed, err := state.HasChanged(configPaths)
	if err != nil {
		t.Fatalf("HasChanged() failed: %v", err)
	}
	if changed {
		t.Error("HasChanged() should return false when path is empty")
	}
}

func TestConfigState_Update_MultipleScopes(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	// Create multiple config files
	globalFile := filepath.Join(tmpDir, "global.toml")
	sessionFile := filepath.Join(tmpDir, "session.toml")
	projectFile := filepath.Join(tmpDir, "project.toml")

	err := os.WriteFile(globalFile, []byte("global"), 0644)
	if err != nil {
		t.Fatalf("Failed to create global config: %v", err)
	}
	err = os.WriteFile(sessionFile, []byte("session"), 0644)
	if err != nil {
		t.Fatalf("Failed to create session config: %v", err)
	}
	err = os.WriteFile(projectFile, []byte("project"), 0644)
	if err != nil {
		t.Fatalf("Failed to create project config: %v", err)
	}

	// Update state with all scopes
	state := NewConfigState(stateFile)
	configPaths := map[UseScope]string{
		Global:  globalFile,
		Session: sessionFile,
		Project: projectFile,
	}

	err = state.Update(configPaths, "export PATH=/test")
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify all mtimes are set
	if state.GlobalMtime == 0 {
		t.Error("GlobalMtime should be set")
	}
	if state.SessionMtime == 0 {
		t.Error("SessionMtime should be set")
	}
	if state.ProjectMtime == 0 {
		t.Error("ProjectMtime should be set")
	}
}

func TestConfigState_GetCachedOutput(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	state := NewConfigState(stateFile)
	state.CachedOutput = "export TEST=value"

	// Get cached output
	output := state.GetCachedOutput()
	if output != "export TEST=value" {
		t.Errorf("GetCachedOutput() = %q, want %q", output, "export TEST=value")
	}
}
