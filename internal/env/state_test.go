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
	state.GlobalMtime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()
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

	err = state.Update(configPaths, "export PATH=/test", "bash")
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

	err = state.Update(configPaths, "export PATH=/test", "bash")
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Modify config file, with an mtime a minute ahead so the change does not
	// depend on the clock or the timestamp precision of the file system
	err = os.WriteFile(configFile, []byte("modified"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify config file: %v", err)
	}
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(configFile, later, later); err != nil {
		t.Fatalf("Failed to set the mtime of the config file: %v", err)
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

	err = state.Update(configPaths, "export PATH=/test", "bash")
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
	err := state.Update(configPaths, "export PATH=/test", "bash")
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

	err = state.Update(configPaths, "export PATH=/test", "bash")
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
	for _, tc := range []struct{ name, cachedShell, targetShell, want string }{
		{"same shell", "bash", "bash", "export TEST=value"},
		{"case insensitive", "bash", "BASH", "export TEST=value"},
		{"different shell", "pwsh", "fish", ""},
		{"legacy cache", "", "bash", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := NewConfigState(filepath.Join(t.TempDir(), "state.json"))
			if err := state.Update(nil, "export TEST=value", tc.cachedShell); err != nil {
				t.Fatal(err)
			}
			loaded := NewConfigState(state.stateFilePath)
			if err := loaded.Load(); err != nil {
				t.Fatal(err)
			}
			if got := loaded.GetCachedOutput(tc.targetShell); got != tc.want {
				t.Errorf("GetCachedOutput(%q) = %q, want %q", tc.targetShell, got, tc.want)
			}
		})
	}
}

func TestConfigState_HasChanged_PathChanged(t *testing.T) {
	// Save original PATH
	origPath := os.Getenv("PATH")
	defer func() {
		os.Setenv("PATH", origPath)
	}()

	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")
	configFile := filepath.Join(tmpDir, ".vfox.toml")

	// Create config file
	err := os.WriteFile(configFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Set initial PATH
	initialPath := "/usr/bin:/bin"
	os.Setenv("PATH", initialPath)

	// Create state and update with current mtime and PATH
	state := NewConfigState(stateFile)
	configPaths := map[UseScope]string{
		Global: configFile,
	}

	err = state.Update(configPaths, "export PATH=/test", "bash")
	if err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Verify cached PATH
	if state.CachedPath != initialPath {
		t.Errorf("CachedPath = %q, want %q", state.CachedPath, initialPath)
	}

	// Check if changed (should be false, PATH and configs are the same)
	changed, err := state.HasChanged(configPaths)
	if err != nil {
		t.Fatalf("HasChanged() failed: %v", err)
	}
	if changed {
		t.Error("HasChanged() should return false when nothing has changed")
	}

	// User manually adds a path
	modifiedPath := "/home/user/bin:/usr/bin:/bin"
	os.Setenv("PATH", modifiedPath)

	// Check if changed (should be true, PATH changed)
	changed, err = state.HasChanged(configPaths)
	if err != nil {
		t.Fatalf("HasChanged() failed: %v", err)
	}
	if !changed {
		t.Error("HasChanged() should return true when PATH has changed")
	}
}

// Output rendered under one PATH ordering policy must not be served under another.
func TestConfigState_PathPolicyChanged(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	state := NewConfigState(stateFile)

	if !state.PathPolicyChanged("legacy") {
		t.Fatal("a state that never recorded a policy must count as changed")
	}
	state.SetPathPolicy("legacy")
	if err := state.Update(map[UseScope]string{}, "export PATH=/test", "bash"); err != nil {
		t.Fatalf("Update() failed: %v", err)
	}
	if state.PathPolicyChanged("legacy") {
		t.Error("the recorded policy must not count as changed")
	}
	if !state.PathPolicyChanged("baseline") {
		t.Error("another policy must count as changed")
	}

	// The policy survives a round trip through the state file, as a new prompt would see it.
	reloaded := NewConfigState(stateFile)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if reloaded.PathPolicyChanged("legacy") || !reloaded.PathPolicyChanged("aggressive") {
		t.Errorf("reloaded policy = %q", reloaded.CachedPathPolicy)
	}
}

// Output rendered under one resolution key (project tools, installs, links)
// must not be served under another.
func TestConfigState_ResolutionKeyChanged(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.json")
	state := NewConfigState(stateFile)

	if state.ResolutionKeyChanged("") {
		t.Fatal("an empty key must match a state that never recorded one")
	}
	if !state.ResolutionKeyChanged("nodejs = \"20\"\n") {
		t.Fatal("a key must count as changed against a state without one")
	}
	state.SetResolutionKey("nodejs = \"20\"\n")
	if err := state.Update(map[UseScope]string{}, "export PATH=/test", "bash"); err != nil {
		t.Fatalf("Update() failed: %v", err)
	}
	if state.ResolutionKeyChanged("nodejs = \"20\"\n") {
		t.Error("the recorded key must not count as changed")
	}
	if !state.ResolutionKeyChanged("nodejs = \"22\"\n") || !state.ResolutionKeyChanged("") {
		t.Error("a different or empty key must count as changed")
	}

	// The key survives a round trip through the state file, as a new prompt would see it.
	reloaded := NewConfigState(stateFile)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if reloaded.ResolutionKeyChanged("nodejs = \"20\"\n") {
		t.Errorf("reloaded resolution key = %q", reloaded.CachedResolutionKey)
	}
}

// A config file whose modification time moves back (replaced by an older
// copy) or stays within the stored second has changed as well.
func TestConfigState_HasChanged_AnyOtherMtime(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "vfox.toml")
	if err := os.WriteFile(configFile, []byte("[tools]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(PathVarName, "/test")
	state := NewConfigState(filepath.Join(dir, "state.json"))
	configPaths := map[UseScope]string{Global: configFile}

	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		what  string
		mtime time.Time
	}{
		{"older", info.ModTime().Add(-48 * time.Hour)},
		{"a millisecond later", info.ModTime().Add(-48 * time.Hour).Add(time.Millisecond)},
	} {
		if err := state.Update(configPaths, "export PATH=/test", "bash"); err != nil {
			t.Fatalf("Update() failed: %v", err)
		}
		if changed, err := state.HasChanged(configPaths); changed || err != nil {
			t.Fatalf("%s: before the change: changed = %v, err = %v", c.what, changed, err)
		}
		if err := os.Chtimes(configFile, c.mtime, c.mtime); err != nil {
			t.Fatal(err)
		}
		if changed, err := state.HasChanged(configPaths); !changed || err != nil {
			t.Errorf("%s: changed = %v, err = %v, want a change", c.what, changed, err)
		}
	}
}
