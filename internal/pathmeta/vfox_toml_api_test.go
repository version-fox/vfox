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
	"testing"
)

func TestToolsAPI(t *testing.T) {
	t.Run("Set and GetVersion", func(t *testing.T) {
		tools := make(Tools)
		tools.Set("nodejs", "21.5.0")

		version, ok := tools.GetVersion("nodejs")
		if !ok {
			t.Fatal("Expected nodejs to be found")
		}
		if version != "21.5.0" {
			t.Errorf("Expected version 21.5.0, got %s", version)
		}
	})

	t.Run("SetWithAttr and GetConfig", func(t *testing.T) {
		tools := make(Tools)
		attr := Attr{
			"vendor": "openjdk",
			"dist":   "temurin",
			"arch":   "x64",
		}
		tools.SetWithAttr("java", "21", attr)

		config, ok := tools.Get("java")
		if !ok {
			t.Fatal("Expected java to be found")
		}
		if config.Version != "21" {
			t.Errorf("Expected version 21, got %s", config.Version)
		}
		if len(config.Attr) != 3 {
			t.Errorf("Expected 3 attributes, got %d", len(config.Attr))
		}
		if config.Attr["vendor"] != "openjdk" {
			t.Errorf("Expected vendor openjdk, got %s", config.Attr["vendor"])
		}
		if config.Attr["dist"] != "temurin" {
			t.Errorf("Expected dist temurin, got %s", config.Attr["dist"])
		}
	})

	t.Run("List returns ToolVersions", func(t *testing.T) {
		tools := make(Tools)
		tools.Set("nodejs", "21.5.0")
		tools.Set("java", "17")

		versions := tools.List()
		if len(versions) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(versions))
		}
		if versions["nodejs"] != "21.5.0" {
			t.Errorf("Expected nodejs@21.5.0, got %s", versions["nodejs"])
		}
		if versions["java"] != "17" {
			t.Errorf("Expected java@17, got %s", versions["java"])
		}
	})

	t.Run("Remove tool", func(t *testing.T) {
		tools := make(Tools)
		tools.Set("nodejs", "21.5.0")
		tools.Set("java", "17")

		tools.Remove("nodejs")

		_, ok := tools.GetVersion("nodejs")
		if ok {
			t.Error("Expected nodejs to be removed")
		}

		version, ok := tools.GetVersion("java")
		if !ok || version != "17" {
			t.Error("Expected java to still exist")
		}
	})
}

func TestVfoxTomlAPI(t *testing.T) {
	t.Run("SetTool and GetToolVersion", func(t *testing.T) {
		config := NewVfoxToml()
		config.SetTool("nodejs", "21.5.0")

		version, ok := config.GetToolVersion("nodejs")
		if !ok {
			t.Fatal("Expected nodejs to be found")
		}
		if version != "21.5.0" {
			t.Errorf("Expected version 21.5.0, got %s", version)
		}
	})

	t.Run("SetToolWithAttr and GetToolConfig", func(t *testing.T) {
		config := NewVfoxToml()
		attr := Attr{
			"vendor": "openjdk",
			"dist":   "temurin",
		}
		config.SetToolWithAttr("java", "21", attr)

		toolConfig, ok := config.GetToolConfig("java")
		if !ok {
			t.Fatal("Expected java to be found")
		}
		if toolConfig.Version != "21" {
			t.Errorf("Expected version 21, got %s", toolConfig.Version)
		}
		if toolConfig.Attr["vendor"] != "openjdk" {
			t.Errorf("Expected vendor openjdk, got %s", toolConfig.Attr["vendor"])
		}
	})

	t.Run("GetAllTools returns ToolVersions", func(t *testing.T) {
		config := NewVfoxToml()
		config.SetTool("nodejs", "21.5.0")
		config.SetTool("java", "17")

		versions := config.GetAllTools()
		if len(versions) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(versions))
		}
		if versions["nodejs"] != "21.5.0" {
			t.Errorf("Expected nodejs@21.5.0, got %s", versions["nodejs"])
		}
	})

	t.Run("RemoveTool", func(t *testing.T) {
		config := NewVfoxToml()
		config.SetTool("nodejs", "21.5.0")
		config.SetTool("java", "17")

		config.RemoveTool("nodejs")

		_, ok := config.GetToolVersion("nodejs")
		if ok {
			t.Error("Expected nodejs to be removed")
		}

		version, ok := config.GetToolVersion("java")
		if !ok || version != "17" {
			t.Error("Expected java to still exist")
		}
	})
}

func TestVfoxTomlChainAPI(t *testing.T) {
	t.Run("GetToolVersion from chain", func(t *testing.T) {
		chain := NewVfoxTomlChain()

		globalConfig := NewVfoxToml()
		globalConfig.SetTool("nodejs", "20.0.0")

		projectConfig := NewVfoxToml()
		projectConfig.SetTool("nodejs", "21.5.0")

		chain.Add(globalConfig)
		chain.Add(projectConfig)

		// Should get from project (higher priority)
		version, ok := chain.GetToolVersion("nodejs")
		if !ok {
			t.Fatal("Expected nodejs to be found")
		}
		if version != "21.5.0" {
			t.Errorf("Expected version 21.5.0, got %s", version)
		}
	})

	t.Run("GetToolConfig from chain", func(t *testing.T) {
		chain := NewVfoxTomlChain()

		globalConfig := NewVfoxToml()
		globalConfig.SetTool("java", "17")

		projectConfig := NewVfoxToml()
		attr := Attr{"vendor": "openjdk"}
		projectConfig.SetToolWithAttr("java", "21", attr)

		chain.Add(globalConfig)
		chain.Add(projectConfig)

		// Should get from project (higher priority)
		config, ok := chain.GetToolConfig("java")
		if !ok {
			t.Fatal("Expected java to be found")
		}
		if config.Version != "21" {
			t.Errorf("Expected version 21, got %s", config.Version)
		}
		if config.Attr["vendor"] != "openjdk" {
			t.Errorf("Expected vendor openjdk, got %s", config.Attr["vendor"])
		}
	})

	t.Run("GetAllTools returns ToolVersions", func(t *testing.T) {
		chain := NewVfoxTomlChain()

		globalConfig := NewVfoxToml()
		globalConfig.SetTool("nodejs", "20.0.0")
		globalConfig.SetTool("java", "17")

		projectConfig := NewVfoxToml()
		projectConfig.SetTool("nodejs", "21.5.0")

		chain.Add(globalConfig)
		chain.Add(projectConfig)

		versions := chain.GetAllTools()
		if len(versions) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(versions))
		}
		// Project should override global
		if versions["nodejs"] != "21.5.0" {
			t.Errorf("Expected nodejs@21.5.0, got %s", versions["nodejs"])
		}
		if versions["java"] != "17" {
			t.Errorf("Expected java@17, got %s", versions["java"])
		}
	})
}

func TestAttrType(t *testing.T) {
	t.Run("create and use Attr", func(t *testing.T) {
		attr := Attr{
			"vendor": "openjdk",
			"dist":   "temurin",
		}

		if len(attr) != 2 {
			t.Errorf("Expected 2 attributes, got %d", len(attr))
		}

		if attr["vendor"] != "openjdk" {
			t.Errorf("Expected vendor openjdk, got %s", attr["vendor"])
		}
	})

	t.Run("nil Attr is empty", func(t *testing.T) {
		var attr Attr
		if attr != nil && len(attr) != 0 {
			t.Errorf("Expected nil attr to be empty, got %d items", len(attr))
		}
	})

	t.Run("Attr can be iterated", func(t *testing.T) {
		attr := Attr{
			"vendor": "openjdk",
			"dist":   "temurin",
			"arch":   "x64",
		}

		count := 0
		for k, v := range attr {
			if k == "" || v == "" {
				t.Errorf("Unexpected empty key or value: %s=%s", k, v)
			}
			count++
		}

		if count != 3 {
			t.Errorf("Expected 3 iterations, got %d", count)
		}
	})
}

func TestToolVersionsType(t *testing.T) {
	t.Run("create and use ToolVersions", func(t *testing.T) {
		versions := ToolVersions{
			"nodejs": "21.5.0",
			"java":   "21",
		}

		if len(versions) != 2 {
			t.Errorf("Expected 2 tools, got %d", len(versions))
		}

		if versions["nodejs"] != "21.5.0" {
			t.Errorf("Expected nodejs@21.5.0, got %s", versions["nodejs"])
		}
	})

	t.Run("ToolVersions can be iterated", func(t *testing.T) {
		versions := ToolVersions{
			"nodejs": "21.5.0",
			"java":   "21",
			"python": "3.12",
		}

		count := 0
		for name, version := range versions {
			if name == "" || version == "" {
				t.Errorf("Unexpected empty name or version: %s=%s", name, version)
			}
			count++
		}

		if count != 3 {
			t.Errorf("Expected 3 iterations, got %d", count)
		}
	})
}
