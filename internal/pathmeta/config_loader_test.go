/*
 *
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
 *
 */

package pathmeta

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox/internal/shared/util"
)

func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("load from non-existent directory", func(t *testing.T) {
		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if config == nil {
			t.Fatal("Expected config to be returned")
		}
		// Path should be set to default (.vfox.toml)
		if config.Path == "" {
			t.Error("Expected config.Path to be set")
		}
		if filepath.Base(config.Path) != ".vfox.toml" {
			t.Errorf("Expected path to end with .vfox.toml, got %s", config.Path)
		}
	})

	t.Run("load from .vfox.toml", func(t *testing.T) {
		// Create .vfox.toml
		tomlPath := filepath.Join(tempDir, ".vfox.toml")
		content := "[tools]\nnodejs = \"21.5.0\"\n"
		if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create .vfox.toml: %v", err)
		}

		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if config.Path != tomlPath {
			t.Errorf("Expected path %s, got %s", tomlPath, config.Path)
		}
		version, ok := config.Tools.GetVersion("nodejs")
		if !ok {
			t.Error("Expected nodejs to be found")
		}
		if version != "21.5.0" {
			t.Errorf("Expected version 21.5.0, got %s", version)
		}
	})

	t.Run("load from vfox.toml (fallback)", func(t *testing.T) {
		// Use a unique temp directory
		testDir := t.TempDir()
		// Create vfox.toml
		tomlPath := filepath.Join(testDir, "vfox.toml")
		content := "[tools]\njava = \"21\"\n"
		if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create vfox.toml: %v", err)
		}

		config, err := LoadConfig(testDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if config.Path != tomlPath {
			t.Errorf("Expected path %s, got %s", tomlPath, config.Path)
		}
		version, ok := config.Tools.GetVersion("java")
		if !ok {
			t.Error("Expected java to be found")
		}
		if version != "21" {
			t.Errorf("Expected version 21, got %s", version)
		}
	})

	t.Run("priority: .vfox.toml > vfox.toml", func(t *testing.T) {
		// Create both files
		dotTomlPath := filepath.Join(tempDir, ".vfox.toml")
		tomlPath := filepath.Join(tempDir, "vfox.toml")

		dotContent := "[tools]\nnodejs = \"21.5.0\"\n"
		tomlContent := "[tools]\nnodejs = \"20.0.0\"\n"

		if err := os.WriteFile(dotTomlPath, []byte(dotContent), 0644); err != nil {
			t.Fatalf("Failed to create .vfox.toml: %v", err)
		}
		if err := os.WriteFile(tomlPath, []byte(tomlContent), 0644); err != nil {
			t.Fatalf("Failed to create vfox.toml: %v", err)
		}

		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if config.Path != dotTomlPath {
			t.Errorf("Expected path %s (priority), got %s", dotTomlPath, config.Path)
		}
		version, ok := config.Tools.GetVersion("nodejs")
		if !ok {
			t.Error("Expected nodejs to be found")
		}
		if version != "21.5.0" {
			t.Errorf("Expected version 21.5.0 from .vfox.toml, got %s", version)
		}
	})

	t.Run("migrate from .tool-versions", func(t *testing.T) {
		// Create .tool-versions
		toolVersionsPath := filepath.Join(tempDir, ".tool-versions")
		content := "nodejs 21.5.0\njava 21\n"
		if err := os.WriteFile(toolVersionsPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create .tool-versions: %v", err)
		}

		config, err := LoadConfig(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Check that config was loaded
		version, ok := config.Tools.GetVersion("nodejs")
		if !ok {
			t.Error("Expected nodejs to be found")
		}
		if version != "21.5.0" {
			t.Errorf("Expected version 21.5.0, got %s", version)
		}

		// Check that .vfox.toml was created
		vfoxTomlPath := filepath.Join(tempDir, ".vfox.toml")
		if !util.FileExists(vfoxTomlPath) {
			t.Error("Expected .vfox.toml to be created")
		}

		// Check that .tool-versions still exists
		if !util.FileExists(toolVersionsPath) {
			t.Error("Expected .tool-versions to still exist")
		}

		// Verify .vfox.toml content
		loadedConfig, err := LoadVfoxToml(vfoxTomlPath)
		if err != nil {
			t.Fatalf("Failed to load migrated .vfox.toml: %v", err)
		}
		if loadedConfig.Path != vfoxTomlPath {
			t.Errorf("Expected path %s, got %s", vfoxTomlPath, loadedConfig.Path)
		}
	})

	t.Run("save new config", func(t *testing.T) {
		emptyDir := t.TempDir()
		config, err := LoadConfig(emptyDir)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Add a tool and save
		config.SetTool("nodejs", "21.5.0")
		if err := config.Save(); err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		// Verify file was created
		expectedPath := filepath.Join(emptyDir, ".vfox.toml")
		if !util.FileExists(expectedPath) {
			t.Errorf("Expected .vfox.toml to be created at %s", expectedPath)
		}

		// Verify content
		loadedConfig, err := LoadVfoxToml(expectedPath)
		if err != nil {
			t.Fatalf("Failed to load saved config: %v", err)
		}
		version, ok := loadedConfig.Tools.GetVersion("nodejs")
		if !ok || version != "21.5.0" {
			t.Errorf("Expected nodejs@21.5.0, got %v", version)
		}
	})
}

func TestDetermineConfigPath(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("default to .vfox.toml when nothing exists", func(t *testing.T) {
		path := DetermineConfigPath(tempDir)
		expected := filepath.Join(tempDir, ".vfox.toml")
		if path != expected {
			t.Errorf("Expected %s, got %s", expected, path)
		}
	})

	t.Run("prefer .vfox.toml if it exists", func(t *testing.T) {
		dotTomlPath := filepath.Join(tempDir, ".vfox.toml")
		if err := os.WriteFile(dotTomlPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create .vfox.toml: %v", err)
		}

		path := DetermineConfigPath(tempDir)
		if path != dotTomlPath {
			t.Errorf("Expected %s, got %s", dotTomlPath, path)
		}
	})

	t.Run("use vfox.toml if only it exists", func(t *testing.T) {
		testDir := t.TempDir()
		// Don't create .vfox.toml
		tomlPath := filepath.Join(testDir, "vfox.toml")
		if err := os.WriteFile(tomlPath, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create vfox.toml: %v", err)
		}

		path := DetermineConfigPath(testDir)
		if path != tomlPath {
			t.Errorf("Expected %s, got %s", tomlPath, path)
		}
	})
}
