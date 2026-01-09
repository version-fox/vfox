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
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox/internal/shared/util"
)

func TestVfoxTomlChain(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("empty chain", func(t *testing.T) {
		chain := NewVfoxTomlChain()
		if !chain.IsEmpty() {
			t.Error("Expected chain to be empty")
		}
		if chain.Length() != 0 {
			t.Errorf("Expected length 0, got %d", chain.Length())
		}

		// Merge of empty chain should return empty config
		merged := chain.Merge()
		if merged == nil {
			t.Error("Expected merged config, got nil")
		}
		if !merged.IsEmpty() {
			t.Error("Expected merged config to be empty")
		}
	})

	t.Run("add and get configs", func(t *testing.T) {
		chain := NewVfoxTomlChain()

		config1 := NewVfoxToml()
		config1.Path = filepath.Join(tempDir, "config1.toml")
		config1.SetTool("nodejs", "21.5.0")

		config2 := NewVfoxToml()
		config2.Path = filepath.Join(tempDir, "config2.toml")
		config2.SetTool("java", "21")

		chain.Add(config1)
		chain.Add(config2)

		if chain.IsEmpty() {
			t.Error("Expected chain to not be empty")
		}
		if chain.Length() != 2 {
			t.Errorf("Expected length 2, got %d", chain.Length())
		}

		// Get by index
		retrieved := chain.GetByIndex(0)
		if retrieved != config1 {
			t.Error("Expected first config to be config1")
		}

		retrieved = chain.GetByIndex(1)
		if retrieved != config2 {
			t.Error("Expected second config to be config2")
		}

		// Get out of bounds
		retrieved = chain.GetByIndex(10)
		if retrieved != nil {
			t.Error("Expected nil for out of bounds index")
		}
	})

	t.Run("merge configs with priority", func(t *testing.T) {
		chain := NewVfoxTomlChain()

		// Global config
		globalConfig := NewVfoxToml()
		globalConfig.SetTool("nodejs", "20.0.0")
		globalConfig.SetTool("java", "17")

		// Project config (higher priority)
		projectConfig := NewVfoxToml()
		projectConfig.SetTool("nodejs", "21.5.0")
		projectConfig.SetTool("python", "3.12")

		chain.Add(globalConfig)
		chain.Add(projectConfig)

		merged := chain.Merge()
		if merged == nil {
			t.Fatal("Expected merged config, got nil")
		}

		// Check that project override takes effect
		version, ok := merged.Tools.GetVersion("nodejs")
		if !ok {
			t.Error("Expected nodejs to be found")
		}
		if version != "21.5.0" {
			t.Errorf("Expected nodejs version 21.5.0 (project), got %s", version)
		}

		// Check that global-only tools are included
		version, ok = merged.Tools.GetVersion("java")
		if !ok {
			t.Error("Expected java to be found")
		}
		if version != "17" {
			t.Errorf("Expected java version 17, got %s", version)
		}

		// Check that project-only tools are included
		version, ok = merged.Tools.GetVersion("python")
		if !ok {
			t.Error("Expected python to be found")
		}
		if version != "3.12" {
			t.Errorf("Expected python version 3.12, got %s", version)
		}
	})

	t.Run("GetAllTools", func(t *testing.T) {
		chain := NewVfoxTomlChain()

		config1 := NewVfoxToml()
		config1.SetTool("nodejs", "20.0.0")
		config1.SetTool("java", "17")

		config2 := NewVfoxToml()
		config2.SetTool("nodejs", "21.5.0")
		config2.SetTool("python", "3.12")

		chain.Add(config1)
		chain.Add(config2)

		allTools := chain.GetAllTools()
		if len(allTools) != 3 {
			t.Errorf("Expected 3 tools, got %d", len(allTools))
		}

		// Check that higher priority wins
		if allTools["nodejs"] != "21.5.0" {
			t.Errorf("Expected nodejs@21.5.0, got %s", allTools["nodejs"])
		}
		if allTools["java"] != "17" {
			t.Errorf("Expected java@17, got %s", allTools["java"])
		}
		if allTools["python"] != "3.12" {
			t.Errorf("Expected python@3.12, got %s", allTools["python"])
		}
	})

	t.Run("GetTool with priority", func(t *testing.T) {
		chain := NewVfoxTomlChain()

		globalConfig := NewVfoxToml()
		globalConfig.SetTool("nodejs", "20.0.0")

		projectConfig := NewVfoxToml()
		projectConfig.SetTool("nodejs", "21.5.0")

		chain.Add(globalConfig)
		chain.Add(projectConfig)

		// Should find in project (higher priority)
		config, ok := chain.GetToolConfig("nodejs")
		if !ok {
			t.Error("Expected nodejs to be found")
		}
		if config.Version != "21.5.0" {
			t.Errorf("Expected version 21.5.0, got %s", config.Version)
		}
		// Empty map is acceptable for no attributes
		if config.Attr != nil && len(config.Attr) != 0 {
			t.Errorf("Expected empty attributes, got %v", config.Attr)
		}
	})

	t.Run("GetTool not found", func(t *testing.T) {
		chain := NewVfoxTomlChain()

		config := NewVfoxToml()
		config.SetTool("nodejs", "21.5.0")

		chain.Add(config)

		_, ok := chain.GetToolVersion("java")
		if ok {
			t.Error("Expected java to not be found")
		}
	})

	t.Run("AddTool to all configs", func(t *testing.T) {
		chain := NewVfoxTomlChain()

		config1 := NewVfoxToml()
		config2 := NewVfoxToml()

		chain.Add(config1)
		chain.Add(config2)

		chain.AddTool("nodejs", "21.5.0")

		// Check that tool was added to all configs
		version, ok := config1.Tools.GetVersion("nodejs")
		if !ok || version != "21.5.0" {
			t.Errorf("Expected nodejs@21.5.0 in config1, got %v", version)
		}

		version, ok = config2.Tools.GetVersion("nodejs")
		if !ok || version != "21.5.0" {
			t.Errorf("Expected nodejs@21.5.0 in config2, got %v", version)
		}
	})

	t.Run("RemoveTool from all configs", func(t *testing.T) {
		chain := NewVfoxTomlChain()

		config1 := NewVfoxToml()
		config1.SetTool("nodejs", "21.5.0")
		config1.SetTool("java", "17")

		config2 := NewVfoxToml()
		config2.SetTool("nodejs", "20.0.0")
		config2.SetTool("python", "3.12")

		chain.Add(config1)
		chain.Add(config2)

		chain.RemoveTool("nodejs")

		// Check that tool was removed from all configs
		_, ok := config1.Tools.GetVersion("nodejs")
		if ok {
			t.Error("Expected nodejs to be removed from config1")
		}

		_, ok = config2.Tools.GetVersion("nodejs")
		if ok {
			t.Error("Expected nodejs to be removed from config2")
		}

		// Check that other tools are still there
		_, ok = config1.Tools.GetVersion("java")
		if !ok {
			t.Error("Expected java to still exist in config1")
		}

		_, ok = config2.Tools.GetVersion("python")
		if !ok {
			t.Error("Expected python to still exist in config2")
		}
	})

	t.Run("Save all configs", func(t *testing.T) {
		tempDir := t.TempDir()
		chain := NewVfoxTomlChain()

		config1 := NewVfoxToml()
		config1.Path = filepath.Join(tempDir, "config1.toml")
		config1.SetTool("nodejs", "21.5.0")

		config2 := NewVfoxToml()
		config2.Path = filepath.Join(tempDir, "config2.toml")
		config2.SetTool("java", "17")

		chain.Add(config1)
		chain.Add(config2)

		// Save all configs
		if err := chain.Save(); err != nil {
			t.Fatalf("Failed to save chain: %v", err)
		}

		// Verify files were created
		if !util.FileExists(config1.Path) {
			t.Errorf("Expected config1 file to be created at %s", config1.Path)
		}
		if !util.FileExists(config2.Path) {
			t.Errorf("Expected config2 file to be created at %s", config2.Path)
		}

		// Verify content
		loaded1, err := LoadVfoxToml(config1.Path)
		if err != nil {
			t.Fatalf("Failed to load config1: %v", err)
		}
		version, ok := loaded1.Tools.GetVersion("nodejs")
		if !ok || version != "21.5.0" {
			t.Errorf("Expected nodejs@21.5.0 in config1, got %v", version)
		}

		loaded2, err := LoadVfoxToml(config2.Path)
		if err != nil {
			t.Fatalf("Failed to load config2: %v", err)
		}
		version, ok = loaded2.Tools.GetVersion("java")
		if !ok || version != "17" {
			t.Errorf("Expected java@17 in config2, got %v", version)
		}
	})

	t.Run("handle nil configs in chain", func(t *testing.T) {
		tempDir := t.TempDir()
		chain := NewVfoxTomlChain()

		config1 := NewVfoxToml()
		config1.Path = filepath.Join(tempDir, "config1.toml")
		config1.SetTool("nodejs", "21.5.0")

		config3 := NewVfoxToml()
		config3.Path = filepath.Join(tempDir, "config3.toml")

		chain.Add(config1)
		chain.Add(nil) // Add nil config
		chain.Add(config3)

		// Should handle gracefully
		merged := chain.Merge()
		if merged == nil {
			t.Fatal("Expected merged config, got nil")
		}

		// Should only have tools from non-nil configs
		version, ok := merged.Tools.GetVersion("nodejs")
		if !ok || version != "21.5.0" {
			t.Errorf("Expected nodejs@21.5.0, got %v", version)
		}

		// Chain operations should handle nil
		chain.AddTool("java", "17")
		chain.RemoveTool("nodejs")

		// Save should handle nil configs (only save configs with paths)
		if err := chain.Save(); err != nil {
			t.Errorf("Save should handle nil configs: %v", err)
		}
	})

	t.Run("merge with nil configs", func(t *testing.T) {
		chain := NewVfoxTomlChain()

		config1 := NewVfoxToml()
		config1.SetTool("nodejs", "21.5.0")

		chain.Add(nil)
		chain.Add(config1)
		chain.Add(nil)

		merged := chain.Merge()
		if merged == nil {
			t.Fatal("Expected merged config, got nil")
		}

		version, ok := merged.Tools.GetVersion("nodejs")
		if !ok || version != "21.5.0" {
			t.Errorf("Expected nodejs@21.5.0, got %v", version)
		}
	})
}
